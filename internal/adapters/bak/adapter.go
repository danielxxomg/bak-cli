// Package bak implements the Adapter interface for bak-cli's own
// configuration. It discovers, inventories, backs up, and restores
// configuration files from ~/.config/bak/.
package bak

import (
	"github.com/danielxxomg/bak-cli/internal/adapters"
)

// AdapterName is the adapter identifier exposed for knowledge validation.
const AdapterName = "bak"

// ConfigRelPath is the adapter config directory relative to the user home, exposed for knowledge validation.
const ConfigRelPath = ".config/bak"

// CategoryMap maps category names to their subdirectory/file patterns, exposed for knowledge validation.
var CategoryMap = map[string]adapters.CategoryDir{
	"config":   {SubPath: "", IsDir: false},
	"adapters": {SubPath: "adapters", IsDir: true},
}

// rootConfigFiles lists the file names under the config root that belong
// to the "config" category, acting as an allowlist to prevent backing up
// migration leftovers (config.json.*.bak) or unexpected files.
var rootConfigFiles = map[string]string{
	"config.json": "config",
}

var base = adapters.GenericAdapter{
	AdapterName:      AdapterName,
	ConfigRelPath:    ConfigRelPath,
	Categories:       CategoryMap,
	DetectErrContext: "stat bak config dir",
	RootConfigFiles:  rootConfigFiles,
}

// Adapter delegates all interface methods to a package-level GenericAdapter,
// preserving the zero-value construction pattern (&Adapter{}) used by register.go.
type Adapter struct{}

// Compile-time check: Adapter satisfies the adapters.Adapter interface.
var _ adapters.Adapter = (*Adapter)(nil)

// Compile-time check: Adapter satisfies ScanConfigurable.
var _ adapters.ScanConfigurable = (*Adapter)(nil)

// Name returns the adapter identifier.
func (a *Adapter) Name() string { return base.Name() }

// Detect checks whether ~/.config/bak/ exists on disk.
func (a *Adapter) Detect(homeDir string) (bool, string, error) { return base.Detect(homeDir) }

// ListItems enumerates files and directories belonging to the requested
// categories, delegating to the underlying GenericAdapter.
func (a *Adapter) ListItems(homeDir string, cats []string) ([]adapters.Item, error) {
	return base.ListItems(homeDir, cats)
}

// Backup copies items from the bak config directory into the backup directory.
func (a *Adapter) Backup(homeDir, backupDir string, items []adapters.Item) error {
	return base.Backup(homeDir, backupDir, items)
}

// Restore copies items from the backup directory back to the bak config directory.
func (a *Adapter) Restore(backupDir, homeDir string, items []adapters.Item) error {
	return base.Restore(backupDir, homeDir, items)
}

// SetScanOptions forwards scan options to the underlying GenericAdapter.
func (a *Adapter) SetScanOptions(opts adapters.ScanOptions) { base.ScanOpts = opts }
