// Package tui provides the root Bubble Tea model and screen routing for
// the bak-cli interactive TUI.
package tui

import "github.com/danielxxomg/bak-cli/internal/tui/screens"

// Deps holds injectable dependencies for the TUI model. Function fields enable
// test doubles without interface boilerplate, matching the existing cmdDeps
// pattern used throughout bak-cli.
type Deps struct {
	// Version is the application version string shown in the UI.
	Version string

	// ListBackups returns all known backups. May be nil during testing.
	ListBackups func() ([]BackupInfo, error)

	// RunBackup executes a backup operation for the given categories and
	// streams progress updates through the provided channel.
	RunBackup func(cats []string, ch chan<- ProgressUpdate) error

	// ConfigExists returns true if a bak configuration directory already
	// exists on this machine. Used for first-run detection.
	ConfigExists func() bool

	// LoadSettings loads the persisted Settings from the config file.
	// nil means use defaults (zero-value Settings).
	// Populated by cmd/root.go before launching the TUI.
	LoadSettings func() (screens.Settings, error)

	// RunRestore executes a restore operation. When dryRun is true, it
	// returns a diff preview string without modifying any files.
	RunRestore func(backupID string, dryRun bool) (string, error)

	// ListProfiles returns all configured backup profiles.
	ListProfiles func() ([]ProfileInfo, error)

	// GetCloudStatus returns the current cloud sync status.
	GetCloudStatus func() (CloudStatus, error)

	// SaveSetting persists a single settings key-value pair to config.
	SaveSetting func(key string, value any) error

	// SaveProfile persists a new or updated profile.
	SaveProfile func(name string, profile any) error

	// DeleteProfile removes a profile by name.
	DeleteProfile func(name string) error

	// SetActiveProfile sets the named profile as active.
	SetActiveProfile func(name string) error

	// RunWizard launches the interactive profile creation wizard and
	// returns the created profile data.
	RunWizard func() (ProfileInfo, error)
}

// BackupInfo represents a single backup record displayed in the dashboard.
type BackupInfo struct {
	ID     string
	Date   string
	Size   string
	Status string
	Cloud  string
}

// ProgressUpdate carries incremental progress during a backup operation.
type ProgressUpdate struct {
	Step    string
	Current int
	Total   int
	Done    bool
}

// MenuSelection captures the user's menu choice after the TUI exits.
type MenuSelection struct {
	// Cursor is the zero-based index of the selected item.
	Cursor int
	// Item is the label of the selected menu entry.
	Item string
}

// DefaultMenuItems are the 7 main menu entries shared by the root model
// and the menu screen renderer.
var DefaultMenuItems = []string{
	"Create backup",
	"Restore",
	"Browse backups",
	"Cloud sync",
	"Profiles",
	"Settings",
	"Quit",
}

// ProfileInfo holds profile data for display in the profiles screen and
// for wizard results. Mirrors screens.ProfileInfo without importing screens.
type ProfileInfo struct {
	Name     string
	Provider string
	Preset   string
	Active   bool
}

// CloudStatus holds the cloud sync status returned by GetCloudStatus.
// Mirrors screens.CloudInfo without importing screens.
type CloudStatus struct {
	Provider   string
	Connected  bool
	LastSync   string
	LocalCount int
	CloudCount int
}
