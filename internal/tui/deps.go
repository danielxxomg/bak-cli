// Package tui provides the root Bubble Tea model and screen routing for
// the bak-cli interactive TUI.
package tui

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
