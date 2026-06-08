package actions

import (
	"fmt"
	"io"

	"github.com/danielxxomg/bak-cli/internal/backup"
	"github.com/danielxxomg/bak-cli/internal/manifest"
)

// VerifyBackupAction verifies a backup's integrity by checking SHA-256
// hashes of every file in the manifest against the actual files on disk.
type VerifyBackupAction struct {
	// Stdout is the writer for success output.
	Stdout io.Writer

	// Stderr is the writer for verbose progress output.
	Stderr io.Writer

	// Verbose enables per-file progress output.
	Verbose bool
}

// Run resolves the backup, loads its manifest, and validates all file
// checksums. Returns nil on success or the first hash mismatch error.
func (a *VerifyBackupAction) Run(backupID string) error {
	backupDir, err := backup.ResolveBackupID(backupID)
	if err != nil {
		return fmt.Errorf("resolve backup %q: %w", backupID, err)
	}

	m, err := manifest.Load(backupDir)
	if err != nil {
		return fmt.Errorf("load manifest: %w", err)
	}

	if a.Verbose {
		_, _ = fmt.Fprintf(a.Stderr, "Verifying %d files in backup %q...\n", m.FileCount, backupID)
	}

	var progressFn func(string)
	if a.Verbose {
		progressFn = func(path string) {
			_, _ = fmt.Fprintf(a.Stderr, "  verifying %s\n", path)
		}
	}

	if err := m.Validate(backupDir, progressFn); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(a.Stdout, "✓ backup %q verified (%d files)\n", backupID, m.FileCount)
	return nil
}
