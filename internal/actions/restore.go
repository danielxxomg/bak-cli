package actions

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/danielxxomg/bak-cli/internal/manifest"
	restorepkg "github.com/danielxxomg/bak-cli/internal/restore"
	"github.com/spf13/cobra"
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
}

// Run executes the restore workflow: load manifest, compute diffs, and
// optionally apply changes.
func (a *RestoreAction) Run(cmd *cobra.Command, args []string) error {
	// Resolve output writers — fall back to os.Stdout/Stderr when cmd is nil.
	var out io.Writer = os.Stdout
	var errOut io.Writer = os.Stderr
	if cmd != nil {
		out = cmd.OutOrStdout()
		errOut = cmd.ErrOrStderr()
	}

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
	if len(diffs) > 0 {
		fmt.Fprintln(out, "Dry-run diff:")
		for _, d := range diffs {
			fmt.Fprintf(out, "  [%s] %s\n", d.Status, d.SourcePath)
			if d.Status == restorepkg.DiffModified && d.Diff != "" && a.Verbose {
				fmt.Fprint(out, d.Diff)
			}
		}
		fmt.Fprintln(out)
	}

	if a.DryRun {
		fmt.Fprintf(out, "Dry-run complete. %d file(s) would be restored, %d unchanged, %d missing.\n",
			countByStatus(diffs, restorepkg.DiffNew)+countByStatus(diffs, restorepkg.DiffModified),
			countByStatus(diffs, restorepkg.DiffUnchanged),
			countByStatus(diffs, restorepkg.DiffMissing),
		)
		return nil
	}

	// 5. Validate manifest checksums before applying.
	if err := m.Validate(a.BackupDir); err != nil {
		if !a.Force {
			return fmt.Errorf("manifest validation failed (use --force to override): %w", err)
		}
		if a.Verbose {
			fmt.Fprintf(errOut, "warning: manifest validation: %v\n", err)
		}
	}

	// 6. Confirmation prompt (unless --force).
	if !a.Force {
		fmt.Fprint(out, "Apply restore? [y/N]: ")
		reader := bufio.NewReader(os.Stdin)
		answer, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("read input: %w", err)
		}
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "y" && answer != "yes" {
			fmt.Fprintln(errOut, "Restore cancelled.")
			return nil
		}
	}

	// 7. Apply restore.
	restored, skipped, failed := 0, 0, 0
	for _, d := range diffs {
		switch d.Status {
		case restorepkg.DiffNew, restorepkg.DiffModified:
			if err := a.restoreFile(d); err != nil {
				failed++
				if a.Verbose {
					fmt.Fprintf(errOut, "restore %s: %v\n", d.SourcePath, err)
				}
			} else {
				restored++
			}
		case restorepkg.DiffUnchanged:
			skipped++
		case restorepkg.DiffMissing:
			skipped++
			if a.Verbose {
				fmt.Fprintf(errOut, "warning: missing backup file %s\n", d.BackupPath)
			}
		}
	}

	// 8. Report results.
	fmt.Fprintf(out, "Restore complete: %s\n", m.ID)
	fmt.Fprintf(out, "  Restored: %d\n", restored)
	fmt.Fprintf(out, "  Skipped:  %d\n", skipped)
	if failed > 0 {
		fmt.Fprintf(out, "  Failed:   %d\n", failed)
	}

	return nil
}

// restoreFile copies a single file from the backup directory to the
// target path, creating parent directories as needed. Validates path
// traversal safety.
func (a *RestoreAction) restoreFile(d restorepkg.FileDiff) error {
	src := filepath.Join(a.BackupDir, d.BackupPath)

	// Security: validate source path stays under backup directory.
	cleanSrc := path.Clean(filepath.ToSlash(src))
	cleanBackupDir := path.Clean(filepath.ToSlash(a.BackupDir)) + "/"
	if !strings.HasPrefix(cleanSrc, cleanBackupDir) {
		return fmt.Errorf("source path escapes backup directory")
	}

	// Security: validate target path stays under home directory.
	homeDir, err := a.FS.UserHomeDir()
	if err != nil {
		return fmt.Errorf("home dir: %w", err)
	}
	cleanTarget := path.Clean(filepath.ToSlash(d.TargetPath))
	cleanHome := path.Clean(filepath.ToSlash(homeDir)) + "/"
	if !strings.HasPrefix(cleanTarget, cleanHome) {
		return fmt.Errorf("target path escapes home directory")
	}

	// Ensure target parent directory exists.
	if err := a.FS.MkdirAll(filepath.Dir(d.TargetPath), 0755); err != nil {
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
