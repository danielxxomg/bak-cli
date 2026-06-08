package adapters_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/danielxxomg/bak-cli/internal/adapters"
)

// newTestAdapter returns a GenericAdapter configured for testing with
// a ".test" config directory under homeDir.
func newTestAdapter(name string) adapters.GenericAdapter {
	return adapters.GenericAdapter{
		AdapterName:      name,
		ConfigRelPath:    ".test",
		Categories: map[string]adapters.CategoryDir{
			"config":  {SubPath: "", IsDir: false},
			"scripts": {SubPath: "scripts", IsDir: true},
		},
		DetectErrContext: "stat " + name + " config dir",
	}
}

// setupConfigHome creates a home directory with a config directory containing
// root-level files and a scripts subdirectory. Returns the home path.
func setupConfigHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	configDir := filepath.Join(home, ".test")

	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "settings.json"), []byte(`{"theme":"dark"}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.yml"), []byte("model: gpt-4"), 0644); err != nil {
		t.Fatal(err)
	}

	scriptsDir := filepath.Join(configDir, "scripts")
	if err := os.MkdirAll(scriptsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(scriptsDir, "deploy.sh"), []byte("#!/bin/bash\necho deploy"), 0644); err != nil {
		t.Fatal(err)
	}

	return home
}

func TestGenericAdapter_Name(t *testing.T) {
	tests := []struct {
		name        string
		adapterName string
		want        string
	}{
		{name: "codex", adapterName: "codex", want: "codex"},
		{name: "claude-code", adapterName: "claude-code", want: "claude-code"},
		{name: "cursor", adapterName: "cursor", want: "cursor"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ga := newTestAdapter(tt.adapterName)
			if got := ga.Name(); got != tt.want {
				t.Errorf("Name() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGenericAdapter_Detect(t *testing.T) {
	ga := newTestAdapter("test-tool")

	t.Run("installed", func(t *testing.T) {
		home := t.TempDir()
		configDir := filepath.Join(home, ".test")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			t.Fatal(err)
		}

		installed, gotDir, err := ga.Detect(home)
		if err != nil {
			t.Fatalf("Detect: %v", err)
		}
		if !installed {
			t.Error("expected installed=true when dir exists")
		}
		if gotDir != configDir {
			t.Errorf("configDir = %q, want %q", gotDir, configDir)
		}
	})

	t.Run("not installed", func(t *testing.T) {
		home := t.TempDir()

		installed, _, err := ga.Detect(home)
		if err != nil {
			t.Fatalf("Detect: %v", err)
		}
		if installed {
			t.Error("expected installed=false when dir is missing")
		}
	})

	t.Run("exists but is file not dir", func(t *testing.T) {
		home := t.TempDir()
		configPath := filepath.Join(home, ".test")
		if err := os.WriteFile(configPath, []byte("not a dir"), 0644); err != nil {
			t.Fatal(err)
		}

		installed, _, err := ga.Detect(home)
		if err != nil {
			t.Fatalf("Detect: %v", err)
		}
		if installed {
			t.Error("expected installed=false when path is a file")
		}
	})

	t.Run("stat error", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("chmod not applicable on Windows")
		}
		home := t.TempDir()
		configDir := filepath.Join(home, ".test")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(configDir, 0000); err != nil {
			t.Fatal(err)
		}
		defer os.Chmod(configDir, 0755)

		_, _, err := ga.Detect(home)
		if err == nil {
			t.Error("expected error for unreadable dir, got nil")
		}
	})

	t.Run("path traversal blocked", func(t *testing.T) {
		evil := newTestAdapter("evil")
		evil.ConfigRelPath = "../../etc"
		home := t.TempDir()

		_, _, err := evil.Detect(home)
		if err == nil {
			t.Error("expected error for path traversal, got nil")
		}
	})
}

func TestGenericAdapter_ListItems(t *testing.T) {
	ga := newTestAdapter("test-tool")

	t.Run("config category returns root files", func(t *testing.T) {
		home := setupConfigHome(t)
		items, err := ga.ListItems(home, []string{"config"})
		if err != nil {
			t.Fatalf("ListItems: %v", err)
		}
		if len(items) < 2 {
			t.Fatalf("expected at least 2 config items, got %d", len(items))
		}
		for _, item := range items {
			if item.Category != "config" {
				t.Errorf("item %q has category %q, want config", item.RelPath, item.Category)
			}
			if !item.IsDir && item.Hash == "" {
				t.Errorf("file item %q has empty hash", item.RelPath)
			}
		}
	})

	t.Run("scripts category returns dir contents", func(t *testing.T) {
		home := setupConfigHome(t)
		items, err := ga.ListItems(home, []string{"scripts"})
		if err != nil {
			t.Fatalf("ListItems: %v", err)
		}
		if len(items) < 1 {
			t.Fatal("expected at least 1 item from scripts dir")
		}
		for _, item := range items {
			if item.Category != "scripts" {
				t.Errorf("item %q has category %q, want scripts", item.RelPath, item.Category)
			}
		}
	})

	t.Run("rel path uses forward slashes", func(t *testing.T) {
		home := setupConfigHome(t)
		items, err := ga.ListItems(home, []string{"config", "scripts"})
		if err != nil {
			t.Fatalf("ListItems: %v", err)
		}
		if len(items) == 0 {
			t.Fatal("expected non-empty result")
		}
		for _, item := range items {
			if strings.Contains(item.RelPath, "\\") {
				t.Errorf("RelPath %q contains backslash, want forward slashes", item.RelPath)
			}
		}
	})

	t.Run("all categories", func(t *testing.T) {
		home := setupConfigHome(t)
		items, err := ga.ListItems(home, []string{"config", "scripts"})
		if err != nil {
			t.Fatalf("ListItems: %v", err)
		}
		if len(items) < 3 {
			t.Fatalf("expected at least 3 items across categories, got %d", len(items))
		}
	})

	t.Run("empty categories returns empty slice", func(t *testing.T) {
		home := setupConfigHome(t)
		items, err := ga.ListItems(home, []string{})
		if err != nil {
			t.Fatalf("ListItems: %v", err)
		}
		if len(items) != 0 {
			t.Errorf("expected 0 items for empty categories, got %d", len(items))
		}
	})

	t.Run("unknown category returns empty slice", func(t *testing.T) {
		home := setupConfigHome(t)
		items, err := ga.ListItems(home, []string{"nonexistent"})
		if err != nil {
			t.Fatalf("ListItems: %v", err)
		}
		if len(items) != 0 {
			t.Errorf("expected 0 items for unknown category, got %d", len(items))
		}
	})

	t.Run("empty result for missing dirs", func(t *testing.T) {
		home := t.TempDir()
		configDir := filepath.Join(home, ".test")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			t.Fatal(err)
		}
		items, err := ga.ListItems(home, []string{"scripts"})
		if err != nil {
			t.Fatalf("ListItems: %v", err)
		}
		if len(items) != 0 {
			t.Errorf("expected 0 items for missing scripts dir, got %d", len(items))
		}
	})
}

func TestGenericAdapter_Backup(t *testing.T) {
	ga := newTestAdapter("test-tool")

	t.Run("copies file to backup dir", func(t *testing.T) {
		home := t.TempDir()
		configDir := filepath.Join(home, ".test")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(configDir, "settings.json"), []byte(`{"theme":"dark"}`), 0644); err != nil {
			t.Fatal(err)
		}

		items := []adapters.Item{
			{
				Category:   "config",
				SourcePath: "~/.test/settings.json",
				RelPath:    "settings.json",
				IsDir:      false,
				Hash:       "sha256:abc",
				Size:       16,
			},
		}

		backupDir := filepath.Join(t.TempDir(), "backup")
		if err := ga.Backup(home, backupDir, items); err != nil {
			t.Fatalf("Backup: %v", err)
		}

		dstFile := filepath.Join(backupDir, "test-tool", "settings.json")
		data, err := os.ReadFile(dstFile)
		if err != nil {
			t.Fatalf("read backup file: %v", err)
		}
		if string(data) != `{"theme":"dark"}` {
			t.Errorf("backup content = %q, want %q", string(data), `{"theme":"dark"}`)
		}
	})

	t.Run("creates directory for dir items", func(t *testing.T) {
		home := t.TempDir()
		configDir := filepath.Join(home, ".test")
		scriptsDir := filepath.Join(configDir, "scripts")
		if err := os.MkdirAll(scriptsDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(scriptsDir, "deploy.sh"), []byte("#!/bin/bash"), 0644); err != nil {
			t.Fatal(err)
		}

		items := []adapters.Item{
			{Category: "scripts", SourcePath: "~/.test/scripts", RelPath: "scripts", IsDir: true},
			{Category: "scripts", SourcePath: "~/.test/scripts/deploy.sh", RelPath: "scripts/deploy.sh", IsDir: false, Hash: "sha256:abc", Size: 12},
		}

		backupDir := filepath.Join(t.TempDir(), "backup")
		if err := ga.Backup(home, backupDir, items); err != nil {
			t.Fatalf("Backup: %v", err)
		}

		dirDst := filepath.Join(backupDir, "test-tool", "scripts")
		info, err := os.Stat(dirDst)
		if err != nil {
			t.Fatalf("backup dir not created: %v", err)
		}
		if !info.IsDir() {
			t.Error("backup path should be a directory")
		}

		fileDst := filepath.Join(backupDir, "test-tool", "scripts", "deploy.sh")
		data, err := os.ReadFile(fileDst)
		if err != nil {
			t.Fatalf("read backup file: %v", err)
		}
		if string(data) != "#!/bin/bash" {
			t.Errorf("backup content = %q, want %q", string(data), "#!/bin/bash")
		}
	})

	t.Run("copy error on unreadable source", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("chmod not applicable on Windows")
		}
		home := t.TempDir()
		configDir := filepath.Join(home, ".test")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			t.Fatal(err)
		}

		unreadableFile := filepath.Join(configDir, "unreadable.txt")
		if err := os.WriteFile(unreadableFile, []byte("data"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(unreadableFile, 0000); err != nil {
			t.Fatal(err)
		}

		items := []adapters.Item{
			{Category: "config", SourcePath: "~/.test/unreadable.txt", RelPath: "unreadable.txt", IsDir: false, Hash: "sha256:abc", Size: 4},
		}

		backupDir := filepath.Join(t.TempDir(), "backup")
		err := ga.Backup(home, backupDir, items)
		if err == nil {
			t.Error("expected error for unreadable file, got nil")
		}
	})
}

func TestGenericAdapter_Restore(t *testing.T) {
	ga := newTestAdapter("test-tool")

	t.Run("copies file from backup to home", func(t *testing.T) {
		backupDir := filepath.Join(t.TempDir(), "backup")
		backupFile := filepath.Join(backupDir, "test-tool", "settings.json")
		if err := os.MkdirAll(filepath.Dir(backupFile), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(backupFile, []byte(`{"restored":true}`), 0644); err != nil {
			t.Fatal(err)
		}

		items := []adapters.Item{
			{Category: "config", SourcePath: "~/.test/settings.json", RelPath: "settings.json", IsDir: false, Hash: "sha256:xyz", Size: 18},
		}

		home := t.TempDir()
		if err := ga.Restore(backupDir, home, items); err != nil {
			t.Fatalf("Restore: %v", err)
		}

		restoredFile := filepath.Join(home, ".test", "settings.json")
		data, err := os.ReadFile(restoredFile)
		if err != nil {
			t.Fatalf("read restored file: %v", err)
		}
		if string(data) != `{"restored":true}` {
			t.Errorf("restored content = %q, want %q", string(data), `{"restored":true}`)
		}
	})

	t.Run("creates directory for dir items on restore", func(t *testing.T) {
		backupDir := filepath.Join(t.TempDir(), "backup")
		scriptsDir := filepath.Join(backupDir, "test-tool", "scripts")
		if err := os.MkdirAll(scriptsDir, 0755); err != nil {
			t.Fatal(err)
		}

		items := []adapters.Item{
			{Category: "scripts", SourcePath: "~/.test/scripts", RelPath: "scripts", IsDir: true},
		}

		home := t.TempDir()
		if err := ga.Restore(backupDir, home, items); err != nil {
			t.Fatalf("Restore: %v", err)
		}

		restoredDir := filepath.Join(home, ".test", "scripts")
		info, err := os.Stat(restoredDir)
		if err != nil {
			t.Fatalf("restored dir not created: %v", err)
		}
		if !info.IsDir() {
			t.Error("restored path should be a directory")
		}
	})

	t.Run("copy error on restore", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("chmod not applicable on Windows")
		}
		home := t.TempDir()
		configDir := filepath.Join(home, ".test")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(configDir, 0500); err != nil {
			t.Fatal(err)
		}

		backupDir := filepath.Join(t.TempDir(), "backup")
		srcFile := filepath.Join(backupDir, "test-tool", "secret.json")
		if err := os.MkdirAll(filepath.Dir(srcFile), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(srcFile, []byte(`{"key":"secret"}`), 0644); err != nil {
			t.Fatal(err)
		}

		items := []adapters.Item{
			{Category: "config", SourcePath: "~/.test/secret.json", RelPath: "secret.json", IsDir: false, Hash: "sha256:xyz", Size: 16},
		}

		err := ga.Restore(backupDir, home, items)
		if err == nil {
			t.Error("expected error for copy to read-only dir, got nil")
		}
	})
}

func TestGenericAdapter_InterfaceCompliance(t *testing.T) {
	ga := newTestAdapter("iface-test")

	if ga.Name() == "" {
		t.Error("Name should not be empty")
	}

	home := t.TempDir()
	installed, configDir, err := ga.Detect(home)
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
