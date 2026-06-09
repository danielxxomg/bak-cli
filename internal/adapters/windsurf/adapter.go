// Package windsurf implements the Adapter interface for Windsurf
// (Codeium's AI-powered IDE). It discovers, inventories, backs up, and
// restores configuration files from ~/.codeium/windsurf/.
package windsurf

import (
	"github.com/danielxxomg/bak-cli/internal/adapters"
)

// AdapterName is the adapter identifier exposed for knowledge validation.
const AdapterName = "windsurf"

// ConfigRelPath is the adapter config directory relative to the user home, exposed for knowledge validation.
const ConfigRelPath = ".codeium/windsurf"

// CategoryMap maps category names to their subdirectory/file patterns, exposed for knowledge validation.
var CategoryMap = map[string]adapters.CategoryDir{
	"config": {SubPath: "", IsDir: false},
	"rules":  {SubPath: "memories", IsDir: true},
	"skills": {SubPath: "skills", IsDir: true},
}

var base = adapters.GenericAdapter{
	AdapterName:      AdapterName,
	ConfigRelPath:    ConfigRelPath,
	Categories:       CategoryMap,
	DetectErrContext: "stat windsurf config dir",
}

// Adapter delegates all interface methods to a package-level GenericAdapter,
// preserving the zero-value construction pattern (&Adapter{}) used by register.go.
type Adapter struct{}

// Compile-time check: Adapter satisfies the adapters.Adapter interface.
var _ adapters.Adapter = (*Adapter)(nil)

// Name returns the adapter identifier.
func (a *Adapter) Name() string { return base.Name() }

// Detect checks whether the Windsurf config directory exists under homeDir.
func (a *Adapter) Detect(homeDir string) (bool, string, error) { return base.Detect(homeDir) }

// ListItems enumerates files and directories belonging to the requested categories.
func (a *Adapter) ListItems(homeDir string, cats []string) ([]adapters.Item, error) {
	return base.ListItems(homeDir, cats)
}

// Backup copies listed items from the Windsurf config directory to backupDir.
func (a *Adapter) Backup(homeDir, backupDir string, items []adapters.Item) error {
	return base.Backup(homeDir, backupDir, items)
}

// Restore copies listed items from backupDir back to the Windsurf config directory.
func (a *Adapter) Restore(backupDir, homeDir string, items []adapters.Item) error {
	return base.Restore(backupDir, homeDir, items)
}
