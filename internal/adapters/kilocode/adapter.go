// Package kilocode implements the Adapter interface for KiloCode
// (an AI coding agent). It discovers, inventories, backs up, and
// restores configuration files from ~/.kilocode/.
package kilocode

import (
	"github.com/danielxxomg/bak-cli/internal/adapters"
)

const adapterName = "kilocode"
const configRelPath = ".kilocode"

var categoryMap = map[string]adapters.CategoryDir{
	"config": {SubPath: "", IsDir: false},
	"rules":  {SubPath: "rules", IsDir: true},
}

var base = adapters.GenericAdapter{
	AdapterName:      adapterName,
	ConfigRelPath:    configRelPath,
	Categories:       categoryMap,
	DetectErrContext: "stat kilocode config dir",
}

// Adapter delegates all interface methods to a package-level GenericAdapter,
// preserving the zero-value construction pattern (&Adapter{}) used by register.go.
type Adapter struct{}

// Compile-time check: Adapter satisfies the adapters.Adapter interface.
var _ adapters.Adapter = (*Adapter)(nil)

func (a *Adapter) Name() string                                                { return base.Name() }
func (a *Adapter) Detect(homeDir string) (bool, string, error)                 { return base.Detect(homeDir) }
func (a *Adapter) ListItems(homeDir string, cats []string) ([]adapters.Item, error) {
	return base.ListItems(homeDir, cats)
}
func (a *Adapter) Backup(homeDir, backupDir string, items []adapters.Item) error {
	return base.Backup(homeDir, backupDir, items)
}
func (a *Adapter) Restore(backupDir, homeDir string, items []adapters.Item) error {
	return base.Restore(backupDir, homeDir, items)
}
