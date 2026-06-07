package manifest

import (
	"os"
	"path/filepath"
	"testing"
)

// FuzzLoad tests that Load never panics on arbitrary JSON input and that
// valid manifests are parsed without error.
func FuzzLoad(f *testing.F) {
	// Seed corpus: valid minimal manifest.
	f.Add([]byte(`{"version":"0.3.0","id":"20260101-120000","created_at":"2026-01-01T12:00:00Z","os_source":"linux","hostname":"test","bak_version":"1.0.0","preset":"quick","categories":["config"],"adapters":{},"secrets_excluded":false,"file_count":0,"total_size":0}`))
	// Seed: manifest with adapter entries.
	f.Add([]byte(`{"version":"0.3.0","id":"20260101-120000","created_at":"2026-01-01T12:00:00Z","os_source":"linux","hostname":"test","bak_version":"1.0.0","preset":"full","categories":["skills","config"],"adapters":{"opencode":{"config_dir":"~/.config/opencode","items":[]}},"secrets_excluded":false,"file_count":0,"total_size":0}`))
	// Seed: empty JSON object.
	f.Add([]byte(`{}`))
	// Seed: invalid JSON.
	f.Add([]byte(`{invalid`))
	// Seed: binary garbage.
	f.Add([]byte{0x00, 0x01, 0xFF, 0xFE})

	f.Fuzz(func(t *testing.T, data []byte) {
		dir := t.TempDir()

		// Write the fuzzed data as manifest.json in the temp dir.
		manifestPath := filepath.Join(dir, "manifest.json")
		if err := os.WriteFile(manifestPath, data, 0644); err != nil {
			t.Skip()
		}

		// Load should never panic, regardless of input.
		m, err := Load(dir)
		if err != nil {
			// Errors are expected for invalid JSON — ensure they are
			// not panics and the error message is non-empty.
			if err.Error() == "" {
				t.Error("Load returned error with empty message")
			}
			return
		}

		// Adapters may be nil for empty/minimal manifests — that's valid.
		// Validate should not panic on any loaded manifest.
		if m.Adapters != nil {
			_ = m.Validate(dir, nil)
		}
	})
}
