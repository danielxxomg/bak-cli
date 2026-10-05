// Package codex implements the Adapter interface for OpenAI Codex CLI.
// It discovers, inventories, backs up, and restores configuration files
// from ~/.codex/.
package codex

import (
	"github.com/danielxxomg/bak-cli/internal/adapters"
)

// AdapterName is the adapter identifier exposed for knowledge validation.
const AdapterName = "codex"

// ConfigRelPath is the adapter config directory relative to the user home, exposed for knowledge validation.
const ConfigRelPath = ".codex"

// CategoryMap maps category names to their subdirectory/file patterns, exposed for knowledge validation.
var CategoryMap = map[string]adapters.CategoryDir{
	configCategory: {SubPath: "", IsDir: false},
	// agents is the directory of sub-agent definition files, matching the
	// opencode adapter. It is not where a main instruction file belongs.
	agentsCategory: {SubPath: "agent", IsDir: true},
}

const (
	agentsCategory = "agents"
	mcpCategory    = "mcp"
)

var base = adapters.GenericAdapter{
	AdapterName:      AdapterName,
	ConfigRelPath:    ConfigRelPath,
	Categories:       CategoryMap,
	DetectErrContext: "stat codex config dir",
	RootConfigFiles: map[string]string{
		// Tool configuration
		"config.toml": configCategory,
		"config.json": configCategory,
		"config.yaml": configCategory,
		"config.yml":  configCategory,
		// Instructions
		"instructions.md": configCategory,
		"INSTRUCTIONS.md": configCategory,
		// A main instruction file is configuration, not a sub-agent
		// definition: the default quick preset must cover it. This matches the
		// opencode adapter, which already maps AGENTS.md to config.
		"AGENTS.md": configCategory,
		"agents.md": configCategory,
		// Hooks
		"hooks.json": configCategory,
		"hooks.toml": configCategory,
		"hooks.yaml": configCategory,
		"hooks.yml":  configCategory,
		// Model Context Protocol
		"mcp.json": mcpCategory,
	},
}

// Adapter delegates all interface methods to a package-level GenericAdapter,
// preserving the zero-value construction pattern (&Adapter{}) used by register.go.
type Adapter struct{}

// Compile-time check: Adapter satisfies the adapters.Adapter interface.
var _ adapters.Adapter = (*Adapter)(nil)

// Compile-time check: Adapter satisfies ScanConfigurable.
var _ adapters.ScanConfigurable = (*Adapter)(nil)

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

// SetScanOptions forwards scan options to the underlying GenericAdapter.
func (a *Adapter) SetScanOptions(opts adapters.ScanOptions) { base.ScanOpts = opts }
