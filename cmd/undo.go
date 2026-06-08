package cmd

import (
	"github.com/danielxxomg/bak-cli/internal/actions"
	gitutil "github.com/danielxxomg/bak-cli/internal/git"
	"github.com/spf13/cobra"
)

// undoCmd represents the undo command.
var undoCmd = &cobra.Command{
	Use:   "undo",
	Short: "Revert the last bak operation",
	Long: `Reverts the last operation (restore or backup) by creating a new
revert commit. This is equivalent to 'git revert HEAD' on the bak
storage directory (~/.bak/).

The undo is safe and non-destructive — it does NOT rewrite history
or force-push. You can undo the undo by running 'bak undo' again.

Examples:
  bak undo          Revert the last restore
  bak undo --verbose Show details`,
	Args: cobra.NoArgs,
	RunE: runUndo,
}

func init() {
	rootCmd.AddCommand(undoCmd)
}

func runUndo(cmd *cobra.Command, args []string) error {
	return runUndoWithDeps(cmd, args, depsFromCmd(cmd))
}

func runUndoWithDeps(cmd *cobra.Command, args []string, deps cmdDeps) error {
	action := &actions.UndoAction{
		Stdout: deps.Stdout,
		IsRepo: gitutil.IsRepo,
		UndoFn: func(repoPath string) error {
			repo, err := gitutil.OpenRepo(repoPath)
			if err != nil {
				return err
			}
			return gitutil.Undo(repo)
		},
	}
	return action.Run()
}
