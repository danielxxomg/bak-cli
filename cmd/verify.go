package cmd

import (
	"fmt"

	"github.com/danielxxomg/bak-cli/internal/backup"
	"github.com/danielxxomg/bak-cli/internal/manifest"
	"github.com/spf13/cobra"
)

// verifyCmd represents the verify command.
var verifyCmd = &cobra.Command{
	Use:   "verify <backup-id>",
	Short: "Verify a backup's integrity",
	Long: `Verifies the integrity of a backup by checking SHA-256 hashes
of every file in the manifest against the actual files on disk.

Exits 0 if all files pass, exits 1 on the first hash mismatch.

Example:
  bak verify 20260604-150405
  bak verify --verbose 20260604-150405`,
	Args: cobra.ExactArgs(1),
	RunE: runVerify,
}

func init() {
	rootCmd.AddCommand(verifyCmd)
}

func runVerify(cmd *cobra.Command, args []string) error {
	backupID := args[0]

	backupDir, err := backup.ResolveBackupID(backupID)
	if err != nil {
		return fmt.Errorf("resolve backup %q: %w", backupID, err)
	}

	m, err := manifest.Load(backupDir)
	if err != nil {
		return fmt.Errorf("load manifest: %w", err)
	}

	if verbose {
		fmt.Fprintf(cmd.ErrOrStderr(), "Verifying %d files in backup %q...\n", m.FileCount, backupID)
	}

	var progressFn func(string)
	if verbose {
		progressFn = func(path string) {
			fmt.Fprintf(cmd.ErrOrStderr(), "  verifying %s\n", path)
		}
	}

	if err := m.Validate(backupDir, progressFn); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "✓ backup %q verified (%d files)\n", backupID, m.FileCount)
	return nil
}
