package cmd

import (
	"fmt"
	"os"

	"github.com/danielxxomg/bak-cli/internal/actions"
	"github.com/danielxxomg/bak-cli/internal/backup"
	"github.com/spf13/cobra"
)

var listProvider string

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all local backups",
	Long: `Scans ~/.bak/backups/ and displays a table of all local backups
with their ID, date, preset, file count, and size.

Use 'bak restore <id>' to restore a backup.
Use 'bak push <id>' to upload a backup to GitHub Gist.

When --provider is set, lists backups from the specified cloud backend
instead of local backups.

Examples:
  bak list
  bak list --provider github-gist`,
	RunE: runList,
}

func init() {
	listCmd.Flags().StringVar(&listProvider, "provider", "",
		"list backups from a cloud provider instead of local (github-gist, codeberg, gitea, github-repo, rclone)")
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
	return runListWithDeps(cmd, args, depsFromCmd(cmd))
}

func runListWithDeps(cmd *cobra.Command, args []string, deps cmdDeps) error {
	// When --provider is set, list from cloud backend.
	if listProvider != "" {
		return runListCloudWithDeps(listProvider, deps)
	}

	// Local listing (existing behavior).
	return runListLocal(cmd)
}

func runListLocal(cmd *cobra.Command) error {
	bakDir, err := backup.BakDir()
	if err != nil {
		return fmt.Errorf("bak dir: %w", err)
	}

	return actions.RunListLocal(bakDir, verbose, cmd.OutOrStdout(), os.Stderr)
}

// runListCloud lists backups from a named cloud provider.
func runListCloud(providerName string) error {
	return runListCloudWithDeps(providerName, defaultDeps)
}

func runListCloudWithDeps(providerName string, deps cmdDeps) error {
	cfg, err := deps.ConfigLoader()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	action := &actions.ListCloudAction{
		Config:  cfg,
		Stdout:  deps.Stdout,
		Stderr:  deps.Stderr,
		Verbose: verbose,
	}

	return action.Run(providerName)
}

// formatSizeBytes delegates to actions.formatSizeBytes for backward compatibility.
func formatSizeBytes(bytes int64) string {
	return actions.FormatSizeBytes(bytes)
}
