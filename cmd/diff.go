package cmd

import (
	"github.com/spf13/cobra"

	"github.com/danielxxomg/bak-cli/internal/actions"
)

// diffCmd represents the diff command.
var diffCmd = &cobra.Command{
	Use:   "diff <id1> <id2>",
	Short: "Show differences between two backups",
	Long: `Compares two backups and shows file-level differences grouped by
category: Added, Removed, Modified, and Unchanged.

Always exits 0 on success, even when differences are found.

Examples:
  bak diff 20260604-150405 20260605-080000`,
	Args: cobra.ExactArgs(2),
	RunE: runDiff,
}

func init() {
	rootCmd.AddCommand(diffCmd)
}

func runDiff(cmd *cobra.Command, args []string) error {
	return runDiffWithDeps(cmd, args, depsFromCmd(cmd))
}

func runDiffWithDeps(cmd *cobra.Command, args []string, deps cmdDeps) error {
	action := &actions.DiffBackupsAction{
		Stdout: deps.Stdout,
	}
	return action.Run(args[0], args[1])
}
