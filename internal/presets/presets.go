// Package presets defines the named backup presets (quick, full, skills)
// and provides a resolver that maps a preset name to its category list.
package presets

import (
	"fmt"
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
	CatSkills    = "skills"
	CatCommands  = "commands"
	CatConfig    = "config"
	CatMCP       = "mcp"
	CatPlugins   = "plugins"
	CatAgents    = "agents"
	CatSecrets   = "secrets" // not backed up by default
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
