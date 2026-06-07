package adapters

import (
	"os"
	"path/filepath"
	"testing"
)

// FuzzLoadYAMLAdapters tests that LoadYAMLAdapters never panics on arbitrary
// YAML input and that valid adapter definitions are parsed correctly.
func FuzzLoadYAMLAdapters(f *testing.F) {
	// Seed corpus: valid adapter YAML.
	f.Add([]byte("name: test-adapter\nconfig_path: .config/testapp\ncategories:\n  - name: config\n    root_files:\n      - config.json\n"))
	// Seed: minimal valid adapter.
	f.Add([]byte("name: min\nconfig_path: .config/min\ncategories: []\n"))
	// Seed: adapter with directory category.
	f.Add([]byte("name: dir-adapter\nconfig_path: .config/dir\ncategories:\n  - name: skills\n    sub_path: skills\n    is_dir: true\n"))
	// Seed: empty YAML (should fail validation).
	f.Add([]byte(""))
	// Seed: invalid YAML syntax.
	f.Add([]byte("name: [unclosed\n  - broken"))
	// Seed: missing required fields.
	f.Add([]byte("categories:\n  - name: test\n"))

	f.Fuzz(func(t *testing.T, data []byte) {
		dir := t.TempDir()

		// Write the fuzzed data as a .yaml file in the temp dir.
		adapterPath := filepath.Join(dir, "adapter.yaml")
		if err := os.WriteFile(adapterPath, data, 0644); err != nil {
			t.Skip()
		}

		// LoadYAMLAdapters should never panic, regardless of input.
		adapters, err := LoadYAMLAdapters(dir, dir)
		if err != nil {
			// Errors are expected — ensure message is non-empty.
			if err.Error() == "" {
				t.Error("LoadYAMLAdapters returned error with empty message")
			}
			return
		}

		// If no error, verify loaded adapters are valid.
		for _, a := range adapters {
			if a.Name() == "" {
				t.Error("loaded adapter has empty name")
			}
			// Detect should not panic with bogus home dir.
			installed, configDir, detectErr := a.Detect(dir)
			if detectErr != nil && detectErr.Error() == "" {
				t.Error("Detect returned error with empty message")
			}
			// installed+configDir should be consistent.
			if installed && configDir == "" {
				t.Error("Detect returned installed=true but empty configDir")
			}
		}
	})
}
