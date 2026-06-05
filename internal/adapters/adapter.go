// Package adapters defines the agent adapter interface and registry.
// Each adapter knows how to detect, inventory, backup, and restore
// configuration files for a specific AI coding tool.
package adapters

// Adapter is the contract every agent adapter must fulfill.
// v1 implements OpenCode only; the interface is designed for future
// agents (Claude Code, Cursor, Codex, Windsurf, etc.) without changes.
type Adapter interface {
	// Name returns the adapter identifier (e.g., "opencode", "claude-code").
	Name() string

	// Detect checks whether the target agent is installed and returns
	// its configuration directory path relative to the user's home.
	Detect(homeDir string) (installed bool, configDir string, err error)

	// ListItems enumerates the files and directories belonging to the
	// requested categories. Returns an empty slice when nothing matches.
	ListItems(homeDir string, categories []string) ([]Item, error)

	// Backup copies items from their source locations into the backup
	// directory, preserving the relative structure.
	Backup(homeDir string, backupDir string, items []Item) error

	// Restore copies items from the backup directory back to the target
	// home, adapting paths for the current platform.
	Restore(backupDir string, homeDir string, items []Item) error
}

// Item represents a single file or directory discovered by an adapter.
type Item struct {
	Category   string // "skills", "commands", "config", "mcp", "plugins", "agents"
	SourcePath string // absolute path on disk (canonical ~/ form)
	RelPath    string // path relative to adapter's config directory
	IsDir      bool
	Hash       string // SHA-256 hex digest of content
	Size       int64  // file size in bytes
}
