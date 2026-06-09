// Package presets provides YAML preset loading and validation for bak-cli.
// Custom presets can be loaded from ~/.config/bak/presets/*.yaml.
package presets

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/danielxxomg/bak-cli/internal/paths"
)

// LoadFromDir scans a directory for *.yaml files and parses them as
// YAML presets. Files with invalid YAML or missing required fields
// are rejected with an error. Returns an empty slice when the
// directory does not exist.
func LoadFromDir(dir string) ([]YAMLPreset, error) {
	// Security: validate path stays under user home BEFORE any filesystem access.
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("get home dir: %w", err)
	}
	cleanDir := paths.CanonicalPath(dir)
	cleanHome := paths.CanonicalPath(home)
	if !strings.HasPrefix(cleanDir, cleanHome+"/") && cleanDir != cleanHome {
		return nil, fmt.Errorf("path traversal: directory outside home")
	}

	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read presets dir: %w", err)
	}

	var presets []YAMLPreset
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(strings.ToLower(entry.Name()), ".yaml") &&
			!strings.HasSuffix(strings.ToLower(entry.Name()), ".yml") {
			continue
		}

		presetPath := filepath.Join(dir, entry.Name())
		p, err := loadPresetFile(presetPath)
		if err != nil {
			return nil, fmt.Errorf("load preset %q: %w", entry.Name(), err)
		}
		presets = append(presets, *p)
	}

	return presets, nil
}

// loadPresetFile reads and validates a single YAML preset file.
func loadPresetFile(filePath string) (*YAMLPreset, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	var p YAMLPreset
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("yaml parse: %w", err)
	}

	if p.Name == "" {
		return nil, fmt.Errorf("missing required field \"name\"")
	}
	if len(p.Categories) == 0 {
		return nil, fmt.Errorf("missing required field \"categories\"")
	}

	return &p, nil
}
