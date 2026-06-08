package pidev

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/danielxxomg/bak-cli/internal/adapters"
)

func TestAdapter_Name(t *testing.T) {
	a := &Adapter{}
	if a.Name() != "pidev" {
		t.Errorf("Name() = %q, want %q", a.Name(), "pidev")
	}
}

func TestAdapter_Detect(t *testing.T) {
	a := &Adapter{}

	t.Run("installed", func(t *testing.T) {
		home := t.TempDir()
		configDir := filepath.Join(home, ".pi")
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

	t.Run("not installed", func(t *testing.T) {
		home := t.TempDir()
		installed, _, err := a.Detect(home)
		if err != nil {
			t.Fatalf("Detect: %v", err)
		}
		if installed {
			t.Error("expected installed=false")
		}
	})

	t.Run("exists but is file not dir", func(t *testing.T) {
		home := t.TempDir()
		configPath := filepath.Join(home, ".pi")
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

func TestAdapter_ListItems(t *testing.T) {
	a := &Adapter{}

	setupHome := func(t *testing.T) string {
		t.Helper()
		home := t.TempDir()
		configDir := filepath.Join(home, ".pi")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte(`{"api":"pi.dev"}`), 0644); err != nil {
			t.Fatal(err)
		}
		// Agents directory
		agentsDir := filepath.Join(configDir, "agents")
		if err := os.MkdirAll(agentsDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(agentsDir, "default.md"), []byte("# Default Agent"), 0644); err != nil {
			t.Fatal(err)
		}
		return home
	}

	t.Run("config category", func(t *testing.T) {
		home := setupHome(t)
		items, err := a.ListItems(home, []string{"config"})
		if err != nil {
			t.Fatalf("ListItems: %v", err)
		}
		if len(items) < 1 {
			t.Fatalf("expected at least 1 config item, got %d", len(items))
		}
	})

	t.Run("agents category", func(t *testing.T) {
		home := setupHome(t)
		items, err := a.ListItems(home, []string{"agents"})
		if err != nil {
			t.Fatalf("ListItems: %v", err)
		}
		if len(items) < 1 {
			t.Fatal("expected at least 1 agents item")
		}
		for _, item := range items {
			if item.Category != "agents" {
				t.Errorf("item %q has category %q, want agents", item.RelPath, item.Category)
			}
		}
	})

	t.Run("all categories", func(t *testing.T) {
		home := setupHome(t)
		items, err := a.ListItems(home, []string{"config", "agents"})
		if err != nil {
			t.Fatalf("ListItems: %v", err)
		}
		if len(items) < 2 {
			t.Fatalf("expected at least 2 items, got %d", len(items))
		}
	})
}

func TestAdapter_Backup(t *testing.T) {
	a := &Adapter{}
	home := t.TempDir()
	configDir := filepath.Join(home, ".pi")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte(`{"api":"pi.dev"}`), 0644); err != nil {
		t.Fatal(err)
	}
	items := []adapters.Item{
		{Category: "config", SourcePath: "~/.pi/config.json", RelPath: "config.json", IsDir: false, Hash: "sha256:abc", Size: 17},
	}
	backupDir := filepath.Join(t.TempDir(), "backup")
	if err := a.Backup(home, backupDir, items); err != nil {
		t.Fatalf("Backup: %v", err)
	}
	dstFile := filepath.Join(backupDir, "pidev", "config.json")
	data, err := os.ReadFile(dstFile)
	if err != nil {
		t.Fatalf("read backup file: %v", err)
	}
	if string(data) != `{"api":"pi.dev"}` {
		t.Errorf("backup content = %q, want %q", string(data), `{"api":"pi.dev"}`)
	}
}

func TestAdapter_Restore(t *testing.T) {
	a := &Adapter{}
	backupDir := filepath.Join(t.TempDir(), "backup")
	backupFile := filepath.Join(backupDir, "pidev", "config.json")
	if err := os.MkdirAll(filepath.Dir(backupFile), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(backupFile, []byte(`{"restored":true}`), 0644); err != nil {
		t.Fatal(err)
	}
	items := []adapters.Item{
		{Category: "config", SourcePath: "~/.pi/config.json", RelPath: "config.json", IsDir: false, Hash: "sha256:xyz", Size: 18},
	}
	home := t.TempDir()
	if err := a.Restore(backupDir, home, items); err != nil {
		t.Fatalf("Restore: %v", err)
	}
	restoredFile := filepath.Join(home, ".pi", "config.json")
	data, err := os.ReadFile(restoredFile)
	if err != nil {
		t.Fatalf("read restored file: %v", err)
	}
	if string(data) != `{"restored":true}` {
		t.Errorf("restored content = %q, want %q", string(data), `{"restored":true}`)
	}
}

func TestAdapter_InterfaceCompliance(t *testing.T) {
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

func TestAdapter_Backup_DirectoryItems(t *testing.T) {
	a := &Adapter{}
	home := t.TempDir()
	configDir := filepath.Join(home, ".pi")
	subDir := filepath.Join(configDir, "myskills")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "code-review.md"), []byte("# Code Review"), 0644); err != nil {
		t.Fatal(err)
	}

	items := []adapters.Item{
		{Category: "skills", SourcePath: "~/.pi/myskills", RelPath: "myskills", IsDir: true},
		{Category: "skills", SourcePath: "~/.pi/myskills/code-review.md", RelPath: "myskills/code-review.md", IsDir: false, Hash: "sha256:abc", Size: 13},
	}

	backupDir := filepath.Join(t.TempDir(), "backup")
	if err := a.Backup(home, backupDir, items); err != nil {
		t.Fatalf("Backup: %v", err)
	}

	dirDst := filepath.Join(backupDir, "pidev", "myskills")
	info, err := os.Stat(dirDst)
	if err != nil {
		t.Fatalf("backup dir not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("backup path should be a directory")
	}

	fileDst := filepath.Join(backupDir, "pidev", "myskills", "code-review.md")
	data, err := os.ReadFile(fileDst)
	if err != nil {
		t.Fatalf("read backup file: %v", err)
	}
	if string(data) != "# Code Review" {
		t.Errorf("backup content = %q, want %q", string(data), "# Code Review")
	}
}

func TestAdapter_Backup_CopyError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod not applicable on Windows")
	}
	a := &Adapter{}
	home := t.TempDir()
	configDir := filepath.Join(home, ".pi")
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
		{Category: "config", SourcePath: "~/.pi/unreadable.txt", RelPath: "unreadable.txt", IsDir: false, Hash: "sha256:abc", Size: 4},
	}

	backupDir := filepath.Join(t.TempDir(), "backup")
	err := a.Backup(home, backupDir, items)
	if err == nil {
		t.Error("expected error for unreadable file, got nil")
	}
}

func TestAdapter_Restore_CopyError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod not applicable on Windows")
	}
	a := &Adapter{}
	home := t.TempDir()
	configDir := filepath.Join(home, ".pi")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(configDir, 0500); err != nil {
		t.Fatal(err)
	}

	backupDir := filepath.Join(t.TempDir(), "backup")
	srcFile := filepath.Join(backupDir, "pidev", "secret.json")
	if err := os.MkdirAll(filepath.Dir(srcFile), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(srcFile, []byte(`{"key":"secret"}`), 0644); err != nil {
		t.Fatal(err)
	}

	items := []adapters.Item{
		{Category: "config", SourcePath: "~/.pi/secret.json", RelPath: "secret.json", IsDir: false, Hash: "sha256:xyz", Size: 16},
	}

	err := a.Restore(backupDir, home, items)
	if err == nil {
		t.Error("expected error for copy to read-only dir, got nil")
	}
}

func TestAdapter_fileHash_Error(t *testing.T) {
	_, _, err := adapters.FileHash(filepath.Join(t.TempDir(), "nonexistent.txt"))
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}
