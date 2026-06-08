package opencode

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/danielxxomg/bak-cli/internal/adapters"
)

func TestAdapter_Name(t *testing.T) {
	a := &Adapter{}
	if a.Name() != "opencode" {
		t.Errorf("Name() = %q, want %q", a.Name(), "opencode")
	}
}

func TestAdapter_Detect(t *testing.T) {
	a := &Adapter{}

	t.Run("installed", func(t *testing.T) {
		home := t.TempDir()
		configDir := filepath.Join(home, ".config", "opencode")
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
		configPath := filepath.Join(home, ".config", "opencode")
		if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
			t.Fatal(err)
		}
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
		configDir := filepath.Join(home, ".config", "opencode")

		// Create skills directory with a skill.
		skillDir := filepath.Join(configDir, "skills", "my-skill")
		if err := os.MkdirAll(skillDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# My Skill"), 0644); err != nil {
			t.Fatal(err)
		}

		// Create commands directory.
		cmdDir := filepath.Join(configDir, "commands")
		if err := os.MkdirAll(cmdDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(cmdDir, "hello.md"), []byte("hello"), 0644); err != nil {
			t.Fatal(err)
		}

		// Create agents directory.
		agentDir := filepath.Join(configDir, "agent")
		if err := os.MkdirAll(agentDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(agentDir, "scribe.md"), []byte("scribe agent"), 0644); err != nil {
			t.Fatal(err)
		}

		// Create root config files.
		if err := os.WriteFile(filepath.Join(configDir, "opencode.json"), []byte(`{"version":"1.0"}`), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(configDir, "AGENTS.md"), []byte("# Agents"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(configDir, "mcp.json"), []byte(`{"servers":{}}`), 0644); err != nil {
			t.Fatal(err)
		}

		return home
	}

	t.Run("skills category", func(t *testing.T) {
		home := setupHome(t)
		items, err := a.ListItems(home, []string{"skills"})
		if err != nil {
			t.Fatalf("ListItems: %v", err)
		}
		if len(items) < 1 {
			t.Fatal("expected at least 1 skill item")
		}
		for _, item := range items {
			if item.Category != "skills" {
				t.Errorf("item %q has category %q, want skills", item.RelPath, item.Category)
			}
			if !item.IsDir && item.Hash == "" {
				t.Errorf("file item %q has empty hash", item.RelPath)
			}
		}
	})

	t.Run("commands category", func(t *testing.T) {
		home := setupHome(t)
		items, err := a.ListItems(home, []string{"commands"})
		if err != nil {
			t.Fatalf("ListItems: %v", err)
		}
		if len(items) < 1 {
			t.Fatal("expected at least 1 command item")
		}
	})

	t.Run("config category", func(t *testing.T) {
		home := setupHome(t)
		items, err := a.ListItems(home, []string{"config"})
		if err != nil {
			t.Fatalf("ListItems: %v", err)
		}
		// Should find opencode.json and AGENTS.md
		if len(items) < 2 {
			t.Fatalf("expected at least 2 config items, got %d", len(items))
		}
		for _, item := range items {
			if item.Category != "config" {
				t.Errorf("item %q has category %q, want config", item.RelPath, item.Category)
			}
		}
	})

	t.Run("mcp category", func(t *testing.T) {
		home := setupHome(t)
		items, err := a.ListItems(home, []string{"mcp"})
		if err != nil {
			t.Fatalf("ListItems: %v", err)
		}
		if len(items) != 1 {
			t.Fatalf("expected 1 mcp item, got %d", len(items))
		}
		if items[0].RelPath != "mcp.json" {
			t.Errorf("RelPath = %q, want mcp.json", items[0].RelPath)
		}
	})

	t.Run("agents category", func(t *testing.T) {
		home := setupHome(t)
		items, err := a.ListItems(home, []string{"agents"})
		if err != nil {
			t.Fatalf("ListItems: %v", err)
		}
		if len(items) < 1 {
			t.Fatal("expected at least 1 agent item")
		}
	})

	t.Run("all categories", func(t *testing.T) {
		home := setupHome(t)
		items, err := a.ListItems(home, []string{"skills", "commands", "config", "mcp", "agents"})
		if err != nil {
			t.Fatalf("ListItems: %v", err)
		}
		if len(items) < 7 {
			t.Fatalf("expected at least 7 items across all categories, got %d", len(items))
		}
	})

	t.Run("empty result for missing dirs", func(t *testing.T) {
		home := t.TempDir()
		// Create only the config dir, but no subdirs or files.
		configDir := filepath.Join(home, ".config", "opencode")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			t.Fatal(err)
		}
		items, err := a.ListItems(home, []string{"plugins"})
		if err != nil {
			t.Fatalf("ListItems: %v", err)
		}
		if len(items) != 0 {
			t.Errorf("expected 0 items for missing plugins dir, got %d", len(items))
		}
	})
}

func TestAdapter_Backup(t *testing.T) {
	a := &Adapter{}

	home := t.TempDir()
	configDir := filepath.Join(home, ".config", "opencode")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a test file.
	srcFile := filepath.Join(configDir, "opencode.json")
	if err := os.WriteFile(srcFile, []byte(`{"key":"value"}`), 0644); err != nil {
		t.Fatal(err)
	}

	items := []adapters.Item{
		{
			Category:   "config",
			SourcePath: "~/.config/opencode/opencode.json",
			RelPath:    "opencode.json",
			IsDir:      false,
			Hash:       "sha256:abc",
			Size:       16,
		},
	}

	backupDir := filepath.Join(t.TempDir(), "backup")
	if err := a.Backup(home, backupDir, items); err != nil {
		t.Fatalf("Backup: %v", err)
	}

	// Verify file was copied.
	dstFile := filepath.Join(backupDir, "opencode", "opencode.json")
	data, err := os.ReadFile(dstFile)
	if err != nil {
		t.Fatalf("read backup file: %v", err)
	}
	if string(data) != `{"key":"value"}` {
		t.Errorf("backup content = %q, want %q", string(data), `{"key":"value"}`)
	}
}

func TestAdapter_Restore(t *testing.T) {
	a := &Adapter{}

	// Set up backup data.
	backupDir := filepath.Join(t.TempDir(), "backup")
	backupFile := filepath.Join(backupDir, "opencode", "opencode.json")
	if err := os.MkdirAll(filepath.Dir(backupFile), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(backupFile, []byte(`{"restored":true}`), 0644); err != nil {
		t.Fatal(err)
	}

	items := []adapters.Item{
		{
			Category:   "config",
			SourcePath: "~/.config/opencode/opencode.json",
			RelPath:    "opencode.json",
			IsDir:      false,
			Hash:       "sha256:xyz",
			Size:       18,
		},
	}

	home := t.TempDir()
	if err := a.Restore(backupDir, home, items); err != nil {
		t.Fatalf("Restore: %v", err)
	}

	// Verify file was restored.
	restoredFile := filepath.Join(home, ".config", "opencode", "opencode.json")
	data, err := os.ReadFile(restoredFile)
	if err != nil {
		t.Fatalf("read restored file: %v", err)
	}
	if string(data) != `{"restored":true}` {
		t.Errorf("restored content = %q", string(data))
	}
}

func TestAdapter_InterfaceCompliance(t *testing.T) {
	// Compile-time check: var _ adapters.Adapter = (*Adapter)(nil) is in adapter.go.
	// This test confirms runtime behavior matches expectations.
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
	configDir := filepath.Join(home, ".config", "opencode")
	subDir := filepath.Join(configDir, "myskills")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "code-review.md"), []byte("# Code Review"), 0644); err != nil {
		t.Fatal(err)
	}

	items := []adapters.Item{
		{Category: "skills", SourcePath: "~/.config/opencode/myskills", RelPath: "myskills", IsDir: true},
		{Category: "skills", SourcePath: "~/.config/opencode/myskills/code-review.md", RelPath: "myskills/code-review.md", IsDir: false, Hash: "sha256:abc", Size: 13},
	}

	backupDir := filepath.Join(t.TempDir(), "backup")
	if err := a.Backup(home, backupDir, items); err != nil {
		t.Fatalf("Backup: %v", err)
	}

	dirDst := filepath.Join(backupDir, "opencode", "myskills")
	info, err := os.Stat(dirDst)
	if err != nil {
		t.Fatalf("backup dir not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("backup path should be a directory")
	}

	fileDst := filepath.Join(backupDir, "opencode", "myskills", "code-review.md")
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
	configDir := filepath.Join(home, ".config", "opencode")
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
		{Category: "config", SourcePath: "~/.config/opencode/unreadable.txt", RelPath: "unreadable.txt", IsDir: false, Hash: "sha256:abc", Size: 4},
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
	configDir := filepath.Join(home, ".config", "opencode")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(configDir, 0500); err != nil {
		t.Fatal(err)
	}

	backupDir := filepath.Join(t.TempDir(), "backup")
	srcFile := filepath.Join(backupDir, "opencode", "secret.json")
	if err := os.MkdirAll(filepath.Dir(srcFile), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(srcFile, []byte(`{"key":"secret"}`), 0644); err != nil {
		t.Fatal(err)
	}

	items := []adapters.Item{
		{Category: "config", SourcePath: "~/.config/opencode/secret.json", RelPath: "secret.json", IsDir: false, Hash: "sha256:xyz", Size: 16},
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
