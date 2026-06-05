package cmd

import (
	"fmt"
	"os"
	"path/filepath"

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
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("cannot determine home directory: %w", err)
	}

	bakDir := filepath.Join(homeDir, ".bak")

	if !gitutil.IsRepo(bakDir) {
		return fmt.Errorf("no bak repository found at %s — run 'bak backup' first", bakDir)
	}

	repo, err := gitutil.OpenRepo(bakDir)
	if err != nil {
		return fmt.Errorf("open bak repository: %w", err)
	}

	if err := gitutil.Undo(repo); err != nil {
		return fmt.Errorf("undo failed: %w", err)
	}

	fmt.Println("✅ Reverted to previous state")
	return nil
}
