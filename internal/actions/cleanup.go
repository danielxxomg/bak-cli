// Package actions provides business logic for bak-cli CLI commands.
package actions

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/danielxxomg/bak-cli/internal/backup"
)

// CleanupAction performs backup retention cleanup. It lists backup
// directories, keeps the newest N, and deletes the rest.
type CleanupAction struct {
	FS         FileSystem
	BackupsDir string
	Keep       int
	DryRun     bool
	Force      bool

	// Confirm is a function that prompts for confirmation. When nil
	// and !Force, the action errors with a helpful message about --force.
	Confirm func() bool

	Stdout io.Writer
	Stderr io.Writer
}

// Run executes the cleanup action.
func (a *CleanupAction) Run() error {
	if a.FS == nil {
		a.FS = &OSFileSystem{}
	}
	if a.Keep <= 0 {
		a.Keep = 3 // default
	}

	entries, err := a.FS.ReadDir(a.BackupsDir)
	if err != nil {
		if os.IsNotExist(err) {
			_, _ = fmt.Fprintln(a.Stdout, "No backups directory found — nothing to clean.")
			return nil
		}
		return fmt.Errorf("read backups dir: %w", err)
	}

	// Backup IDs sorted descending (lexicographic == chronological) via the
	// canonical resolver core; the ReadDir stays on the injected FileSystem.
	ids := backup.SortedBackupIDs(entries)

	if len(ids) == 0 {
		_, _ = fmt.Fprintln(a.Stdout, "No backups found — nothing to clean.")
		return nil
	}

	keep := a.Keep
	if keep > len(ids) {
		keep = len(ids)
	}
	toDelete := ids[keep:]

	if len(toDelete) == 0 {
		_, _ = fmt.Fprintf(a.Stdout, "Nothing to clean (%d backups, keeping %d).\n", len(ids), keep)
		return nil
	}

	// Dry-run: print plan only.
	if a.DryRun {
		printDryRunPlan(a.Stdout, toDelete, keep)
		return nil
	}

	// Confirmation gate.
	if !a.Force {
		if a.Confirm == nil {
			return fmt.Errorf("cleanup requires --force or a TTY (use --dry-run to preview)")
		}
		if !a.Confirm() {
			_, _ = fmt.Fprintln(a.Stdout, "Cleanup cancelled.")
			return nil
		}
	}

	// Delete.
	var failed int
	for _, id := range toDelete {
		path := filepath.Join(a.BackupsDir, id)
		if err := a.FS.RemoveAll(path); err != nil {
			_, _ = fmt.Fprintf(a.Stderr, "failed to delete %s: %v\n", id, err)
			failed++
			continue
		}
		_, _ = fmt.Fprintf(a.Stderr, "deleted %s\n", id)
	}

	deleted := len(toDelete) - failed
	_, _ = fmt.Fprintf(a.Stdout, "Deleted %d/%d backups (%d failed).\n", deleted, len(toDelete), failed)
	return nil
}

// printDryRunPlan writes a dry-run deletion plan to out: a header line
// naming how many backups would be deleted and how many kept, followed by
// each backup ID to be deleted (indented).
func printDryRunPlan(out io.Writer, toDelete []string, keep int) {
	_, _ = fmt.Fprintf(out, "Would delete %d backups (keeping %d newest):\n", len(toDelete), keep)
	for _, id := range toDelete {
		_, _ = fmt.Fprintln(out, "  "+id)
	}
}
