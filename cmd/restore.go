package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/danielxxomg/bak-cli/internal/manifest"
	"github.com/danielxxomg/bak-cli/internal/restore"
	"github.com/spf13/cobra"
)

var restoreDryRun bool
var restoreForce bool

// restoreCmd represents the restore command.
var restoreCmd = &cobra.Command{
	Use:   "restore [--dry-run] [--force] <backup-id>",
	Short: "Restore a backup to your system",
	Long: `Restores a previously created backup by copying files back to their
original locations. A dry-run diff is always shown before any files
are modified.

Without --dry-run, you will be prompted to confirm before applying.
Use --force to skip the confirmation (useful for scripting).

Examples:
  bak restore 20260604-232200 --dry-run
  bak restore 20260604-232200
  bak restore 20260604-232200 --force`,
	Args: cobra.ExactArgs(1),
	RunE: runRestore,
}

func init() {
	restoreCmd.Flags().BoolVar(&restoreDryRun, "dry-run", false,
		"show what would change without applying")
	restoreCmd.Flags().BoolVar(&restoreForce, "force", false,
		"skip confirmation prompt")

	rootCmd.AddCommand(restoreCmd)
}

func runRestore(cmd *cobra.Command, args []string) error {
	backupID := args[0]

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("cannot determine home directory: %w", err)
	}

	// Locate the backup.
	bakDir := filepath.Join(homeDir, ".bak")
	backupDir := filepath.Join(bakDir, "backups", backupID)

	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		return fmt.Errorf("backup %q not found at %s", backupID, backupDir)
	}

	// Load manifest.
	m, err := manifest.Load(backupDir)
	if err != nil {
		return fmt.Errorf("load manifest: %w", err)
	}

	// Build and run engine.
	engine := &restore.Engine{
		HomeDir:   homeDir,
		BackupDir: backupDir,
		DryRun:    restoreDryRun,
		Force:     restoreForce,
		Verbose:   verbose,
	}

	result, err := engine.Run(m)
	if err != nil {
		return err
	}

	// Show diffs.
	if len(result.Diffs) > 0 {
		fmt.Println("Dry-run diff:")
		for _, d := range result.Diffs {
			fmt.Printf("  [%s] %s\n", d.Status, d.SourcePath)
			if d.Status == restore.DiffModified && d.Diff != "" && verbose {
				fmt.Print(d.Diff)
			}
		}
		fmt.Println()
	}

	if restoreDryRun {
		fmt.Printf("Dry-run complete. %d file(s) would be restored, %d unchanged, %d missing.\n",
			countByStatus(result.Diffs, restore.DiffNew)+countByStatus(result.Diffs, restore.DiffModified),
			countByStatus(result.Diffs, restore.DiffUnchanged),
			countByStatus(result.Diffs, restore.DiffMissing),
		)
		return nil
	}

	// Report results.
	fmt.Printf("Restore complete: %s\n", result.ID)
	fmt.Printf("  Restored: %d\n", result.Restored)
	fmt.Printf("  Skipped:  %d\n", result.Skipped)
	if result.Failed > 0 {
		fmt.Printf("  Failed:   %d\n", result.Failed)
	}

	return nil
}

// countByStatus returns the number of diffs with the given status.
func countByStatus(diffs []restore.FileDiff, status restore.DiffStatus) int {
	count := 0
	for _, d := range diffs {
		if d.Status == status {
			count++
		}
	}
	return count
}
