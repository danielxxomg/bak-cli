package restore

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/danielxxomg/bak-cli/internal/manifest"

	gitutil "github.com/danielxxomg/bak-cli/internal/git"
)

// Engine orchestrates the restore workflow: compute dry-run diffs,
// optionally apply them, and report results. When GitDir is set and
// points to a git repository, the engine auto-commits before and
// after applying changes for safety.
type Engine struct {
	HomeDir   string // target home directory
	BackupDir string // backup storage directory (contains manifest.json + adapter dirs)
	DryRun    bool   // if true, only compute diffs, don't write files
	Force     bool   // if true, skip confirmation prompts
	Verbose   bool   // enable verbose logging
	GitDir    string // optional path to git repo for safety commits (e.g., ~/.bak/)
}

// RestoreResult summarizes a restore operation.
type RestoreResult struct {
	ID       string     // backup ID from manifest
	Restored int        // files successfully restored
	Skipped  int        // files skipped (missing backup, unchanged)
	Failed   int        // files that could not be restored
	Diffs    []FileDiff // computed diff for all items
}

// Run executes the restore workflow: compute diffs, then apply unless
// dry-run mode is set. When a git repo is configured, auto-commits are
// created before and after applying changes for safety.
func (e *Engine) Run(m *manifest.Manifest) (*RestoreResult, error) {
	// Step 1: Compute dry-run diffs (mandatory gate).
	diffs, err := ComputeDryRun(m, e.BackupDir, e.HomeDir)
	if err != nil {
		return nil, fmt.Errorf("compute dry-run: %w", err)
	}

	result := &RestoreResult{
		ID:    m.ID,
		Diffs: diffs,
	}

	// Step 2: If dry-run only, return diffs without applying.
	if e.DryRun {
		return result, nil
	}

	// Step 3: Validate manifest checksums before applying (integrity check).
	// Fail on checksum mismatch unless --force is set.
	if err := m.Validate(e.BackupDir); err != nil {
		if !e.Force {
			return nil, fmt.Errorf("manifest validation failed (use --force to override): %w", err)
		}
		if e.Verbose {
			fmt.Fprintf(os.Stderr, "warning: manifest validation: %v\n", err)
		}
	}

	// Step 4: Git safety — commit current state before applying changes.
	e.tryAutoCommit("pre-restore snapshot")

	// Step 4: Apply restore for new and modified files.
	for _, d := range diffs {
		switch d.Status {
		case DiffNew, DiffModified:
			if err := e.restoreFile(d); err != nil {
				result.Failed++
				if e.Verbose {
					fmt.Fprintf(os.Stderr, "restore %s: %v\n", d.SourcePath, err)
				}
			} else {
				result.Restored++
			}
		case DiffUnchanged:
			result.Skipped++
		case DiffMissing:
			result.Skipped++
			if e.Verbose {
				fmt.Fprintf(os.Stderr, "warning: missing backup file %s\n", d.BackupPath)
			}
		}
	}

	// Step 5: Git safety — commit new state after applying.
	e.writeRestoreLog(m.ID, result)
	e.tryAutoCommit(fmt.Sprintf("restored: %s", m.ID))

	return result, nil
}

// tryAutoCommit attempts to create a safety commit in the configured
// git repository. It silently skips when no git repo is configured or
// the path is not a git repository, making git safety optional.
func (e *Engine) tryAutoCommit(action string) {
	if e.GitDir == "" {
		return
	}
	if !gitutil.IsRepo(e.GitDir) {
		return
	}

	repo, err := gitutil.OpenRepo(e.GitDir)
	if err != nil {
		if e.Verbose {
			fmt.Fprintf(os.Stderr, "git: open repo: %v\n", err)
		}
		return
	}

	if err := gitutil.StageAll(repo); err != nil {
		if e.Verbose {
			fmt.Fprintf(os.Stderr, "git: stage: %v\n", err)
		}
		return
	}

	if err := gitutil.Commit(repo, gitutil.AutoCommitMessage(action)); err != nil {
		if e.Verbose {
			fmt.Fprintf(os.Stderr, "git: commit: %v\n", err)
		}
		return
	}
}

// writeRestoreLog appends a restore entry to .bak/restore-log.jsonl
// in the backup directory so the post-restore commit has a meaningful
// change to capture.
func (e *Engine) writeRestoreLog(backupID string, result *RestoreResult) {
	if e.GitDir == "" {
		return
	}
	logPath := filepath.Join(e.BackupDir, "restore-log.jsonl")
	entry := fmt.Sprintf(`{"id":"%s","restored":%d,"skipped":%d,"failed":%d}`+"\n",
		backupID, result.Restored, result.Skipped, result.Failed)
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		if e.Verbose {
			fmt.Fprintf(os.Stderr, "restore-log: mkdir: %v\n", err)
		}
		return
	}
	if err := os.WriteFile(logPath, []byte(entry), 0644); err != nil {
		if e.Verbose {
			fmt.Fprintf(os.Stderr, "restore-log: write: %v\n", err)
		}
	}
}

// restoreFile copies a single file from the backup directory to the
// target path, creating parent directories as needed. Validates that
// both source and target paths stay within their expected directories.
func (e *Engine) restoreFile(d FileDiff) error {
	// Security: validate source path stays under backup directory.
	src := filepath.Join(e.BackupDir, d.BackupPath)
	cleanSrc := path.Clean(filepath.ToSlash(src))
	cleanBackupDir := path.Clean(filepath.ToSlash(e.BackupDir)) + "/"
	if !strings.HasPrefix(cleanSrc, cleanBackupDir) {
		return fmt.Errorf("source path %q escapes backup directory", d.BackupPath)
	}

	// Security: validate target path stays under home directory.
	cleanTarget := path.Clean(filepath.ToSlash(d.TargetPath))
	cleanHome := path.Clean(filepath.ToSlash(e.HomeDir)) + "/"
	if !strings.HasPrefix(cleanTarget, cleanHome) {
		return fmt.Errorf("target path %q escapes home directory", d.TargetPath)
	}

	// Ensure target parent directory exists.
	if err := os.MkdirAll(filepath.Dir(d.TargetPath), 0755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source: %w", err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(d.TargetPath)
	if err != nil {
		return fmt.Errorf("create target: %w", err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("copy: %w", err)
	}

	return nil
}
