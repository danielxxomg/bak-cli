package manifest

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	m := New("20260604-test", "linux", "testbox", "0.1.0", "full", []string{"skills", "config"})

	if m.Version != ManifestVersion {
		t.Errorf("version = %q, want %q", m.Version, ManifestVersion)
	}
	if m.ID != "20260604-test" {
		t.Errorf("id = %q, want %q", m.ID, "20260604-test")
	}
	if m.OSSource != "linux" {
		t.Errorf("os_source = %q, want %q", m.OSSource, "linux")
	}
	if m.Preset != "full" {
		t.Errorf("preset = %q, want %q", m.Preset, "full")
	}
	if len(m.Categories) != 2 {
		t.Errorf("categories len = %d, want 2", len(m.Categories))
	}
	if m.CreatedAt.After(time.Now()) {
		t.Error("created_at is in the future")
	}
	if m.Adapters == nil {
		t.Error("adapters map is nil")
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()

	m := New("20260604-roundtrip", "darwin", "macbook", "0.1.0", "quick", []string{"config"})
	m.AddAdapter("opencode", "1.5.0", "~/.config/opencode", []Item{
		{Category: "config", SourcePath: "~/.config/opencode/config.json", BackupPath: "opencode/config.json", Hash: "sha256:abc123", Size: 512},
	})

	if err := m.Save(dir); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.ID != m.ID {
		t.Errorf("id = %q, want %q", loaded.ID, m.ID)
	}
	if loaded.Preset != m.Preset {
		t.Errorf("preset = %q, want %q", loaded.Preset, m.Preset)
	}
	if loaded.FileCount != 1 {
		t.Errorf("file_count = %d, want 1", loaded.FileCount)
	}
	if loaded.TotalSize != 512 {
		t.Errorf("total_size = %d, want 512", loaded.TotalSize)
	}

	am, ok := loaded.Adapters["opencode"]
	if !ok {
		t.Fatal("opencode adapter not found in loaded manifest")
	}
	if am.VersionDetected != "1.5.0" {
		t.Errorf("version_detected = %q, want %q", am.VersionDetected, "1.5.0")
	}
	if len(am.Items) != 1 {
		t.Fatalf("items len = %d, want 1", len(am.Items))
	}
	if am.Items[0].Hash != "sha256:abc123" {
		t.Errorf("hash = %q, want %q", am.Items[0].Hash, "sha256:abc123")
	}
}

func TestValidate_HappyPath(t *testing.T) {
	dir := t.TempDir()

	// Create a real file with known content.
	backupFile := filepath.Join(dir, "opencode", "config.json")
	if err := os.MkdirAll(filepath.Dir(backupFile), 0755); err != nil {
		t.Fatal(err)
	}
	content := []byte(`{"key": "value"}`)
	if err := os.WriteFile(backupFile, content, 0644); err != nil {
		t.Fatal(err)
	}

	m := New("validate-test", "linux", "box", "0.1.0", "full", []string{"config"})
	m.AddAdapter("opencode", "1.0.0", "~/.config/opencode", []Item{
		{Category: "config", SourcePath: "~/.config/opencode/config.json", BackupPath: "opencode/config.json", Hash: "sha256:9724c1e20e6e3e4d7f57ed25f9d4efb006e508590d528c90da597f6a775c13e5", Size: 16},
	})

	if err := m.Validate(dir); err != nil {
		t.Errorf("Validate: unexpected error: %v", err)
	}
}

func TestValidate_HashMismatch(t *testing.T) {
	dir := t.TempDir()

	backupFile := filepath.Join(dir, "opencode", "config.json")
	if err := os.MkdirAll(filepath.Dir(backupFile), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(backupFile, []byte(`wrong`), 0644); err != nil {
		t.Fatal(err)
	}

	m := New("mismatch-test", "linux", "box", "0.1.0", "full", []string{"config"})
	m.AddAdapter("opencode", "1.0.0", "~/.config/opencode", []Item{
		{Category: "config", SourcePath: "~/.config/opencode/config.json", BackupPath: "opencode/config.json", Hash: "sha256:0000000000000000000000000000000000000000000000000000000000000000", Size: 5},
	})

	err := m.Validate(dir)
	if err == nil {
		t.Error("Validate: expected hash mismatch error, got nil")
	}
}

func TestValidate_EmptyVersion(t *testing.T) {
	m := &Manifest{Adapters: map[string]AdapterManifest{"x": {}}}
	if err := m.Validate("."); err == nil {
		t.Error("expected error for empty version")
	}
}

func TestValidate_NoAdapters(t *testing.T) {
	m := &Manifest{Version: "1.0.0"}
	if err := m.Validate("."); err == nil {
		t.Error("expected error for no adapters")
	}
}

func TestLoad_NonExistent(t *testing.T) {
	_, err := Load(t.TempDir())
	if err == nil {
		t.Error("expected error for missing manifest.json")
	}
}
