package restore

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/danielxxomg/bak-cli/internal/manifest"
)

// Engine orchestrates the restore workflow: compute dry-run diffs,
// optionally apply them, and report results.
type Engine struct {
	HomeDir   string // target home directory
	BackupDir string // backup storage directory (contains manifest.json + adapter dirs)
	DryRun    bool   // if true, only compute diffs, don't write files
	Force     bool   // if true, skip confirmation prompts
	Verbose   bool   // enable verbose logging
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
// dry-run mode is set. Returns a summary.
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

	// Step 3: Apply restore for new and modified files.
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

	return result, nil
}

// restoreFile copies a single file from the backup directory to the
// target path, creating parent directories as needed.
func (e *Engine) restoreFile(d FileDiff) error {
	src := filepath.Join(e.BackupDir, d.BackupPath)

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
