// Package cursor implements the Adapter interface for Cursor
// (the AI-first code editor). It discovers, inventories, backs up, and
// restores configuration files from ~/.cursor/.
package cursor

import (
	"github.com/danielxxomg/bak-cli/internal/adapters"
)

// AdapterName is the adapter identifier exposed for knowledge validation.
const AdapterName = "cursor"

// ConfigRelPath is the adapter config directory relative to the user home, exposed for knowledge validation.
const ConfigRelPath = ".cursor"

// CategoryMap maps category names to their subdirectory/file patterns, exposed for knowledge validation.
var CategoryMap = map[string]adapters.CategoryDir{
	"config":     {SubPath: "", IsDir: false},
	"extensions": {SubPath: "extensions", IsDir: true},
	"mcp":        {SubPath: "", IsDir: false},
}

var base = adapters.GenericAdapter{
	AdapterName:      AdapterName,
	ConfigRelPath:    ConfigRelPath,
	Categories:       CategoryMap,
	DetectErrContext: "stat cursor config dir",
}

// Adapter delegates all interface methods to a package-level GenericAdapter,
// preserving the zero-value construction pattern (&Adapter{}) used by register.go.
type Adapter struct{}

// Compile-time check: Adapter satisfies the adapters.Adapter interface.
var _ adapters.Adapter = (*Adapter)(nil)

func (a *Adapter) Name() string                                { return base.Name() }
func (a *Adapter) Detect(homeDir string) (bool, string, error) { return base.Detect(homeDir) }
func (a *Adapter) ListItems(homeDir string, cats []string) ([]adapters.Item, error) {
	return base.ListItems(homeDir, cats)
}
func (a *Adapter) Backup(homeDir, backupDir string, items []adapters.Item) error {
	return base.Backup(homeDir, backupDir, items)
}
func (a *Adapter) Restore(backupDir, homeDir string, items []adapters.Item) error {
	return base.Restore(backupDir, homeDir, items)
}
