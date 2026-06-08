// Package presets provides YAML preset loading and validation for bak-cli.
// Custom presets can be loaded from ~/.config/bak/presets/*.yaml.
package presets

// YAMLPreset represents a user-defined preset loaded from YAML.
type YAMLPreset struct {
	Name       string       `yaml:"name"`
	Categories []string     `yaml:"categories"`
	Metadata   YAMLMetadata `yaml:"metadata,omitempty"`
}

// YAMLMetadata holds optional descriptive fields for a YAML preset.
type YAMLMetadata struct {
	Description string `yaml:"description,omitempty"`
	Author      string `yaml:"author,omitempty"`
}
