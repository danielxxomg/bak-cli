package actions

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/danielxxomg/bak-cli/internal/backup"
	"github.com/danielxxomg/bak-cli/internal/manifest"
	"github.com/danielxxomg/bak-cli/internal/paths"
	restorepkg "github.com/danielxxomg/bak-cli/internal/restore"
)

// RestoreAction encapsulates the restore workflow with injectable
// dependencies. All OS operations that are the action's responsibility
// go through a.FS.
type RestoreAction struct {
	FS        FileSystem
	BackupDir string // resolved backup directory
	DryRun    bool
	Force     bool
	Verbose   bool
	GitDir    string // optional git repo for safety commits

	// ProgressFn is an optional callback invoked once per file during restore.
	// When nil (default), no progress is reported.
	ProgressFn func(currentFile string, filesDone int, filesTotal int)

	// Stdin is the reader for confirmation prompts. Nil falls back to os.Stdin.
	Stdin io.Reader
	// Stdout receives informational output. Nil falls back to os.Stdout.
	Stdout io.Writer
	// Stderr receives warnings and error diagnostics. Nil falls back to os.Stderr.
	Stderr io.Writer
}

// ResolveBackup resolves the backup ID to a directory path and sets a.BackupDir.
func (a *RestoreAction) ResolveBackup(backupID string) error {
	dir, err := backup.ResolveBackupID(backupID)
	if err != nil {
		return fmt.Errorf("resolve backup %q: %w", backupID, err)
	}
	a.BackupDir = dir
	return nil
}

// Run executes the restore workflow: load manifest, compute diffs, and
// optionally apply changes. Each phase is delegated to a helper to keep the
// orchestration readable and below the cognitive-complexity threshold.
func (a *RestoreAction) Run() error {
	out, errOut := a.resolveWriters()

	// 1. Load manifest.
	m, err := manifest.Load(a.BackupDir)
	if err != nil {
		return fmt.Errorf("load manifest: %w", err)
	}

	// 2. Get home directory.
	homeDir, err := a.FS.UserHomeDir()
	if err != nil {
		return fmt.Errorf("cannot determine home directory: %w", err)
	}

	// 3. Compute dry-run diffs.
	diffs, err := restorepkg.ComputeDryRun(m, a.BackupDir, homeDir)
	if err != nil {
		return fmt.Errorf("compute dry-run: %w", err)
	}

	// 4. Show diffs.
	a.printDryRunDiff(out, diffs)

	if a.DryRun {
		_, _ = fmt.Fprintf(out, "Dry-run complete. %d file(s) would be restored, %d unchanged, %d missing.\n",
			countByStatus(diffs, restorepkg.DiffNew)+countByStatus(diffs, restorepkg.DiffModified),
			countByStatus(diffs, restorepkg.DiffUnchanged),
			countByStatus(diffs, restorepkg.DiffMissing),
		)
		return nil
	}

	// 5. Validate manifest checksums before applying.
	if err := a.validateManifest(m, errOut); err != nil {
		return err
	}

	// 6. Confirmation prompt (unless --force).
	proceed, err := a.confirmRestore(out, errOut)
	if err != nil {
		return err
	}
	if !proceed {
		return nil
	}

	// 7. Apply restore.
	restored, skipped, failed := a.applyRestore(diffs, out, errOut)

	// 8. Report results.
	reportRestore(out, m, restored, skipped, failed)

	return nil
}

// resolveWriters returns the output and error writers, falling back to
// os.Stdout / os.Stderr when the action's fields are nil.
func (a *RestoreAction) resolveWriters() (io.Writer, io.Writer) {
	out := a.Stdout
	if out == nil {
		out = os.Stdout
	}
	errOut := a.Stderr
	if errOut == nil {
		errOut = os.Stderr
	}
	return out, errOut
}

// printDryRunDiff writes the per-file dry-run diff to out. When verbose and
// a file is modified with a non-empty diff, the unified diff is appended.
func (a *RestoreAction) printDryRunDiff(out io.Writer, diffs []restorepkg.FileDiff) {
	if len(diffs) == 0 {
		return
	}
	_, _ = fmt.Fprintln(out, "Dry-run diff:")
	for _, d := range diffs {
		_, _ = fmt.Fprintf(out, "  [%s] %s\n", d.Status, d.SourcePath)
		if d.Status == restorepkg.DiffModified && d.Diff != "" && a.Verbose {
			_, _ = fmt.Fprint(out, d.Diff)
		}
	}
	_, _ = fmt.Fprintln(out)
}

// validateManifest validates checksums; with --force a validation failure is
// downgraded to a verbose warning, otherwise it is a hard error.
func (a *RestoreAction) validateManifest(m *manifest.Manifest, errOut io.Writer) error {
	if err := m.Validate(a.BackupDir, nil); err != nil {
		if !a.Force {
			return fmt.Errorf("manifest validation failed (use --force to override): %w", err)
		}
		if a.Verbose {
			_, _ = fmt.Fprintf(errOut, "warning: manifest validation: %v\n", err)
		}
	}
	return nil
}

// confirmRestore prompts the user (unless --force) and reports whether the
// restore should proceed. A "y"/"yes" answer (or --force) returns true; any
// other answer prints "Restore cancelled." and returns false. A read error
// is propagated.
func (a *RestoreAction) confirmRestore(out, errOut io.Writer) (bool, error) {
	if a.Force {
		return true, nil
	}
	_, _ = fmt.Fprint(out, "Apply restore? [y/N]: ")
	stdin := a.Stdin
	if stdin == nil {
		stdin = os.Stdin
	}
	reader := bufio.NewReader(stdin)
	answer, err := reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("read input: %w", err)
	}
	answer = strings.TrimSpace(strings.ToLower(answer))
	if answer != "y" && answer != "yes" {
		_, _ = fmt.Fprintln(errOut, "Restore cancelled.")
		return false, nil
	}
	return true, nil
}

// applyRestore copies each new/modified file, skipping unchanged and missing
// files. Progress is reported via ProgressFn when set. Returns the counts of
// restored, skipped, and failed files.
func (a *RestoreAction) applyRestore(diffs []restorepkg.FileDiff, out, errOut io.Writer) (restored, skipped, failed int) {
	filesTotal := 0
	for _, d := range diffs {
		if d.Status == restorepkg.DiffNew || d.Status == restorepkg.DiffModified {
			filesTotal++
		}
	}
	filesDone := 0

	for _, d := range diffs {
		switch d.Status {
		case restorepkg.DiffNew, restorepkg.DiffModified:
			filesDone++
			if a.ProgressFn != nil {
				a.ProgressFn(d.SourcePath, filesDone, filesTotal)
			}
			if err := a.restoreFile(d); err != nil {
				failed++
				if a.Verbose {
					_, _ = fmt.Fprintf(errOut, "restore %s: %v\n", d.SourcePath, err)
				}
			} else {
				restored++
			}
		case restorepkg.DiffUnchanged:
			skipped++
		case restorepkg.DiffMissing:
			skipped++
			if a.Verbose {
				_, _ = fmt.Fprintf(errOut, "warning: missing backup file %s\n", d.BackupPath)
			}
		}
	}
	return restored, skipped, failed
}

// reportRestore writes the final restore summary to out, including the failed
// count only when at least one file failed.
func reportRestore(out io.Writer, m *manifest.Manifest, restored, skipped, failed int) {
	_, _ = fmt.Fprintf(out, "Restore complete: %s\n", m.ID)
	_, _ = fmt.Fprintf(out, "  Restored: %d\n", restored)
	_, _ = fmt.Fprintf(out, "  Skipped:  %d\n", skipped)
	if failed > 0 {
		_, _ = fmt.Fprintf(out, "  Failed:   %d\n", failed)
	}
}

// restoreFile copies a single file from the backup directory to the
// target path, creating parent directories as needed. Validates path
// traversal safety.
func (a *RestoreAction) restoreFile(d restorepkg.FileDiff) error {
	src := filepath.Join(a.BackupDir, d.BackupPath)

	// Security: validate source path stays under backup directory.
	cleanSrc := paths.CanonicalPath(src)
	cleanBackupDir := paths.CanonicalPath(a.BackupDir) + "/"
	if !strings.HasPrefix(cleanSrc, cleanBackupDir) {
		return fmt.Errorf("source path escapes backup directory")
	}

	// Security: validate target path stays under home directory.
	homeDir, err := a.FS.UserHomeDir()
	if err != nil {
		return fmt.Errorf("home dir: %w", err)
	}
	cleanTarget := paths.CanonicalPath(d.TargetPath)
	cleanHome := paths.CanonicalPath(homeDir) + "/"
	if !strings.HasPrefix(cleanTarget, cleanHome) {
		return fmt.Errorf("target path escapes home directory")
	}

	// Ensure target parent directory exists.
	if err := a.FS.MkdirAll(filepath.Dir(d.TargetPath), 0755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	if err := a.FS.CopyFile(src, d.TargetPath); err != nil {
		return fmt.Errorf("copy: %w", err)
	}

	return nil
}

// countByStatus returns the number of diffs with the given status.
func countByStatus(diffs []restorepkg.FileDiff, status restorepkg.DiffStatus) int {
	count := 0
	for _, d := range diffs {
		if d.Status == status {
			count++
		}
	}
	return count
}
