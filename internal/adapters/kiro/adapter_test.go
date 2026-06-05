package kiro

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/danielxxomg/bak-cli/internal/adapters"
)

func TestAdapter_Name(t *testing.T) {
	a := &Adapter{}
	if a.Name() != "kiro" {
		t.Errorf("Name() = %q, want %q", a.Name(), "kiro")
	}
}

func TestAdapter_Detect(t *testing.T) {
	a := &Adapter{}

	t.Run("installed", func(t *testing.T) {
		home := t.TempDir()
		configDir := filepath.Join(home, ".kiro")
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
		configPath := filepath.Join(home, ".kiro")
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
		configDir := filepath.Join(home, ".kiro")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte(`{"model":"gpt-4"}`), 0644); err != nil {
			t.Fatal(err)
		}
		// Hooks directory
		hooksDir := filepath.Join(configDir, "hooks")
		if err := os.MkdirAll(hooksDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(hooksDir, "pre-commit.sh"), []byte("#!/bin/sh"), 0644); err != nil {
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

	t.Run("hooks category", func(t *testing.T) {
		home := setupHome(t)
		items, err := a.ListItems(home, []string{"hooks"})
		if err != nil {
			t.Fatalf("ListItems: %v", err)
		}
		if len(items) < 1 {
			t.Fatal("expected at least 1 hooks item")
		}
		for _, item := range items {
			if item.Category != "hooks" {
				t.Errorf("item %q has category %q, want hooks", item.RelPath, item.Category)
			}
		}
	})

	t.Run("all categories", func(t *testing.T) {
		home := setupHome(t)
		items, err := a.ListItems(home, []string{"config", "hooks"})
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
	configDir := filepath.Join(home, ".kiro")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte(`{"model":"gpt-4"}`), 0644); err != nil {
		t.Fatal(err)
	}
	items := []adapters.Item{
		{Category: "config", SourcePath: "~/.kiro/config.json", RelPath: "config.json", IsDir: false, Hash: "sha256:abc", Size: 18},
	}
	backupDir := filepath.Join(t.TempDir(), "backup")
	if err := a.Backup(home, backupDir, items); err != nil {
		t.Fatalf("Backup: %v", err)
	}
	dstFile := filepath.Join(backupDir, "kiro", "config.json")
	data, err := os.ReadFile(dstFile)
	if err != nil {
		t.Fatalf("read backup file: %v", err)
	}
	if string(data) != `{"model":"gpt-4"}` {
		t.Errorf("backup content = %q, want %q", string(data), `{"model":"gpt-4"}`)
	}
}

func TestAdapter_Restore(t *testing.T) {
	a := &Adapter{}
	backupDir := filepath.Join(t.TempDir(), "backup")
	backupFile := filepath.Join(backupDir, "kiro", "config.json")
	if err := os.MkdirAll(filepath.Dir(backupFile), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(backupFile, []byte(`{"restored":true}`), 0644); err != nil {
		t.Fatal(err)
	}
	items := []adapters.Item{
		{Category: "config", SourcePath: "~/.kiro/config.json", RelPath: "config.json", IsDir: false, Hash: "sha256:xyz", Size: 18},
	}
	home := t.TempDir()
	if err := a.Restore(backupDir, home, items); err != nil {
		t.Fatalf("Restore: %v", err)
	}
	restoredFile := filepath.Join(home, ".kiro", "config.json")
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
