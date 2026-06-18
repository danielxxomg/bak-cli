package cmd

import (
	"errors"
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"

	"github.com/danielxxomg/bak-cli/internal/actions"
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
	Args: cobra.MaximumNArgs(1),
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
	return runRestoreWithDeps(cmd, args, depsFromCmd(cmd))
}

func runRestoreWithDeps(cmd *cobra.Command, args []string, deps cmdDeps) error {
	// No-arg: launch interactive picker (TTY) or error (non-TTY).
	if len(args) == 0 {
		if !isTTY() {
			return fmt.Errorf("specify a backup-id (see 'bak list') or run 'bak' for interactive mode")
		}

		// List backups for the picker.
		backups, err := listBackups()
		if err != nil {
			return fmt.Errorf("list backups: %w", err)
		}

		if len(backups) == 0 {
			return fmt.Errorf("no backups found — create one with 'bak backup' first")
		}

		// Launch interactive picker.
		m := restorePickerModel{backups: backups}
		p := tea.NewProgram(m)
		result, runErr := p.Run()
		if runErr != nil {
			return fmt.Errorf("picker: %w", runErr)
		}

		model, ok := result.(restorePickerModel)
		if !ok {
			return fmt.Errorf("picker: unexpected model type %T", result)
		}

		selectedID := model.SelectedID()
		if selectedID == "" {
			_, _ = fmt.Fprintln(deps.Stdout, "Restore cancelled.")
			return nil
		}

		args = []string{selectedID}
	}

	backupID := args[0]

	// Validate backup ID format early (UX guard, action also validates).
	if !actions.IsValidBackupID(backupID) {
		return errors.New(actions.FormatBackupIDError(backupID))
	}

	action := &actions.RestoreAction{
		FS:      &actions.OSFileSystem{},
		DryRun:  restoreDryRun,
		Force:   restoreForce,
		Verbose: verbose,
		Stdin:   deps.Stdin,
		Stdout:  deps.Stdout,
		Stderr:  deps.Stderr,
	}

	if err := action.ResolveBackup(backupID); err != nil {
		return err
	}

	return action.Run()
}
