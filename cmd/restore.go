package cmd

import (
	"fmt"
	"os"

	"github.com/danielxxomg/bak-cli/internal/actions"
	"github.com/danielxxomg/bak-cli/internal/backup"
	"github.com/danielxxomg/bak-cli/internal/presets"
	restorepkg "github.com/danielxxomg/bak-cli/internal/restore"
	"github.com/spf13/cobra"
)

var restoreDryRun bool
var restoreForce bool
var restoreOverride bool

// restoreCmd represents the restore command.
var restoreCmd = &cobra.Command{
	Use:   "restore [--dry-run] [--force] [--override] <backup-id>",
	Short: "Restore a backup to your system",
	Long: `Restores a previously created backup by copying files back to their
original locations. A dry-run diff is always shown before any files
are modified.

Without --dry-run, you will be prompted to confirm before applying.
Use --force to skip the confirmation (useful for scripting).

Examples:
  bak restore 20260604-232200 --dry-run
  bak restore 20260604-232200
  bak restore 20260604-232200 --force
  bak restore 20260604-232200 --override`,
	Args: cobra.ExactArgs(1),
	RunE: runRestore,
}

func init() {
	restoreCmd.Flags().BoolVar(&restoreDryRun, "dry-run", false,
		"show what would change without applying")
	restoreCmd.Flags().BoolVar(&restoreForce, "force", false,
		"skip confirmation prompt")
	restoreCmd.Flags().BoolVar(&restoreOverride, "override", false,
		"prefer custom YAML presets and adapters over built-ins")

	rootCmd.AddCommand(restoreCmd)
}

func runRestore(cmd *cobra.Command, args []string) error {
	backupID := args[0]

	// Validate backup ID format early (UX guard, action also validates).
	if !isValidBackupID(backupID) {
		return fmt.Errorf("%s", formatBackupIDError(backupID))
	}

	// Resolve backup directory.
	backupDir, err := backup.ResolveBackupID(backupID)
	if err != nil {
		return fmt.Errorf("resolve backup %q: %w", backupID, err)
	}

	// Check for YAML presets override warning (informational).
	if restoreOverride && verbose {
		if _, err := presets.ResolveAll("quick", true); err != nil {
			fmt.Fprintf(os.Stderr, "warning: custom preset resolution: %v\n", err)
		}
	}

	// Build and run action.
	action := &actions.RestoreAction{
		FS:        &actions.OSFileSystem{},
		BackupDir: backupDir,
		DryRun:    restoreDryRun,
		Force:     restoreForce,
		Verbose:   verbose,
	}

	return action.Run(cmd, args)
}

// countByStatus returns the number of diffs with the given status.
// Kept as a package-level helper for testability; RestoreAction
// handles reporting internally.
func countByStatus(diffs []restorepkg.FileDiff, status restorepkg.DiffStatus) int {
	count := 0
	for _, d := range diffs {
		if d.Status == status {
			count++
		}
	}
	return count
}
