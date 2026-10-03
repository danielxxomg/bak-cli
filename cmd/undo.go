package cmd

import (
	"github.com/spf13/cobra"

	"github.com/danielxxomg/bak-cli/internal/actions"
	gitutil "github.com/danielxxomg/bak-cli/internal/git"
)

// undoCmd represents the undo command.
var undoCmd = &cobra.Command{
	Use:   "undo",
	Short: "Revert the last bak operation",
	Long: `Reverts the last restore operation by restoring target configuration
files to their pre-restore state and creating a revert commit in the bak
storage directory (~/.bak/).

Undo includes fail-closed drift protection: if any target file was modified,
deleted, created, or replaced since the restore was applied, undo refuses all
changes before writing any file.

The undo is safe, non-destructive, and history-preserving — it does NOT rewrite
history or force-push.

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
		FS:     &actions.OSFileSystem{},
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
