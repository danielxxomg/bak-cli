// Package presets defines the named backup presets (quick, full, skills)
// and provides a resolver that maps a preset name to its category list.
package presets

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
)

// Preset names supported by the backup command.
const (
	Quick  = "quick"
	Full   = "full"
	Skills = "skills"
)

// Available categories. Each adapter maps these to actual file paths.
const (
	CatSkills   = "skills"
	CatCommands = "commands"
	CatConfig   = "config"
	CatMCP      = "mcp"
	CatPlugins  = "plugins"
	CatAgents   = "agents"
	CatSecrets  = "secrets" // not backed up by default
)

// AllCategories lists every category known to the system.
var AllCategories = []string{
	CatSkills, CatCommands, CatConfig, CatMCP, CatPlugins, CatAgents,
}

// presetCategories maps preset names to their category lists.
var presetCategories = map[string][]string{
	Quick:  {CatConfig},
	Full:   AllCategories,
	Skills: {CatSkills},
}

// Resolve maps a preset name to its category list.
// Returns an error for unknown presets.
func Resolve(name string) ([]string, error) {
	cats, ok := presetCategories[name]
	if !ok {
		return nil, fmt.Errorf("unknown preset %q (valid: quick, full, skills)", name)
	}
	// Return a copy so callers cannot mutate the internal slice.
	return slices.Clone(cats), nil
}

// Names returns all valid preset names.
func Names() []string {
	return []string{Quick, Full, Skills}
}

// IsValid returns true when name is a known preset.
func IsValid(name string) bool {
	_, ok := presetCategories[name]
	return ok
}

// ResolveAll loads YAML-defined presets from the standard directory
// (~/.config/bak/presets/) and merges them with built-in presets.
// When a YAML preset has the same name as a built-in:
//   - override=true: the YAML version wins, replacing the built-in.
//   - override=false: a warning is printed to stderr and the built-in is used.
//
// Unknown preset names fall through to the built-in resolver.
func ResolveAll(presetName string, override bool) ([]string, error) {
	yamlPresets, err := loadYAMLPresets()
	if err != nil {
		return nil, fmt.Errorf("load yaml presets: %w", err)
	}

	for _, yp := range yamlPresets {
		if yp.Name == presetName {
			if !override {
				_, isBuiltin := presetCategories[presetName]
				if isBuiltin {
					fmt.Fprintf(os.Stderr, "warning: preset %q exists as both built-in and custom; using built-in (use --override to prefer custom)\n", presetName)
					return Resolve(presetName)
				}
			}
			return slices.Clone(yp.Categories), nil
		}
	}

	return Resolve(presetName)
}

// loadYAMLPresets loads custom preset definitions from the standard
// YAML presets directory. Returns an empty slice when the directory
// does not exist.
func loadYAMLPresets() ([]YAMLPreset, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("home dir: %w", err)
	}
	dir := filepath.Join(home, ".config", "bak", "presets")
	return LoadFromDir(dir)
}
