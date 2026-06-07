package adapters

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// ---------- ConfigAdapter.Detect -----------------------------------------

func TestConfigAdapter_Detect(t *testing.T) {
	t.Run("installed", func(t *testing.T) {
		home := t.TempDir()
		configDir := filepath.Join(home, ".config", "myapp")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			t.Fatal(err)
		}

		a := &ConfigAdapter{def: YAMLAdapter{Name: "myapp", ConfigPath: ".config/myapp"}}
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
		a := &ConfigAdapter{def: YAMLAdapter{Name: "myapp", ConfigPath: ".config/myapp"}}

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
		configPath := filepath.Join(home, ".config", "myapp")
		if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(configPath, []byte("not a dir"), 0644); err != nil {
			t.Fatal(err)
		}

		a := &ConfigAdapter{def: YAMLAdapter{Name: "myapp", ConfigPath: ".config/myapp"}}
		installed, _, err := a.Detect(home)
		if err != nil {
			t.Fatalf("Detect: %v", err)
		}
		if installed {
			t.Error("expected installed=false when path is a file")
		}
	})
}

// ---------- ConfigAdapter.ListItems --------------------------------------

func TestConfigAdapter_ListItems(t *testing.T) {
	setupHome := func(t *testing.T) (string, string) {
		t.Helper()
		home := t.TempDir()
		configDir := filepath.Join(home, ".config", "myapp")

		// root config files (non-directory category)
		if err := os.MkdirAll(configDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte(`{}`), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(configDir, "README.md"), []byte(`# Readme`), 0644); err != nil {
			t.Fatal(err)
		}

		// directory category: plugins
		pluginsDir := filepath.Join(configDir, "plugins")
		if err := os.MkdirAll(pluginsDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(pluginsDir, "plugin-a.js"), []byte(`// A`), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(pluginsDir, "plugin-b.js"), []byte(`// B`), 0644); err != nil {
			t.Fatal(err)
		}

		return home, configDir
	}

	t.Run("dir category returns items", func(t *testing.T) {
		home, _ := setupHome(t)
		a := &ConfigAdapter{def: YAMLAdapter{
			Name:       "myapp",
			ConfigPath: ".config/myapp",
			Categories: []YAMLCategoryPattern{
				{Name: "plugins", SubPath: "plugins", IsDir: true},
			},
		}}

		items, err := a.ListItems(home, []string{"plugins"})
		if err != nil {
			t.Fatalf("ListItems: %v", err)
		}
		if len(items) != 2 {
			t.Fatalf("expected 2 items in plugins dir, got %d", len(items))
		}
		for _, it := range items {
			if it.Category != "plugins" {
				t.Errorf("item %q has category %q, want plugins", it.RelPath, it.Category)
			}
			if it.Hash == "" {
				t.Errorf("file item %q has empty hash", it.RelPath)
			}
		}
	})

	t.Run("root files category returns items", func(t *testing.T) {
		home, _ := setupHome(t)
		a := &ConfigAdapter{def: YAMLAdapter{
			Name:       "myapp",
			ConfigPath: ".config/myapp",
			Categories: []YAMLCategoryPattern{
				{Name: "config", RootFiles: []string{"config.json", "README.md"}},
			},
		}}

		items, err := a.ListItems(home, []string{"config"})
		if err != nil {
			t.Fatalf("ListItems: %v", err)
		}
		if len(items) != 2 {
			t.Fatalf("expected 2 config items, got %d", len(items))
		}
		for _, it := range items {
			if it.Category != "config" {
				t.Errorf("item %q has category %q, want config", it.RelPath, it.Category)
			}
		}
	})

	t.Run("missing category returns empty", func(t *testing.T) {
		home, _ := setupHome(t)
		a := &ConfigAdapter{def: YAMLAdapter{
			Name:       "myapp",
			ConfigPath: ".config/myapp",
		}}

		items, err := a.ListItems(home, []string{"nonexistent"})
		if err != nil {
			t.Fatalf("ListItems: %v", err)
		}
		if len(items) != 0 {
			t.Errorf("expected 0 items for unknown category, got %d", len(items))
		}
	})

	t.Run("missing dir category returns empty", func(t *testing.T) {
		home := t.TempDir()
		configDir := filepath.Join(home, ".config", "myapp")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			t.Fatal(err)
		}

		a := &ConfigAdapter{def: YAMLAdapter{
			Name:       "myapp",
			ConfigPath: ".config/myapp",
			Categories: []YAMLCategoryPattern{
				{Name: "plugins", SubPath: "plugins", IsDir: true},
			},
		}}

		items, err := a.ListItems(home, []string{"plugins"})
		if err != nil {
			t.Fatalf("ListItems: %v", err)
		}
		if len(items) != 0 {
			t.Errorf("expected 0 items for non-existent dir, got %d", len(items))
		}
	})
}

// ---------- ConfigAdapter.Backup -----------------------------------------

func TestConfigAdapter_Backup(t *testing.T) {
	home := t.TempDir()
	configDir := filepath.Join(home, ".config", "myapp")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a test file in config root.
	srcContent := []byte(`{"key":"value"}`)
	srcFile := filepath.Join(configDir, "config.json")
	if err := os.WriteFile(srcFile, srcContent, 0644); err != nil {
		t.Fatal(err)
	}

	// Create a plugin directory with a file.
	pluginFile := filepath.Join(configDir, "plugins", "util.js")
	if err := os.MkdirAll(filepath.Dir(pluginFile), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pluginFile, []byte(`// util`), 0644); err != nil {
		t.Fatal(err)
	}

	a := &ConfigAdapter{def: YAMLAdapter{Name: "myapp", ConfigPath: ".config/myapp"}}
	backupDir := filepath.Join(t.TempDir(), "backup")

	items := []Item{
		{Category: "config", RelPath: "config.json", IsDir: false},
		{Category: "plugins", RelPath: "plugins", IsDir: true},
		{Category: "plugins", RelPath: "plugins/util.js", IsDir: false},
	}

	if err := a.Backup(home, backupDir, items); err != nil {
		t.Fatalf("Backup: %v", err)
	}

	// Verify config file was copied.
	dstFile := filepath.Join(backupDir, "myapp", "config.json")
	data, err := os.ReadFile(dstFile)
	if err != nil {
		t.Fatalf("read backup file: %v", err)
	}
	if string(data) != string(srcContent) {
		t.Errorf("backup content = %q, want %q", string(data), string(srcContent))
	}

	// Verify directory was created in backup.
	dstDir := filepath.Join(backupDir, "myapp", "plugins")
	if info, err := os.Stat(dstDir); err != nil || !info.IsDir() {
		t.Errorf("expected backup dir %q to exist", dstDir)
	}

	// Verify plugin file was copied.
	dstPlugin := filepath.Join(backupDir, "myapp", "plugins", "util.js")
	pluginData, err := os.ReadFile(dstPlugin)
	if err != nil {
		t.Fatalf("read backup plugin file: %v", err)
	}
	if string(pluginData) != "// util" {
		t.Errorf("backup plugin content = %q, want %q", string(pluginData), "// util")
	}
}

// ---------- ConfigAdapter.Restore ----------------------------------------

func TestConfigAdapter_Restore(t *testing.T) {
	backupDir := filepath.Join(t.TempDir(), "backup")
	backupFile := filepath.Join(backupDir, "myapp", "config.json")
	if err := os.MkdirAll(filepath.Dir(backupFile), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(backupFile, []byte(`{"restored":true}`), 0644); err != nil {
		t.Fatal(err)
	}

	backupPlugin := filepath.Join(backupDir, "myapp", "plugins", "util.js")
	if err := os.MkdirAll(filepath.Dir(backupPlugin), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(backupPlugin, []byte(`// restored`), 0644); err != nil {
		t.Fatal(err)
	}

	a := &ConfigAdapter{def: YAMLAdapter{Name: "myapp", ConfigPath: ".config/myapp"}}
	home := t.TempDir()

	items := []Item{
		{Category: "config", RelPath: "config.json", IsDir: false},
		{Category: "plugins", RelPath: "plugins", IsDir: true},
		{Category: "plugins", RelPath: "plugins/util.js", IsDir: false},
	}

	if err := a.Restore(backupDir, home, items); err != nil {
		t.Fatalf("Restore: %v", err)
	}

	// Verify file was restored.
	restoredFile := filepath.Join(home, ".config", "myapp", "config.json")
	data, err := os.ReadFile(restoredFile)
	if err != nil {
		t.Fatalf("read restored file: %v", err)
	}
	if string(data) != `{"restored":true}` {
		t.Errorf("restored content = %q", string(data))
	}

	// Verify plugin file was restored.
	restoredPlugin := filepath.Join(home, ".config", "myapp", "plugins", "util.js")
	pluginData, err := os.ReadFile(restoredPlugin)
	if err != nil {
		t.Fatalf("read restored plugin: %v", err)
	}
	if string(pluginData) != "// restored" {
		t.Errorf("restored plugin content = %q", string(pluginData))
	}
}

// ---------- fileHash -----------------------------------------------------

func Test_fileHash(t *testing.T) {
	t.Run("compute hash of known content", func(t *testing.T) {
		dir := t.TempDir()
		fpath := filepath.Join(dir, "test.txt")
		if err := os.WriteFile(fpath, []byte("hello world"), 0644); err != nil {
			t.Fatal(err)
		}

		hash, size, err := fileHash(fpath)
		if err != nil {
			t.Fatalf("fileHash: %v", err)
		}
		if size != 11 {
			t.Errorf("size = %d, want 11", size)
		}

		// Verify hash is correct SHA-256 of "hello world".
		h := sha256.New()
		h.Write([]byte("hello world"))
		expected := fmt.Sprintf("sha256:%x", h.Sum(nil))
		if hash != expected {
			t.Errorf("hash = %q, want %q", hash, expected)
		}
	})

	t.Run("empty file", func(t *testing.T) {
		dir := t.TempDir()
		fpath := filepath.Join(dir, "empty.txt")
		if err := os.WriteFile(fpath, []byte{}, 0644); err != nil {
			t.Fatal(err)
		}

		hash, size, err := fileHash(fpath)
		if err != nil {
			t.Fatalf("fileHash: %v", err)
		}
		if size != 0 {
			t.Errorf("size = %d, want 0", size)
		}
		if hash == "" {
			t.Error("hash should not be empty for empty file")
		}
	})

	t.Run("nonexistent file", func(t *testing.T) {
		dir := t.TempDir()
		fpath := filepath.Join(dir, "missing.txt")

		_, _, err := fileHash(fpath)
		if err == nil {
			t.Error("expected error for nonexistent file")
		}
	})
}

// ---------- scanCategoryDir ----------------------------------------------

func Test_scanCategoryDir(t *testing.T) {
	t.Run("scans files and dirs under category", func(t *testing.T) {
		home := t.TempDir()
		configDir := filepath.Join(home, ".config", "myapp")
		pluginsDir := filepath.Join(configDir, "plugins")
		if err := os.MkdirAll(pluginsDir, 0755); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(filepath.Join(pluginsDir, "a.js"), []byte("a"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(pluginsDir, "b.js"), []byte("bb"), 0644); err != nil {
			t.Fatal(err)
		}

		items, err := scanCategoryDir(pluginsDir, "plugins", configDir)
		if err != nil {
			t.Fatalf("scanCategoryDir: %v", err)
		}
		if len(items) != 2 {
			t.Fatalf("expected 2 items, got %d", len(items))
		}
		for _, it := range items {
			if it.Category != "plugins" {
				t.Errorf("item category = %q, want plugins", it.Category)
			}
			if it.Hash == "" {
				t.Errorf("item %q has empty hash", it.RelPath)
			}
			if it.IsDir {
				t.Errorf("item %q should not be a directory", it.RelPath)
			}
		}
	})

	t.Run("empty dir returns no items", func(t *testing.T) {
		home := t.TempDir()
		pluginsDir := filepath.Join(home, ".config", "myapp", "plugins")
		if err := os.MkdirAll(pluginsDir, 0755); err != nil {
			t.Fatal(err)
		}

		items, err := scanCategoryDir(pluginsDir, "plugins", filepath.Join(home, ".config", "myapp"))
		if err != nil {
			t.Fatalf("scanCategoryDir: %v", err)
		}
		if len(items) != 0 {
			t.Errorf("expected 0 items in empty dir, got %d", len(items))
		}
	})

	t.Run("nonexistent dir returns error", func(t *testing.T) {
		dir := t.TempDir()
		missingPath := filepath.Join(dir, "nonexistent")
		// Ensure the path does not exist.
		_, err := scanCategoryDir(missingPath, "plugins", filepath.Join(dir, "config"))
		if err == nil {
			t.Error("expected error for nonexistent directory")
		}
	})
}

// ---------- copyFile -----------------------------------------------------

func Test_copyFile(t *testing.T) {
	t.Run("copies content to destination", func(t *testing.T) {
		dir := t.TempDir()
		src := filepath.Join(dir, "src.txt")
		dst := filepath.Join(dir, "sub", "dst.txt")

		if err := os.WriteFile(src, []byte("copy me"), 0644); err != nil {
			t.Fatal(err)
		}

		if err := copyFile(src, dst); err != nil {
			t.Fatalf("copyFile: %v", err)
		}

		data, err := os.ReadFile(dst)
		if err != nil {
			t.Fatalf("read dst: %v", err)
		}
		if string(data) != "copy me" {
			t.Errorf("dst content = %q, want %q", string(data), "copy me")
		}
	})

	t.Run("nonexistent source returns error", func(t *testing.T) {
		dir := t.TempDir()
		src := filepath.Join(dir, "missing.txt")
		dst := filepath.Join(dir, "dst.txt")

		err := copyFile(src, dst)
		if err == nil {
			t.Error("expected error for nonexistent source")
		}
	})
}

// ---------- LoadYAMLAdapters (NEW signature: dir, homeDir) ---------------
// RED phase: these tests reference LoadYAMLAdapters(dir, homeDir)
// which does NOT exist yet on the current source.

func TestLoadYAMLAdapters(t *testing.T) {
	t.Run("valid yaml loads adapter", func(t *testing.T) {
		homeDir := t.TempDir()
		adaptersDir := filepath.Join(homeDir, ".config", "bak", "adapters")
		if err := os.MkdirAll(adaptersDir, 0755); err != nil {
			t.Fatal(err)
		}

		yamlContent := `name: test-app
config_path: .config/testapp
categories:
  - name: config
    root_files:
      - config.yaml
`
		if err := os.WriteFile(filepath.Join(adaptersDir, "test.yaml"), []byte(yamlContent), 0644); err != nil {
			t.Fatal(err)
		}

		adapters, err := LoadYAMLAdapters(adaptersDir, homeDir)
		if err != nil {
			t.Fatalf("LoadYAMLAdapters: %v", err)
		}
		if len(adapters) != 1 {
			t.Fatalf("expected 1 adapter, got %d", len(adapters))
		}
		if adapters[0].Name() != "test-app" {
			t.Errorf("Name() = %q, want test-app", adapters[0].Name())
		}
	})

	t.Run("missing directory returns empty", func(t *testing.T) {
		homeDir := t.TempDir()
		adaptersDir := filepath.Join(homeDir, ".config", "bak", "adapters")

		adapters, err := LoadYAMLAdapters(adaptersDir, homeDir)
		if err != nil {
			t.Fatalf("LoadYAMLAdapters: %v", err)
		}
		if len(adapters) != 0 {
			t.Errorf("expected 0 adapters for missing dir, got %d", len(adapters))
		}
	})

	t.Run("invalid yaml returns error", func(t *testing.T) {
		homeDir := t.TempDir()
		adaptersDir := filepath.Join(homeDir, ".config", "bak", "adapters")
		if err := os.MkdirAll(adaptersDir, 0755); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(filepath.Join(adaptersDir, "bad.yaml"), []byte(`{{{invalid`), 0644); err != nil {
			t.Fatal(err)
		}

		_, err := LoadYAMLAdapters(adaptersDir, homeDir)
		if err == nil {
			t.Error("expected error for invalid YAML")
		}
	})

	t.Run("missing name field returns error", func(t *testing.T) {
		homeDir := t.TempDir()
		adaptersDir := filepath.Join(homeDir, ".config", "bak", "adapters")
		if err := os.MkdirAll(adaptersDir, 0755); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(filepath.Join(adaptersDir, "noname.yaml"), []byte(`config_path: .config/app`), 0644); err != nil {
			t.Fatal(err)
		}

		_, err := LoadYAMLAdapters(adaptersDir, homeDir)
		if err == nil {
			t.Error("expected error for missing name field")
		}
	})

	t.Run("traversal rejected: dir outside home", func(t *testing.T) {
		homeDir := t.TempDir()
		outsideDir := t.TempDir()

		if err := os.WriteFile(filepath.Join(outsideDir, "test.yaml"), []byte(`name: bad\nconfig_path: /etc`), 0644); err != nil {
			t.Fatal(err)
		}

		_, err := LoadYAMLAdapters(outsideDir, homeDir)
		if err == nil {
			t.Error("expected error for directory outside home")
		}
	})

	t.Run("ignores non-yaml files", func(t *testing.T) {
		homeDir := t.TempDir()
		adaptersDir := filepath.Join(homeDir, ".config", "bak", "adapters")
		if err := os.MkdirAll(adaptersDir, 0755); err != nil {
			t.Fatal(err)
		}

		// Create a valid yaml and a .txt file.
		yamlContent := `name: myapp
config_path: .config/myapp
`
		if err := os.WriteFile(filepath.Join(adaptersDir, "myapp.yaml"), []byte(yamlContent), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(adaptersDir, "readme.txt"), []byte("ignored"), 0644); err != nil {
			t.Fatal(err)
		}

		adapters, err := LoadYAMLAdapters(adaptersDir, homeDir)
		if err != nil {
			t.Fatalf("LoadYAMLAdapters: %v", err)
		}
		if len(adapters) != 1 {
			t.Errorf("expected 1 adapter (only yaml), got %d", len(adapters))
		}
	})
}
