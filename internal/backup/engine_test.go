package backup

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/danielxxomg/bak-cli/internal/adapters"
	opencodeadapter "github.com/danielxxomg/bak-cli/internal/adapters/opencode"
)

func TestBakDir(t *testing.T) {
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

func TestEngine_Run_QuickPreset(t *testing.T) {
	home := t.TempDir()
	createOpenCodeFixture(t, home)

	engine := setupTestEngine(t, home)
	engine.Preset = "quick" // config-only

	result, err := engine.Run()
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

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
}

func TestEngine_Run_FullPreset(t *testing.T) {
	home := t.TempDir()
	createOpenCodeFixture(t, home)

	engine := setupTestEngine(t, home)
	engine.Preset = "full"

	result, err := engine.Run()
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if result.FileCount < 4 {
		t.Errorf("expected at least 4 files for full preset, got %d", result.FileCount)
	}
}

func TestEngine_Run_WithSecret(t *testing.T) {
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

func TestEngine_Run_NoAdaptersDetected(t *testing.T) {
	home := t.TempDir()
	// No OpenCode dir — adapter won't detect.

	engine := setupTestEngine(t, home)
	_, err := engine.Run()
	if err == nil {
		t.Error("expected error when no adapters detected")
	}
}

func TestEngine_Run_AdapterFilter(t *testing.T) {
	home := t.TempDir()
	createOpenCodeFixture(t, home)

	engine := setupTestEngine(t, home)
	engine.AdapterFilter = []string{"opencode"}

	result, err := engine.Run()
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.AdaptersRun != 1 {
		t.Errorf("AdaptersRun = %d, want 1", result.AdaptersRun)
	}
}

func TestEngine_Run_InvalidAdapterFilter(t *testing.T) {
	home := t.TempDir()
	createOpenCodeFixture(t, home)

	engine := setupTestEngine(t, home)
	engine.AdapterFilter = []string{"nonexistent"}

	_, err := engine.Run()
	if err == nil {
		t.Error("expected error for unknown adapter")
	}
}

func TestEngine_Run_AdapterFilterSlice(t *testing.T) {
	home := t.TempDir()
	createOpenCodeFixture(t, home)

	engine := setupTestEngine(t, home)
	engine.AdapterFilter = []string{"opencode"}

	result, err := engine.Run()
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.AdaptersRun != 1 {
		t.Errorf("AdaptersRun = %d, want 1", result.AdaptersRun)
	}
}

func TestEngine_Run_MultiAdapterFilter(t *testing.T) {
	home := t.TempDir()
	createOpenCodeFixture(t, home)

	engine := setupTestEngine(t, home)
	// Filter for opencode (installed) and a non-existent adapter should
	// error because the second adapter is not registered.
	engine.AdapterFilter = []string{"opencode", "nonexistent"}

	_, err := engine.Run()
	if err == nil {
		t.Error("expected error for unknown adapter in multi-filter")
	}
}

func TestEngine_Run_InvalidPreset(t *testing.T) {
	home := t.TempDir()
	createOpenCodeFixture(t, home)

	engine := setupTestEngine(t, home)
	engine.Preset = "bananas"

	_, err := engine.Run()
	if err == nil {
		t.Error("expected error for unknown preset")
	}
}

func TestEngine_Run_BackupFilesExist(t *testing.T) {
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


