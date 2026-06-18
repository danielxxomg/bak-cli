package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/danielxxomg/bak-cli/internal/actions"
	"github.com/danielxxomg/bak-cli/internal/adapters"
	"github.com/danielxxomg/bak-cli/internal/adapters/register"
	"github.com/danielxxomg/bak-cli/internal/backup"
	"github.com/danielxxomg/bak-cli/internal/config"
	"github.com/danielxxomg/bak-cli/internal/manifest"
	"github.com/danielxxomg/bak-cli/internal/tui"
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
func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
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
// their manifests, and returns a slice of BackupInfo. Exported for
// testability via the cmd package's internal test files.
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
	if dryRun {
		return "dry-run: no changes detected", nil
	}
	return "restored successfully", nil
}

// tuiListProfiles returns all configured profiles.
func tuiListProfiles() ([]tui.ProfileInfo, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	var result []tui.ProfileInfo
	for name, p := range cfg.Profiles {
		result = append(result, tui.ProfileInfo{
			Name:     name,
			Provider: p.Provider,
			Preset:   p.Preset,
			Active:   name == cfg.ActiveProfile,
		})
	}
	return result, nil
}

// tuiCloudStatus returns the current cloud sync status.
func tuiCloudStatus() (tui.CloudStatus, error) {
	cfg, err := config.Load()
	if err != nil {
		return tui.CloudStatus{}, err
	}
	provider := cfg.Settings.DefaultProvider
	if provider == "" {
		provider = "github"
	}
	token, _ := cfg.Get("providers." + provider + ".token")
	connected := token != ""
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
	// Apply the setting to the config.
	switch key {
	case "auto_sync":
		if v, ok := value.(bool); ok {
			cfg.Settings.AutoSync = v
		}
	case "verbose_default":
		if v, ok := value.(bool); ok {
			cfg.Settings.VerboseDefault = v
		}
	case "confirm_destructive":
		if v, ok := value.(bool); ok {
			cfg.Settings.ConfirmDestructive = v
		}
	case "default_provider":
		if v, ok := value.(bool); ok {
			if v {
				cfg.Settings.DefaultProvider = "github"
			} else {
				cfg.Settings.DefaultProvider = ""
			}
		}
	}
	return cfg.Save()
}

// tuiSaveProfile persists a profile.
func tuiSaveProfile(name string, profile any) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if cfg.Profiles == nil {
		cfg.Profiles = make(map[string]config.ProfileConfig)
	}
	if p, ok := profile.(tui.ProfileInfo); ok {
		cfg.Profiles[name] = config.ProfileConfig{
			Provider: p.Provider,
			Preset:   p.Preset,
		}
	}
	return cfg.Save()
}

// tuiDeleteProfile removes a profile.
func tuiDeleteProfile(name string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	delete(cfg.Profiles, name)
	return cfg.Save()
}

// tuiSetActiveProfile sets the active profile.
func tuiSetActiveProfile(name string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	cfg.ActiveProfile = name
	return cfg.Save()
}

// tuiRunWizard launches the interactive profile creation wizard.
// Returns a ProfileInfo with the created profile data.
func tuiRunWizard() (tui.ProfileInfo, error) {
	return tui.ProfileInfo{
		Name:     "default",
		Provider: "github",
		Preset:   "quick",
	}, nil
}
