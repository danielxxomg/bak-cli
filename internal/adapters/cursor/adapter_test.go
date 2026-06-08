package cursor

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/danielxxomg/bak-cli/internal/adapters"
)

func TestAdapter_Name(t *testing.T) {
	a := &Adapter{}
	if a.Name() != "cursor" {
		t.Errorf("Name() = %q, want %q", a.Name(), "cursor")
	}
}

func TestAdapter_Detect(t *testing.T) {
	a := &Adapter{}

	t.Run("installed", func(t *testing.T) {
		home := t.TempDir()
		configDir := filepath.Join(home, ".cursor")
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
		configPath := filepath.Join(home, ".cursor")
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
		configDir := filepath.Join(home, ".cursor")

		if err := os.MkdirAll(configDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(configDir, "settings.json"), []byte(`{"editor":"cursor"}`), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(configDir, "keybindings.json"), []byte(`[]`), 0644); err != nil {
			t.Fatal(err)
		}

		// Extensions directory
		extDir := filepath.Join(configDir, "extensions")
		if err := os.MkdirAll(extDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(extDir, "extensions.json"), []byte(`["ext1","ext2"]`), 0644); err != nil {
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
		if len(items) < 2 {
			t.Fatalf("expected at least 2 config items, got %d", len(items))
		}
		for _, item := range items {
			if item.Category != "config" {
				t.Errorf("item %q has category %q, want config", item.RelPath, item.Category)
			}
		}
	})

	t.Run("extensions category", func(t *testing.T) {
		home := setupHome(t)
		items, err := a.ListItems(home, []string{"extensions"})
		if err != nil {
			t.Fatalf("ListItems: %v", err)
		}
		if len(items) < 1 {
			t.Fatal("expected at least 1 extensions item")
		}
		for _, item := range items {
			if item.Category != "extensions" {
				t.Errorf("item %q has category %q, want extensions", item.RelPath, item.Category)
			}
		}
	})

	t.Run("all categories", func(t *testing.T) {
		home := setupHome(t)
		items, err := a.ListItems(home, []string{"config", "extensions"})
		if err != nil {
			t.Fatalf("ListItems: %v", err)
		}
		if len(items) < 3 {
			t.Fatalf("expected at least 3 items across all categories, got %d", len(items))
		}
	})
}

func TestAdapter_Backup(t *testing.T) {
	a := &Adapter{}

	home := t.TempDir()
	configDir := filepath.Join(home, ".cursor")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}

	srcFile := filepath.Join(configDir, "settings.json")
	if err := os.WriteFile(srcFile, []byte(`{"editor":"cursor"}`), 0644); err != nil {
		t.Fatal(err)
	}

	items := []adapters.Item{
		{
			Category:   "config",
			SourcePath: "~/.cursor/settings.json",
			RelPath:    "settings.json",
			IsDir:      false,
			Hash:       "sha256:abc",
			Size:       18,
		},
	}

	backupDir := filepath.Join(t.TempDir(), "backup")
	if err := a.Backup(home, backupDir, items); err != nil {
		t.Fatalf("Backup: %v", err)
	}

	dstFile := filepath.Join(backupDir, "cursor", "settings.json")
	data, err := os.ReadFile(dstFile)
	if err != nil {
		t.Fatalf("read backup file: %v", err)
	}
	if string(data) != `{"editor":"cursor"}` {
		t.Errorf("backup content = %q, want %q", string(data), `{"editor":"cursor"}`)
	}
}

func TestAdapter_Restore(t *testing.T) {
	a := &Adapter{}

	backupDir := filepath.Join(t.TempDir(), "backup")
	backupFile := filepath.Join(backupDir, "cursor", "settings.json")
	if err := os.MkdirAll(filepath.Dir(backupFile), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(backupFile, []byte(`{"restored":true}`), 0644); err != nil {
		t.Fatal(err)
	}

	items := []adapters.Item{
		{
			Category:   "config",
			SourcePath: "~/.cursor/settings.json",
			RelPath:    "settings.json",
			IsDir:      false,
			Hash:       "sha256:xyz",
			Size:       18,
		},
	}

	home := t.TempDir()
	if err := a.Restore(backupDir, home, items); err != nil {
		t.Fatalf("Restore: %v", err)
	}

	restoredFile := filepath.Join(home, ".cursor", "settings.json")
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
	configDir := filepath.Join(home, ".cursor")
	subDir := filepath.Join(configDir, "myskills")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "code-review.md"), []byte("# Code Review"), 0644); err != nil {
		t.Fatal(err)
	}

	items := []adapters.Item{
		{Category: "skills", SourcePath: "~/.cursor/myskills", RelPath: "myskills", IsDir: true},
		{Category: "skills", SourcePath: "~/.cursor/myskills/code-review.md", RelPath: "myskills/code-review.md", IsDir: false, Hash: "sha256:abc", Size: 13},
	}

	backupDir := filepath.Join(t.TempDir(), "backup")
	if err := a.Backup(home, backupDir, items); err != nil {
		t.Fatalf("Backup: %v", err)
	}

	dirDst := filepath.Join(backupDir, "cursor", "myskills")
	info, err := os.Stat(dirDst)
	if err != nil {
		t.Fatalf("backup dir not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("backup path should be a directory")
	}

	fileDst := filepath.Join(backupDir, "cursor", "myskills", "code-review.md")
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
	configDir := filepath.Join(home, ".cursor")
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
		{Category: "config", SourcePath: "~/.cursor/unreadable.txt", RelPath: "unreadable.txt", IsDir: false, Hash: "sha256:abc", Size: 4},
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
	configDir := filepath.Join(home, ".cursor")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(configDir, 0500); err != nil {
		t.Fatal(err)
	}

	backupDir := filepath.Join(t.TempDir(), "backup")
	srcFile := filepath.Join(backupDir, "cursor", "secret.json")
	if err := os.MkdirAll(filepath.Dir(srcFile), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(srcFile, []byte(`{"key":"secret"}`), 0644); err != nil {
		t.Fatal(err)
	}

	items := []adapters.Item{
		{Category: "config", SourcePath: "~/.cursor/secret.json", RelPath: "secret.json", IsDir: false, Hash: "sha256:xyz", Size: 16},
	}

	err := a.Restore(backupDir, home, items)
	if err == nil {
		t.Error("expected error for copy to read-only dir, got nil")
	}
}

func TestAdapter_fileHash_Error(t *testing.T) {
	_, _, err := fileHash(filepath.Join(t.TempDir(), "nonexistent.txt"))
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}
