// Package opencode implements the Adapter interface for the OpenCode
// AI coding tool. It discovers, inventories, backs up, and restores
// configuration files from ~/.config/opencode/.
//
// The adapter is a thin delegating wrapper around a package-level
// adapters.GenericAdapter (the same pattern used by kilocode and codex).
// All scan/backup/restore logic lives in GenericAdapter; this package
// only supplies OpenCode-specific constants and the category/root-file
// maps.
package opencode

import (
	"github.com/danielxxomg/bak-cli/internal/adapters"
)

// adapterName is the adapter identifier exposed to the registry and CLI.
const adapterName = "opencode"

// configRelPath is the OpenCode config directory relative to the user home.
const configRelPath = ".config/opencode"

// categoryMap maps category names to their subdirectory/file patterns under
// the OpenCode config root. The "config" and "mcp" categories are root-level
// (no subdirectory); their files are resolved via rootConfigFiles.
var categoryMap = map[string]adapters.CategoryDir{
	"skills":   {SubPath: "skills", IsDir: true},
	"commands": {SubPath: "commands", IsDir: true},
	"config":   {SubPath: "", IsDir: false}, // root-level config files
	"mcp":      {SubPath: "", IsDir: false}, // root-level mcp.json
	"plugins":  {SubPath: "plugins", IsDir: true},
	"agents":   {SubPath: "agent", IsDir: true},
}

// rootConfigFiles lists the file names under the config root that belong
// to the "config" and "mcp" categories, mapping each to its category.
var rootConfigFiles = map[string]string{
	// config category
	"opencode.jsonc": "config",
	"opencode.json":  "config",
	"config.json":    "config",
	"AGENTS.md":      "config",
	"tui.json":       "config",
	// mcp category
	"mcp.json": "mcp",
}

var base = adapters.GenericAdapter{
	AdapterName:      adapterName,
	ConfigRelPath:    configRelPath,
	Categories:       categoryMap,
	DetectErrContext: "stat opencode config dir",
	RootConfigFiles:  rootConfigFiles,
}

// Adapter delegates all interface methods to a package-level GenericAdapter,
// preserving the zero-value construction pattern (&Adapter{}) used by
// register.go.
type Adapter struct{}

// Compile-time check: Adapter satisfies the adapters.Adapter interface.
var _ adapters.Adapter = (*Adapter)(nil)

// Compile-time check: Adapter satisfies ScanConfigurable.
var _ adapters.ScanConfigurable = (*Adapter)(nil)

// Name returns the adapter identifier.
func (a *Adapter) Name() string { return base.Name() }

// Detect checks whether ~/.config/opencode/ exists on disk.
func (a *Adapter) Detect(homeDir string) (bool, string, error) { return base.Detect(homeDir) }

// ListItems enumerates files and directories belonging to the requested
// categories, delegating to the underlying GenericAdapter.
func (a *Adapter) ListItems(homeDir string, cats []string) ([]adapters.Item, error) {
	return base.ListItems(homeDir, cats)
}

// Backup copies items into the backup directory under an "opencode/" prefix.
func (a *Adapter) Backup(homeDir, backupDir string, items []adapters.Item) error {
	return base.Backup(homeDir, backupDir, items)
}

// Restore copies items from the backup directory back under ~/.config/opencode/.
func (a *Adapter) Restore(backupDir, homeDir string, items []adapters.Item) error {
	return base.Restore(backupDir, homeDir, items)
}

// SetScanOptions forwards scan options to the underlying GenericAdapter.
func (a *Adapter) SetScanOptions(opts adapters.ScanOptions) { base.ScanOpts = opts }
