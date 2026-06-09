// Package windsurf implements the Adapter interface for Windsurf
// (Codeium's AI-powered IDE). It discovers, inventories, backs up, and
// restores configuration files from ~/.codeium/windsurf/.
package windsurf

import (
	"github.com/danielxxomg/bak-cli/internal/adapters"
)

const adapterName = "windsurf"
const configRelPath = ".codeium/windsurf"

var categoryMap = map[string]adapters.CategoryDir{
	"config": {SubPath: "", IsDir: false},
	"rules":  {SubPath: "rules", IsDir: true},
}

var base = adapters.GenericAdapter{
	AdapterName:      adapterName,
	ConfigRelPath:    configRelPath,
	Categories:       categoryMap,
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
