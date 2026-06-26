package backup

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/danielxxomg/bak-cli/internal/adapters"
	opencodeadapter "github.com/danielxxomg/bak-cli/internal/adapters/opencode"
)

// --- utility tests -------------------------------------------------------

func TestBakDir(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	dir, err := BakDir()
	if err != nil {
		t.Fatalf("BakDir: %v", err)
	}
	if dir == "" {
		t.Error("BakDir returned empty string")
	}
	// Should end with .bak
	if filepath.Base(dir) != ".bak" {
		t.Errorf("BakDir = %q, want path ending in .bak", dir)
	}
}

// --- helpers -------------------------------------------------------------

func setupTestEngine(t *testing.T, home string) *Engine {
	t.Helper()

	reg := adapters.NewRegistry()
	if err := reg.Register(&opencodeadapter.Adapter{}); err != nil {
		t.Fatalf("register: %v", err)
	}

	bakDir := filepath.Join(home, ".bak")

	return &Engine{
		HomeDir:    home,
		BakDir:     bakDir,
		Registry:   reg,
		Preset:     "quick",
		BakVersion: "test",
	}
}

func createOpenCodeFixture(t *testing.T, home string) {
	t.Helper()

	configDir := filepath.Join(home, ".config", "opencode")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Root config files (config + mcp categories).
	if err := os.WriteFile(filepath.Join(configDir, "opencode.json"), []byte(`{"version":"1.0"}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "AGENTS.md"), []byte("# Agents"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "mcp.json"), []byte(`{"servers":{}}`), 0644); err != nil {
		t.Fatal(err)
	}

	// Skills directory.
	skillDir := filepath.Join(configDir, "skills", "my-skill")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# Skill"), 0644); err != nil {
		t.Fatal(err)
	}
}

// --- table-driven: presets -----------------------------------------------

func TestEngine_Run_Presets(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	tests := []struct {
		name        string
		preset      string
		withFixture bool
		wantErr     bool
		validate    func(t *testing.T, result *Result)
	}{
		{
			name:        "quick preset",
			preset:      "quick",
			withFixture: true,
			wantErr:     false,
			validate: func(t *testing.T, result *Result) {
				if result.ID == "" {
					t.Error("backup ID is empty")
				}
				if result.FileCount == 0 {
					t.Error("expected at least 1 file for quick preset")
				}
				if result.AdaptersRun != 1 {
					t.Errorf("AdaptersRun = %d, want 1", result.AdaptersRun)
				}

				// Verify manifest exists.
				manifestPath := filepath.Join(result.BackupDir, "manifest.json")
				if _, err := os.Stat(manifestPath); err != nil {
					t.Fatalf("manifest not found: %v", err)
				}

				// Verify manifest content.
				data, err := os.ReadFile(manifestPath)
				if err != nil {
					t.Fatal(err)
				}
				var m map[string]interface{}
				if err := json.Unmarshal(data, &m); err != nil {
					t.Fatalf("invalid manifest JSON: %v", err)
				}

				if m["preset"] != "quick" {
					t.Errorf("manifest preset = %v, want quick", m["preset"])
				}
				if m["version"] != "0.3.0" {
					t.Errorf("manifest version = %v, want 0.3.0", m["version"])
				}
			},
		},
		{
			name:        "full preset",
			preset:      "full",
			withFixture: true,
			wantErr:     false,
			validate: func(t *testing.T, result *Result) {
				if result.FileCount < 4 {
					t.Errorf("expected at least 4 files for full preset, got %d", result.FileCount)
				}
			},
		},
		{
			name:        "invalid preset",
			preset:      "bananas",
			withFixture: true,
			wantErr:     true,
			validate:    nil,
		},
		{
			name:        "no adapters detected",
			preset:      "quick",
			withFixture: false,
			wantErr:     true,
			validate:    nil,
		},
	}

	for _, tt := range tests { //nolint:paralleltest // subtests share table/struct state
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
			home := t.TempDir()
			if tt.withFixture {
				createOpenCodeFixture(t, home)
			}

			engine := setupTestEngine(t, home)
			engine.Preset = tt.preset

			result, err := engine.Run()
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

// --- table-driven: adapter filters ---------------------------------------

func TestEngine_Run_AdapterFilters(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	tests := []struct {
		name    string
		filters []string
		wantErr bool
	}{
		{
			name:    "valid single filter",
			filters: []string{"opencode"},
			wantErr: false,
		},
		{
			name:    "unknown filter",
			filters: []string{"nonexistent"},
			wantErr: true,
		},
		{
			name:    "mixed valid and invalid",
			filters: []string{"opencode", "nonexistent"},
			wantErr: true,
		},
	}

	for _, tt := range tests { //nolint:paralleltest // subtests share table/struct state
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
			home := t.TempDir()
			createOpenCodeFixture(t, home)

			engine := setupTestEngine(t, home)
			engine.AdapterFilter = tt.filters

			result, err := engine.Run()
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			if result.AdaptersRun != 1 {
				t.Errorf("AdaptersRun = %d, want 1", result.AdaptersRun)
			}
		})
	}
}

// --- table-driven: progress function -------------------------------------

func TestEngine_Run_ProgressFn(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	type call struct {
		file  string
		done  int
		total int
	}

	tests := []struct {
		name     string
		cb       func(file string, done, total int)
		validate func(t *testing.T, calls []call)
	}{
		{
			name: "callback called with incrementing done count",
			cb:   nil, // collector is wired inside t.Run based on validate != nil
			validate: func(t *testing.T, calls []call) {
				if len(calls) == 0 {
					t.Fatal("ProgressFn was not called at all")
				}

				// Verify incrementing done count.
				for i := 1; i < len(calls); i++ {
					if calls[i].done <= calls[i-1].done {
						t.Errorf("done count did not increment: calls[%d].done=%d, calls[%d].done=%d",
							i-1, calls[i-1].done, i, calls[i].done)
					}
				}

				// Total should be consistent across all calls.
				total := calls[0].total
				if total <= 0 {
					t.Fatalf("expected positive total, got %d", total)
				}
				for i, c := range calls {
					if c.total != total {
						t.Errorf("calls[%d].total = %d, want %d", i, c.total, total)
					}
				}

				// Last done should equal total.
				lastCall := calls[len(calls)-1]
				if lastCall.done != lastCall.total {
					t.Errorf("last call done=%d, want total=%d", lastCall.done, lastCall.total)
				}
			},
		},
		{
			name:     "nil callback does not panic",
			cb:       nil,
			validate: nil,
		},
	}

	for _, tt := range tests { //nolint:paralleltest // subtests share table/struct state
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
			home := t.TempDir()
			createOpenCodeFixture(t, home)

			engine := setupTestEngine(t, home)

			var calls []call
			switch {
			case tt.validate != nil:
				// "callback called" case: wire the collector.
				engine.Preset = "full"
				engine.ProgressFn = func(file string, done, total int) {
					calls = append(calls, call{file: file, done: done, total: total})
				}
			default:
				// "nil callback" case: explicitly nil.
				engine.Preset = "quick"
				engine.ProgressFn = nil
			}

			_, err := engine.Run()
			if err != nil {
				t.Fatalf("Run: %v", err)
			}

			if tt.validate != nil {
				tt.validate(t, calls)
			}
		})
	}
}

// --- standalone: with secret ---------------------------------------------

func TestEngine_Run_WithSecret(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	home := t.TempDir()
	createOpenCodeFixture(t, home)

	// Add a file with a secret.
	configDir := filepath.Join(home, ".config", "opencode")
	if err := os.WriteFile(filepath.Join(configDir, "config.json"),
		[]byte(`{"github_token":"ghp_abcdef1234567890123456789012345678901234"}`),
		0644); err != nil {
		t.Fatal(err)
	}

	engine := setupTestEngine(t, home)
	engine.Preset = "quick"

	result, err := engine.Run()
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if result.Secrets < 1 {
		t.Error("expected at least 1 secret detected")
	}

	// Check .env.example exists.
	examplePath := filepath.Join(result.BackupDir, ".env.example")
	if _, err := os.Stat(examplePath); err != nil {
		t.Errorf(".env.example not found: %v", err)
	}
}

// --- standalone: backup files exist --------------------------------------

func TestEngine_Run_BackupFilesExist(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	home := t.TempDir()
	createOpenCodeFixture(t, home)

	engine := setupTestEngine(t, home)
	engine.Preset = "full"

	result, err := engine.Run()
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Walk the backup dir and verify files have content.
	fileCount := 0
	err = filepath.WalkDir(result.BackupDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		// Skip manifest.json and .env.example (they're metadata, not backed-up files).
		base := filepath.Base(path)
		if base == "manifest.json" || base == ".env.example" {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if info.Size() == 0 {
			t.Errorf("backed-up file is empty: %s", path)
		}
		fileCount++
		return nil
	})
	if err != nil {
		t.Fatalf("walk backup dir: %v", err)
	}
	if fileCount == 0 {
		t.Error("no backed-up files found in backup dir")
	}
}

// --- standalone: applies excludes ----------------------------------------

// TestEngine_Run_AppliesExcludes verifies that when ExcludesLoader is set,
// the engine calls it and applies ScanOptions to ScanConfigurable adapters.
func TestEngine_Run_AppliesExcludes(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	home := t.TempDir()
	createOpenCodeFixture(t, home)

	engine := setupTestEngine(t, home)
	engine.Preset = "quick"

	excludeCalled := false
	engine.ExcludesLoader = func() (adapters.ScanOptions, error) {
		excludeCalled = true
		return adapters.ScanOptions{
			Excludes:    []string{"node_modules/", "*.log"},
			MaxFileSize: 1048576,
		}, nil
	}

	_, err := engine.Run()
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if !excludeCalled {
		t.Error("ExcludesLoader was not called")
	}
}
