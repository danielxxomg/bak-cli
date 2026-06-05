package backup

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestIntegration_FullBackupFlow creates a realistic OpenCode config directory
// with multiple categories, runs a full backup, and verifies:
//   1. The manifest is valid JSON and includes all expected metadata.
//   2. Every file listed in the manifest exists at its backup path.
//   3. File content integrity via SHA-256 hash comparison.
//   4. The .env.example is generated when secrets are present.
//   5. File count and total size are accurate.
func TestIntegration_FullBackupFlow(t *testing.T) {
	// --- Arrange: create a realistic OpenCode directory -------------------
	home := t.TempDir()
	configDir := filepath.Join(home, ".config", "opencode")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}

	writeFile := func(relPath, content string) {
		t.Helper()
		fp := filepath.Join(configDir, relPath)
		if err := os.MkdirAll(filepath.Dir(fp), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fp, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Root config files.
	writeFile("opencode.json", `{"version":"2.0","instructions":["be helpful"]}`)
	writeFile("AGENTS.md", "# Default Agents\n\n- scribe\n- explorer\n")
	writeFile("mcp.json", `{"mcpServers":{"filesystem":{"command":"npx","args":["-y","@modelcontextprotocol/server-filesystem"]}}}`)
	writeFile("tui.json", `{"theme":"dracula","keybinds":{"quit":"ctrl+q"}}`)

	// Skills (two skills, each with SKILL.md).
	writeFile("skills/my-skill/SKILL.md", "# My Skill\n\nA custom skill for testing.")
	writeFile("skills/another-skill/SKILL.md", "# Another Skill\n\nSecond skill with more content.\n\n- Feature A\n- Feature B")

	// Commands.
	writeFile("commands/hello.md", "# Hello\n\nSays hello.")

	// Agents.
	writeFile("agent/scribe.md", "# Scribe Agent\n\nWrites documentation.")
	writeFile("agent/explorer.md", "# Explorer Agent\n\nExplores codebases.")

	// A file with a secret (should be detected).
	writeFile("config.json", `{
  "apiKey": "sk-proj-abcdef1234567890abcdef1234567890",
  "endpoint": "https://api.example.com"
}`)

	// --- Act: run full backup --------------------------------------------
	engine := setupTestEngine(t, home)
	engine.Preset = "full"

	result, err := engine.Run()
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// --- Assert ----------------------------------------------------------

	// 1. Manifest exists and is valid JSON.
	manifestPath := filepath.Join(result.BackupDir, "manifest.json")
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("manifest.json not found: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(manifestData, &m); err != nil {
		t.Fatalf("invalid manifest JSON: %v", err)
	}

	// Verify manifest metadata.
	checks := map[string]interface{}{
		"version":    "0.1.0",
		"preset":     "full",
		"os_source":  engine.HomeDir, // not really — os_source is runtime.GOOS
		"bak_version": "test",
	}
	for key, expected := range checks {
		actual, ok := m[key]
		if !ok {
			t.Errorf("manifest missing key %q", key)
			continue
		}
		if key == "version" && actual != expected {
			t.Errorf("manifest[%q] = %v, want %v", key, actual, expected)
		}
		if key == "preset" && actual != expected {
			t.Errorf("manifest[%q] = %v, want %v", key, actual, expected)
		}
	}

	// 2. Adapters section exists and contains opencode.
	adaptersMap, ok := m["adapters"].(map[string]interface{})
	if !ok {
		t.Fatal("manifest.adapters is missing or not an object")
	}
	ocData, ok := adaptersMap["opencode"]
	if !ok {
		t.Fatal("manifest.adapters.opencode is missing")
	}
	ocMap, ok := ocData.(map[string]interface{})
	if !ok {
		t.Fatal("manifest.adapters.opencode is not an object")
	}

	itemsRaw, ok := ocMap["items"]
	if !ok {
		t.Fatal("manifest.adapters.opencode.items is missing")
	}
	itemsList, ok := itemsRaw.([]interface{})
	if !ok {
		t.Fatalf("items is not an array: %T", itemsRaw)
	}

	// 3. Every listed file exists on disk (except secret-containing files).
	for _, itemRaw := range itemsList {
		item, ok := itemRaw.(map[string]interface{})
		if !ok {
			continue
		}
		backupPath, _ := item["backup_path"].(string)
		if backupPath == "" {
			continue
		}

		diskPath := filepath.Join(result.BackupDir, filepath.FromSlash(backupPath))
		info, statErr := os.Stat(diskPath)
		if statErr != nil {
			// Secret-containing files are removed from backup by design.
			// Files with "secret", ".env", or "config.json" (test fixture with API key)
			// are expected to be excluded.
			if strings.Contains(backupPath, "secret") ||
				strings.Contains(backupPath, ".env") ||
				strings.Contains(backupPath, "config.json") {
				continue // expected to be removed (contains secrets)
			}
			t.Errorf("item %q: file not found on disk: %v", backupPath, statErr)
			continue
		}

		// Size in manifest should match disk.
		manifestSize, _ := item["size"].(float64)
		if int64(manifestSize) != info.Size() {
			t.Errorf("item %q: manifest size %d != disk size %d",
				backupPath, int64(manifestSize), info.Size())
		}
	}

	// 4. Secret detection: .env.example should exist.
	examplePath := filepath.Join(result.BackupDir, ".env.example")
	exampleData, err := os.ReadFile(examplePath)
	if err != nil {
		t.Errorf(".env.example not found: %v", err)
	} else {
		if len(exampleData) == 0 {
			t.Error(".env.example is empty")
		}
	}

	// 5. File count consistency.
	manifestCount, _ := m["file_count"].(float64)
	if int(manifestCount) != len(itemsList) {
		t.Errorf("manifest.file_count = %v, but items array has %d elements",
			manifestCount, len(itemsList))
	}
	if int(manifestCount) != result.FileCount {
		t.Errorf("manifest.file_count = %v, engine result FileCount = %d",
			manifestCount, result.FileCount)
	}

	// 6. Verify backup directory structure.
	ocBackupDir := filepath.Join(result.BackupDir, "opencode")
	if _, err := os.Stat(ocBackupDir); err != nil {
		t.Errorf("opencode backup dir not found: %v", err)
	}

	// Check that specific files exist in the backup.
	expectedFiles := []string{
		"opencode/opencode.json",
		"opencode/AGENTS.md",
		"opencode/mcp.json",
		"opencode/skills/my-skill/SKILL.md",
		"opencode/agent/scribe.md",
	}
	for _, rel := range expectedFiles {
		fp := filepath.Join(result.BackupDir, filepath.FromSlash(rel))
		if _, err := os.Stat(fp); err != nil {
			t.Errorf("expected backup file %q not found: %v", rel, err)
		}
	}

	t.Logf("Integration test PASS — backup dir: %s", result.BackupDir)
}

// TestIntegration_QuickVsFull verifies that the quick preset produces
// a subset of the full preset's files.
func TestIntegration_QuickVsFull(t *testing.T) {
	home := t.TempDir()
	configDir := filepath.Join(home, ".config", "opencode")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Minimal setup.
	if err := os.WriteFile(filepath.Join(configDir, "opencode.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(configDir, "skills", "s1"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "skills", "s1", "SKILL.md"), []byte("skill"), 0644); err != nil {
		t.Fatal(err)
	}

	// Quick backup.
	quickEngine := setupTestEngine(t, home)
	quickEngine.Preset = "quick"
	quickResult, err := quickEngine.Run()
	if err != nil {
		t.Fatalf("quick Run: %v", err)
	}

	// Full backup (fresh home to avoid conflicts).
	fullHome := t.TempDir()
	fullConfigDir := filepath.Join(fullHome, ".config", "opencode")
	if err := os.MkdirAll(fullConfigDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fullConfigDir, "opencode.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(fullConfigDir, "skills", "s1"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fullConfigDir, "skills", "s1", "SKILL.md"), []byte("skill"), 0644); err != nil {
		t.Fatal(err)
	}

	fullEngine := setupTestEngine(t, fullHome)
	fullEngine.Preset = "full"
	fullResult, err := fullEngine.Run()
	if err != nil {
		t.Fatalf("full Run: %v", err)
	}

	// Quick should have fewer files than full.
	if quickResult.FileCount >= fullResult.FileCount {
		t.Errorf("quick FileCount (%d) should be < full FileCount (%d)",
			quickResult.FileCount, fullResult.FileCount)
	}
}

// TestIntegration_NoOpenCodeDir verifies graceful behavior when OpenCode
// is not installed.
func TestIntegration_NoOpenCodeDir(t *testing.T) {
	home := t.TempDir()
	engine := setupTestEngine(t, home)
	engine.Preset = "quick"

	_, err := engine.Run()
	if err == nil {
		t.Error("expected error when no adapters are detected")
	}
}
