package presets

import (
	"os"
	"path/filepath"
	"testing"
)

// FuzzLoadFromDir tests that LoadFromDir never panics on arbitrary YAML
// input and that valid presets are parsed correctly.
func FuzzLoadFromDir(f *testing.F) {
	// Seed corpus: valid preset YAML.
	f.Add([]byte("name: my-preset\ncategories:\n  - skills\n  - config\n"))
	// Seed: minimal valid preset.
	f.Add([]byte("name: min\ncategories:\n  - config\n"))
	// Seed: preset with metadata.
	f.Add([]byte("name: full\ncategories:\n  - skills\n  - config\nmetadata:\n  description: test\n  author: me\n"))
	// Seed: empty YAML (should fail validation, not panic).
	f.Add([]byte(""))
	// Seed: invalid YAML.
	f.Add([]byte("name: [unclosed"))
	// Seed: missing required fields.
	f.Add([]byte("categories:\n  - skills\n"))

	f.Fuzz(func(t *testing.T, data []byte) {
		dir := t.TempDir()

		// Write the fuzzed data as a .yaml file in the temp dir.
		presetPath := filepath.Join(dir, "preset.yaml")
		if err := os.WriteFile(presetPath, data, 0644); err != nil {
			t.Skip()
		}

		// LoadFromDir should never panic, regardless of input.
		presets, err := LoadFromDir(dir)
		if err != nil {
			// Errors are expected for invalid YAML — ensure they are
			// not panics and the error message is non-empty.
			if err.Error() == "" {
				t.Error("LoadFromDir returned error with empty message")
			}
			return
		}

		// If no error, verify loaded presets are valid.
		for _, p := range presets {
			if p.Name == "" {
				t.Error("loaded preset has empty name")
			}
			if len(p.Categories) == 0 {
				t.Error("loaded preset has empty categories")
			}
		}
	})
}
