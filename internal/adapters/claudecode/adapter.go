// Package claudecode implements the Adapter interface for Claude Code
// (Anthropic's CLI coding agent). It discovers, inventories, backs up, and
// restores configuration files from ~/.claude/.
package claudecode

import (
	"github.com/danielxxomg/bak-cli/internal/adapters"
)

// AdapterName is the adapter identifier exposed for knowledge validation.
const AdapterName = "claude-code"

// ConfigRelPath is the adapter config directory relative to the user home, exposed for knowledge validation.
const ConfigRelPath = ".claude"

// CategoryMap maps category names to their subdirectory/file patterns, exposed for knowledge validation.
var CategoryMap = map[string]adapters.CategoryDir{
	"config":   {SubPath: "", IsDir: false},
	"skills":   {SubPath: "skills", IsDir: true},
	"commands": {SubPath: "commands", IsDir: true},
	"agents":   {SubPath: "agents", IsDir: true},
	"plugins":  {SubPath: "plugins", IsDir: true},
}

var base = adapters.GenericAdapter{
	AdapterName:      AdapterName,
	ConfigRelPath:    ConfigRelPath,
	Categories:       CategoryMap,
	DetectErrContext: "stat claude-code config dir",
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
