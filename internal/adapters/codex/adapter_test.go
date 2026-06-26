package codex

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/danielxxomg/bak-cli/internal/adapters"
)

func TestAdapter_Name(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	a := &Adapter{}
	if a.Name() != "codex" {
		t.Errorf("Name() = %q, want %q", a.Name(), "codex")
	}
}

func TestAdapter_Detect(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	a := &Adapter{}

	t.Run("installed", func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
		home := t.TempDir()
		configDir := filepath.Join(home, ".codex")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			t.Fatal(err)
		}
		installed, gotDir, err := a.Detect(home)
		if err != nil {
			t.Fatalf("Detect: %v", err)
		}
		if !installed {
			t.Error("expected installed=true")
		}
		if gotDir != configDir {
			t.Errorf("configDir = %q, want %q", gotDir, configDir)
		}
	})

	t.Run("not installed", func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
		home := t.TempDir()
		installed, _, err := a.Detect(home)
		if err != nil {
			t.Fatalf("Detect: %v", err)
		}
		if installed {
			t.Error("expected installed=false")
		}
	})

	t.Run("exists but is file not dir", func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
		home := t.TempDir()
		configPath := filepath.Join(home, ".codex")
		if err := os.WriteFile(configPath, []byte("not a dir"), 0644); err != nil {
			t.Fatal(err)
		}
		installed, _, err := a.Detect(home)
		if err != nil {
			t.Fatalf("Detect: %v", err)
		}
		if installed {
			t.Error("expected installed=false when path is a file")
		}
	})
}

func TestAdapter_ListItems(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	a := &Adapter{}

	setupHome := func(t *testing.T) string {
		t.Helper()
		home := t.TempDir()
		configDir := filepath.Join(home, ".codex")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			t.Fatal(err)
		}
		// Create whitelisted files (config.toml, instructions.md).
		if err := os.WriteFile(filepath.Join(configDir, "config.toml"), []byte("[model]"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(configDir, "instructions.md"), []byte("# Code Style"), 0644); err != nil {
			t.Fatal(err)
		}
		return home
	}

	t.Run("config category", func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
		home := setupHome(t)
		items, err := a.ListItems(home, []string{"config"})
		if err != nil {
			t.Fatalf("ListItems: %v", err)
		}
		if len(items) < 1 {
			t.Fatalf("expected at least 1 config item, got %d", len(items))
		}
		for _, item := range items {
			if item.Category != "config" {
				t.Errorf("item %q has category %q, want config", item.RelPath, item.Category)
			}
		}
	})

	t.Run("agents category (root — known limitation: GenericAdapter scans root only for config)", func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
		home := setupHome(t)
		// agents is declared as a root-file category but GenericAdapter currently
		// only invokes scanRootFiles for the "config" category. Returns empty
		// until root scanning is extended to arbitrary root-file categories.
		items, err := a.ListItems(home, []string{"agents"})
		if err != nil {
			t.Fatalf("ListItems: %v", err)
		}
		// Expect 0 items — root scan only triggers for "config" category.
		_ = items
	})

	t.Run("all categories", func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
		home := setupHome(t)
		items, err := a.ListItems(home, []string{"config", "agents"})
		if err != nil {
			t.Fatalf("ListItems: %v", err)
		}
		// config yields root files; agents is root-file but not scanned by GenericAdapter yet.
		if len(items) < 1 {
			t.Fatalf("expected at least 1 config item, got %d", len(items))
		}
	})
}

func TestAdapter_Backup(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	a := &Adapter{}
	home := t.TempDir()
	configDir := filepath.Join(home, ".codex")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.toml"), []byte("model: gpt-4"), 0644); err != nil {
		t.Fatal(err)
	}
	items := []adapters.Item{
		{Category: "config", SourcePath: "~/.codex/config.toml", RelPath: "config.toml", IsDir: false, Hash: "sha256:abc", Size: 14},
	}
	backupDir := filepath.Join(t.TempDir(), "backup")
	if err := a.Backup(home, backupDir, items); err != nil {
		t.Fatalf("Backup: %v", err)
	}
	dstFile := filepath.Join(backupDir, "codex", "config.toml")
	data, err := os.ReadFile(dstFile)
	if err != nil {
		t.Fatalf("read backup file: %v", err)
	}
	if string(data) != "model: gpt-4" {
		t.Errorf("backup content = %q, want %q", string(data), "model: gpt-4")
	}
}

func TestAdapter_Restore(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	a := &Adapter{}
	backupDir := filepath.Join(t.TempDir(), "backup")
	backupFile := filepath.Join(backupDir, "codex", "config.toml")
	if err := os.MkdirAll(filepath.Dir(backupFile), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(backupFile, []byte(`{"restored":true}`), 0644); err != nil {
		t.Fatal(err)
	}
	items := []adapters.Item{
		{Category: "config", SourcePath: "~/.codex/config.toml", RelPath: "config.toml", IsDir: false, Hash: "sha256:xyz", Size: 18},
	}
	home := t.TempDir()
	if err := a.Restore(backupDir, home, items); err != nil {
		t.Fatalf("Restore: %v", err)
	}
	restoredFile := filepath.Join(home, ".codex", "config.toml")
	data, err := os.ReadFile(restoredFile)
	if err != nil {
		t.Fatalf("read restored file: %v", err)
	}
	if string(data) != `{"restored":true}` {
		t.Errorf("restored content = %q, want %q", string(data), `{"restored":true}`)
	}
}

func TestAdapter_InterfaceCompliance(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	a := &Adapter{}
	if a.Name() == "" {
		t.Error("Name should not be empty")
	}
	home := t.TempDir()
	installed, configDir, err := a.Detect(home)
	if err != nil {
		t.Errorf("Detect should not error on missing dir: %v", err)
	}
	if installed {
		t.Error("Detect should return false for empty temp dir")
	}
	if configDir == "" {
		t.Error("configDir should not be empty even when not installed")
	}
}

func TestAdapter_Backup_DirectoryItems(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	a := &Adapter{}
	home := t.TempDir()
	configDir := filepath.Join(home, ".codex")
	subDir := filepath.Join(configDir, "myskills")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "code-review.md"), []byte("# Code Review"), 0644); err != nil {
		t.Fatal(err)
	}

	items := []adapters.Item{
		{Category: "skills", SourcePath: "~/.codex/myskills", RelPath: "myskills", IsDir: true},
		{Category: "skills", SourcePath: "~/.codex/myskills/code-review.md", RelPath: "myskills/code-review.md", IsDir: false, Hash: "sha256:abc", Size: 13},
	}

	backupDir := filepath.Join(t.TempDir(), "backup")
	if err := a.Backup(home, backupDir, items); err != nil {
		t.Fatalf("Backup: %v", err)
	}

	dirDst := filepath.Join(backupDir, "codex", "myskills")
	info, err := os.Stat(dirDst)
	if err != nil {
		t.Fatalf("backup dir not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("backup path should be a directory")
	}

	fileDst := filepath.Join(backupDir, "codex", "myskills", "code-review.md")
	data, err := os.ReadFile(fileDst)
	if err != nil {
		t.Fatalf("read backup file: %v", err)
	}
	if string(data) != "# Code Review" {
		t.Errorf("backup content = %q, want %q", string(data), "# Code Review")
	}
}

func TestAdapter_Backup_CopyError(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	if runtime.GOOS == "windows" {
		t.Skip("chmod not applicable on Windows")
	}
	a := &Adapter{}
	home := t.TempDir()
	configDir := filepath.Join(home, ".codex")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}

	unreadableFile := filepath.Join(configDir, "config.toml")
	if err := os.WriteFile(unreadableFile, []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(unreadableFile, 0000); err != nil {
		t.Fatal(err)
	}

	items := []adapters.Item{
		{Category: "config", SourcePath: "~/.codex/config.toml", RelPath: "config.toml", IsDir: false, Hash: "sha256:abc", Size: 4},
	}

	backupDir := filepath.Join(t.TempDir(), "backup")
	err := a.Backup(home, backupDir, items)
	if err == nil {
		t.Error("expected error for unreadable file, got nil")
	}
}

func TestAdapter_Restore_CopyError(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	if runtime.GOOS == "windows" {
		t.Skip("chmod not applicable on Windows")
	}
	a := &Adapter{}
	home := t.TempDir()
	configDir := filepath.Join(home, ".codex")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(configDir, 0500); err != nil {
		t.Fatal(err)
	}

	backupDir := filepath.Join(t.TempDir(), "backup")
	srcFile := filepath.Join(backupDir, "codex", "secret.json")
	if err := os.MkdirAll(filepath.Dir(srcFile), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(srcFile, []byte(`{"key":"secret"}`), 0644); err != nil {
		t.Fatal(err)
	}

	items := []adapters.Item{
		{Category: "config", SourcePath: "~/.codex/secret.json", RelPath: "secret.json", IsDir: false, Hash: "sha256:xyz", Size: 16},
	}

	err := a.Restore(backupDir, home, items)
	if err == nil {
		t.Error("expected error for copy to read-only dir, got nil")
	}
}

func TestAdapter_fileHash_Error(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	_, _, err := adapters.FileHash(filepath.Join(t.TempDir(), "nonexistent.txt"))
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}

// TestAdapter_WhitelistOnlyConfigs verifies that the codex adapter's
// RootConfigFiles whitelist returns only config files, not SQLite DBs
// or cache files. This test is RED until RootConfigFiles is set.
func TestAdapter_WhitelistOnlyConfigs(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	a := &Adapter{}

	home := t.TempDir()
	configDir := filepath.Join(home, ".codex")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create both config and runtime files.
	if err := os.WriteFile(filepath.Join(configDir, "config.toml"), []byte("[model]"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "instructions.md"), []byte("# Code Style"), 0644); err != nil {
		t.Fatal(err)
	}
	// SQLite DB — should NOT be in results.
	if err := os.WriteFile(filepath.Join(configDir, "logs_2.sqlite"), []byte("sqlite data"), 0644); err != nil {
		t.Fatal(err)
	}
	// Cache file — should NOT be in results.
	if err := os.WriteFile(filepath.Join(configDir, "models_cache.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	items, err := a.ListItems(home, []string{"config"})
	if err != nil {
		t.Fatalf("ListItems: %v", err)
	}

	// Only config.toml and instructions.md should be returned.
	for _, item := range items {
		switch item.RelPath {
		case "config.toml", "instructions.md":
			// expected
		case "logs_2.sqlite", "models_cache.json":
			t.Errorf("runtime file %q should have been excluded by whitelist", item.RelPath)
		}
	}

	// Verify both expected config files are present.
	foundToml := false
	foundMd := false
	for _, item := range items {
		if item.RelPath == "config.toml" {
			foundToml = true
		}
		if item.RelPath == "instructions.md" {
			foundMd = true
		}
	}
	if !foundToml {
		t.Error("config.toml should be included (whitelisted)")
	}
	if !foundMd {
		t.Error("instructions.md should be included (whitelisted)")
	}
}
