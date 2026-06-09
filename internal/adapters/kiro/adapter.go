// Package kiro implements the Adapter interface for Kiro (an AI
// coding agent). It discovers, inventories, backs up, and restores
// configuration files from ~/.kiro/.
package kiro

import (
	"github.com/danielxxomg/bak-cli/internal/adapters"
)

const adapterName = "kiro"
const configRelPath = ".kiro"

var categoryMap = map[string]adapters.CategoryDir{
	"config": {SubPath: "", IsDir: false},
	"hooks":  {SubPath: "hooks", IsDir: true},
}

var base = adapters.GenericAdapter{
	AdapterName:      adapterName,
	ConfigRelPath:    configRelPath,
	Categories:       categoryMap,
	DetectErrContext: "stat kiro config dir",
}

// Adapter delegates all interface methods to a package-level GenericAdapter,
// preserving the zero-value construction pattern (&Adapter{}) used by register.go.
type Adapter struct{}

// Compile-time check: Adapter satisfies the adapters.Adapter interface.
var _ adapters.Adapter = (*Adapter)(nil)

// Name returns the adapter identifier.
func (a *Adapter) Name() string { return base.Name() }

// Detect checks whether ~/.kiro exists and is a directory.
func (a *Adapter) Detect(homeDir string) (bool, string, error) { return base.Detect(homeDir) }

// ListItems enumerates config files and hooks from ~/.kiro.
func (a *Adapter) ListItems(homeDir string, cats []string) ([]adapters.Item, error) {
	return base.ListItems(homeDir, cats)
}

// Backup copies kiro items from ~/.kiro into the backup directory.
func (a *Adapter) Backup(homeDir, backupDir string, items []adapters.Item) error {
	return base.Backup(homeDir, backupDir, items)
}

// Restore copies kiro items from the backup directory back to ~/.kiro.
func (a *Adapter) Restore(backupDir, homeDir string, items []adapters.Item) error {
	return base.Restore(backupDir, homeDir, items)
}
