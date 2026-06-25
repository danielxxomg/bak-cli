package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	tea "charm.land/bubbletea/v2"

	"github.com/danielxxomg/bak-cli/internal/actions"
	"github.com/danielxxomg/bak-cli/internal/adapters"
	"github.com/danielxxomg/bak-cli/internal/adapters/register"
	"github.com/danielxxomg/bak-cli/internal/backup"
	"github.com/danielxxomg/bak-cli/internal/config"
	"github.com/danielxxomg/bak-cli/internal/manifest"
	"github.com/danielxxomg/bak-cli/internal/tui"
	"github.com/danielxxomg/bak-cli/internal/tui/screens"
)

var verbose bool

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   "bak",
	Short: "Backup and restore your AI coding setup",
	Long: `bak packs, restores, and syncs your OpenCode configuration
across machines with safety guarantees.

Run 'bak backup' to create a backup.
Run 'bak restore --dry-run <id>' to preview before applying.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Launch the interactive TUI when bak is invoked with no arguments
		// and stdout is a terminal. Cobra handles --help before RunE, so
		// `bak --help` still shows cobra help regardless of TTY.
		if len(args) == 0 && isTTY() {
			deps := tui.Deps{
				Version:          Version,
				ConfigExists:     configExists,
				ListBackups:      listBackups,
				RunBackup:        tuiRunBackup,
				RunRestore:       tuiRunRestore,
				ListProfiles:     tuiListProfiles,
				GetCloudStatus:   tuiCloudStatus,
				SaveSetting:      tuiSaveSetting,
				SaveProfile:      tuiSaveProfile,
				DeleteProfile:    tuiDeleteProfile,
				SetActiveProfile: tuiSetActiveProfile,
				RunWizard:        tuiRunWizard,
				LoadSettings:     loadSettingsForTUI,
			}
			return runTUI(deps)
		}
		// Fall through to cobra help output (non-TTY or non-empty args).
		return cmd.Help()
	},
}

// configExists returns true if the bak configuration file exists on disk.
// Used by the TUI for first-run welcome detection.
func configExists() bool {
	cfgPath, err := config.DefaultPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(cfgPath)
	return err == nil
}

// Execute runs the root command.
func Execute() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	if err := rootCmd.Execute(); err != nil {
		if verbose {
			fmt.Fprintf(os.Stderr, "Error: %+v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		os.Exit(1)
	}
}

// formatSize returns a human-readable byte count.
func formatSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

// listBackups scans the local backups directory and returns a slice of
// BackupInfo suitable for populating the TUI dashboard.
func listBackups() ([]tui.BackupInfo, error) {
	bakDir, err := backup.BakDir()
	if err != nil {
		return nil, fmt.Errorf("bak dir: %w", err)
	}
	return listBackupsFrom(bakDir)
}

// listBackupsFrom scans the given bakDir for backup directories, loads
// their manifests, and returns a slice of BackupInfo. This function is
// package-visible for testability via cmd's internal test files.
func listBackupsFrom(bakDir string) ([]tui.BackupInfo, error) {
	backupsDir := filepath.Join(bakDir, "backups")
	entries, err := os.ReadDir(backupsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read backups dir: %w", err)
	}

	var result []tui.BackupInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		m, err := manifest.Load(filepath.Join(backupsDir, entry.Name()))
		if err != nil {
			continue // skip corrupt/incomplete backup dirs
		}

		// Format date from backup ID (YYYYMMDD-HHMMSS).
		date := ""
		if len(m.ID) >= 15 {
			date = fmt.Sprintf("%s-%s-%s %s:%s:%s",
				m.ID[0:4], m.ID[4:6], m.ID[6:8],
				m.ID[9:11], m.ID[11:13], m.ID[13:15])
		}

		// Pick first adapter name as cloud provider hint.
		cloudStr := "none"
		for name := range m.Adapters {
			cloudStr = name
			break
		}

		result = append(result, tui.BackupInfo{
			ID:     m.ID,
			Date:   date,
			Size:   formatSize(m.TotalSize),
			Status: "ok",
			Cloud:  cloudStr,
		})
	}
	return result, nil
}

// --- TUI Deps injection stubs ---
// These functions are thin adapters that bridge cmd/ infrastructure to
// the TUI's Deps function fields. Full implementations wire into
// actions.*Action; stubs return default/empty results for now.

// tuiRunBackup executes a backup operation and sends progress updates through
// the provided channel. It wraps actions.BackupAction with a progressFn
// that bridges the callback to TUI ProgressUpdate messages.
func tuiRunBackup(cats []string, ch chan<- tui.ProgressUpdate) error {
	defer close(ch)

	reg, err := newTuiRegistry()
	if err != nil {
		return err
	}

	action := &actions.BackupAction{
		FS:         &actions.OSFileSystem{},
		Registry:   reg,
		Preset:     "quick",
		BakVersion: Version,
		ProgressFn: func(file string, done, total int) {
			select {
			case ch <- tui.ProgressUpdate{
				Step:    file,
				Current: done,
				Total:   total,
			}:
			default:
				// Channel full — drop update (non-blocking to avoid
				// deadlocking the backup goroutine).
			}
		},
		CustomCategories: cats,
		ExcludesLoader:   loadExcludes,
	}

	err = action.Run()

	// Send final "Done" update regardless of error.
	ch <- tui.ProgressUpdate{Done: true}

	return err
}

// newTuiRegistry creates an adapter registry with all built-in agents
// registered. Used by the TUI runBackup adapter.
func newTuiRegistry() (*adapters.Registry, error) {
	reg := adapters.NewRegistry()
	if err := register.All(reg); err != nil {
		return nil, fmt.Errorf("register adapters: %w", err)
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("home dir: %w", err)
	}
	if err := register.LoadYAMLAdapters(reg, false, homeDir); err != nil {
		return nil, fmt.Errorf("load yaml adapters: %w", err)
	}
	return reg, nil
}

// tuiRunRestore wraps the restore action for TUI flow.
func tuiRunRestore(backupID string, dryRun bool) (string, error) {
	var buf bytes.Buffer

	action := &actions.RestoreAction{
		FS:      &actions.OSFileSystem{},
		DryRun:  dryRun,
		Force:   !dryRun, // TUI modal is the confirmation gate
		Verbose: verbose,
		Stdout:  &buf,
		Stderr:  &buf,
	}

	if err := action.ResolveBackup(backupID); err != nil {
		return "", err
	}

	if err := action.Run(); err != nil {
		return buf.String(), err
	}

	return buf.String(), nil
}

// tuiListProfiles returns all configured profiles.
func tuiListProfiles() ([]tui.ProfileInfo, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	infos := actions.ListProfileInfos(cfg)
	result := make([]tui.ProfileInfo, len(infos))
	for i, info := range infos {
		result[i] = tui.ProfileInfo{
			Name:     info.Name,
			Provider: info.Provider,
			Preset:   info.Preset,
			Active:   info.Active,
		}
	}
	return result, nil
}

// tuiCloudStatus returns the current cloud sync status.
func tuiCloudStatus() (tui.CloudStatus, error) {
	cfg, err := config.Load()
	if err != nil {
		return tui.CloudStatus{}, err
	}
	provider, connected := actions.GetCloudProviderStatus(cfg)
	return tui.CloudStatus{
		Provider:  provider,
		Connected: connected,
		LastSync:  "never",
	}, nil
}

// tuiSaveSetting persists a single settings value.
func tuiSaveSetting(key string, value any) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	actions.SaveSetting(cfg, key, value)
	return cfg.Save()
}

// tuiSaveProfile persists a profile.
func tuiSaveProfile(name string, profile any) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if p, ok := profile.(tui.ProfileInfo); ok {
		actions.SaveProfileFromInfo(cfg, name, p.Provider, p.Preset)
	}
	return cfg.Save()
}

// tuiDeleteProfile removes a profile.
func tuiDeleteProfile(name string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	actions.DeleteProfileSilent(cfg, name)
	return cfg.Save()
}

// tuiSetActiveProfile sets the active profile.
func tuiSetActiveProfile(name string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	actions.SetActiveProfile(cfg, name)
	return cfg.Save()
}

// tuiRunWizard launches the interactive profile creation wizard.
// Returns a ProfileInfo with the created profile data.
func tuiRunWizard() (tui.ProfileInfo, error) {
	m := screens.NewWizardModel("profile-create", nil) // nil providers → wizard auto-detects
	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return tui.ProfileInfo{}, err
	}
	wm, ok := finalModel.(*screens.WizardModel)
	if !ok {
		return tui.ProfileInfo{}, fmt.Errorf("wizard: unexpected model type %T", finalModel)
	}
	if !wm.Confirmed {
		return tui.ProfileInfo{}, fmt.Errorf("wizard cancelled")
	}
	return tui.ProfileInfo{
		Name:     wm.ProfileName(),
		Provider: wm.SelectedProvider,
		Preset:   wm.SelectedPreset,
	}, nil
}

// loadSettingsForTUI reads the config and converts config.Settings to
// screens.Settings for the TUI settings screen.
func loadSettingsForTUI() (screens.Settings, error) {
	cfg, err := config.Load()
	if err != nil {
		return screens.Settings{}, err
	}
	confirmDestructive := false
	if cfg.Settings.ConfirmDestructive != nil {
		confirmDestructive = *cfg.Settings.ConfirmDestructive
	}
	return screens.Settings{
		DefaultPreset:      cfg.Settings.DefaultPreset,
		AutoSync:           cfg.Settings.AutoSync,
		ExcludePatterns:    cfg.Settings.ExcludePatterns,
		MaxFileSize:        cfg.Settings.MaxFileSize,
		ConfirmDestructive: confirmDestructive,
		VerboseDefault:     cfg.Settings.VerboseDefault,
		DefaultProvider:    cfg.Settings.DefaultProvider,
	}, nil
}
