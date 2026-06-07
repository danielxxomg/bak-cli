package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/danielxxomg/bak-cli/internal/actions"
	"github.com/spf13/cobra"
)

var exportOutput string

var exportCmd = &cobra.Command{
	Use:   "export <backup-id>",
	Short: "Export a backup as a tar.gz archive",
	Long: `Creates a compressed tar.gz archive of the specified backup.

The archive includes the manifest and all backed-up files, preserving
directory structure. This is useful for sharing backups or archiving
them outside of the bak storage directory.

Examples:
  bak export 20260604-150405
  bak export 20260604-150405 --output ./my-backup.tar.gz`,
	Args: cobra.ExactArgs(1),
	RunE: runExport,
}

func init() {
	exportCmd.Flags().StringVarP(&exportOutput, "output", "o", "bak-export.tar.gz",
		"output file path (default: ./bak-export.tar.gz)")
	rootCmd.AddCommand(exportCmd)
}

func runExport(cmd *cobra.Command, args []string) error {
	backupID := args[0]

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	return actions.RunExport(homeDir, backupID, exportOutput, cmd.OutOrStdout())
}

// createTarGz delegates to the actions package for test backward compatibility.
func createTarGz(srcDir string, w io.Writer) error {
	return actions.CreateTarGz(srcDir, w)
}

// isValidBackupID checks the format YYYYMMDD-HHMMSS.
func isValidBackupID(id string) bool {
	if len(id) != 15 || id[8] != '-' {
		return false
	}
	for i, c := range id {
		if i == 8 {
			continue
		}
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// formatBackupIDError returns a user-friendly error for invalid IDs.
func formatBackupIDError(id string) string {
	return fmt.Sprintf("invalid backup ID %q (expected format: YYYYMMDD-HHMMSS, e.g. 20260604-150405)", id)
}