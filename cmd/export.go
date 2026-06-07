package cmd

import (
	"fmt"
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
	return runExportWithDeps(cmd, args, depsFromCmd(cmd))
}

func runExportWithDeps(cmd *cobra.Command, args []string, deps cmdDeps) error {
	backupID := args[0]

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get home directory: %w", err)
	}

	return actions.RunExport(homeDir, backupID, exportOutput, deps.Stdout)
}
