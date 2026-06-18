package cmd

import (
	"bufio"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/danielxxomg/bak-cli/internal/actions"
	"github.com/danielxxomg/bak-cli/internal/backup"
)

var cleanupKeep int
var cleanupDryRun bool
var cleanupForce bool

// cleanupCmd represents the cleanup command.
var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Delete old backups, keeping the newest N",
	Long: `Remove old backup directories to free disk space. By default,
keeps the 3 newest backups. Use --dry-run to preview what would be
deleted before committing.

Destructive operation: requires --force to delete without confirmation.
Without --force, a confirmation prompt is shown on TTY.

Examples:
  bak cleanup --dry-run
  bak cleanup --keep 5 --dry-run
  bak cleanup --keep 3
  bak cleanup --keep 1 --force`,
	Args: cobra.NoArgs,
	RunE: runCleanup,
}

func init() {
	cleanupCmd.Flags().IntVar(&cleanupKeep, "keep", 3,
		"number of newest backups to keep")
	cleanupCmd.Flags().BoolVar(&cleanupDryRun, "dry-run", false,
		"show what would be deleted without making changes")
	cleanupCmd.Flags().BoolVar(&cleanupForce, "force", false,
		"delete without confirmation prompt")

	rootCmd.AddCommand(cleanupCmd)
}

func runCleanup(cmd *cobra.Command, args []string) error {
	return runCleanupWithDeps(cmd, args, depsFromCmd(cmd))
}

func runCleanupWithDeps(cmd *cobra.Command, args []string, deps cmdDeps) error {
	bakDir, err := backup.BakDir()
	if err != nil {
		return fmt.Errorf("bak dir: %w", err)
	}

	backupsDir := filepath.Join(bakDir, "backups")

	action := &actions.CleanupAction{
		FS:         &actions.OSFileSystem{},
		BackupsDir: backupsDir,
		Keep:       cleanupKeep,
		DryRun:     cleanupDryRun,
		Force:      cleanupForce,
		Stdout:     deps.Stdout,
		Stderr:     deps.Stderr,
	}

	// Wire TTY confirmation when not using --force.
	if !cleanupForce && isTTY() {
		action.Confirm = func() bool {
			_, _ = fmt.Fprintf(deps.Stdout, "Delete backups? [y/N]: ")
			reader := bufio.NewReader(deps.Stdin)
			response, _ := reader.ReadString('\n')
			response = strings.TrimSpace(strings.ToLower(response))
			return response == "y" || response == "yes"
		}
	}

	return action.Run()
}
