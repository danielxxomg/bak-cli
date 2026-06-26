package adapters_test

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/danielxxomg/bak-cli/internal/adapters"
)

// newMultiCatAdapter returns a GenericAdapter whose RootConfigFiles maps
// root entries to multiple categories (config + mcp), mirroring the
// opencode shape. Used to exercise multi-category root-file scanning.
func newMultiCatAdapter(name string) adapters.GenericAdapter {
	return adapters.GenericAdapter{
		AdapterName:   name,
		ConfigRelPath: ".test",
		Categories: map[string]adapters.CategoryDir{
			"config": {SubPath: "", IsDir: false},
			"mcp":    {SubPath: "", IsDir: false},
		},
		DetectErrContext: "stat " + name + " config dir",
		RootConfigFiles: map[string]string{
			"opencode.json": "config",
			"mcp.json":      "mcp",
		},
	}
}

// captureStderr redirects os.Stderr for the duration of fn and returns
// whatever was written. It restores the original stderr afterward.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stderr = w
	defer func() { os.Stderr = old }()

	fn()

	if cerr := w.Close(); cerr != nil {
		t.Fatalf("close pipe writer: %v", cerr)
	}
	var buf bytes.Buffer
	if _, cerr := io.Copy(&buf, r); cerr != nil {
		t.Fatalf("read stderr pipe: %v", cerr)
	}
	return buf.String()
}

// writeMultiCatRoot creates a config root with opencode.json (config) and
// mcp.json (mcp) and returns the home directory.
func writeMultiCatRoot(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	configDir := filepath.Join(home, ".test")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "opencode.json"), []byte(`{"v":1}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "mcp.json"), []byte(`{"servers":{}}`), 0644); err != nil {
		t.Fatal(err)
	}
	return home
}

// TestGenericAdapter_MultiCategoryRootFiles covers the generalized
// scanRootFiles: a root file is included iff its mapped category is in the
// requested set, its Item.Category is the mapped category (not a fixed
// default), and the root scan runs once across multiple categories.
func TestGenericAdapter_MultiCategoryRootFiles(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	ga := newMultiCatAdapter("multicat")

	t.Run("file included when its category is requested", func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
		home := writeMultiCatRoot(t)
		items, err := ga.ListItems(home, []string{"mcp"})
		if err != nil {
			t.Fatalf("ListItems: %v", err)
		}
		if len(items) != 1 {
			t.Fatalf("expected exactly 1 item for [mcp], got %d: %+v", len(items), items)
		}
		if items[0].RelPath != "mcp.json" {
			t.Errorf("RelPath = %q, want mcp.json", items[0].RelPath)
		}
		if items[0].Category != "mcp" {
			t.Errorf("Category = %q, want mcp", items[0].Category)
		}
	})

	t.Run("file excluded when its category is not requested", func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
		home := writeMultiCatRoot(t)
		items, err := ga.ListItems(home, []string{"config"})
		if err != nil {
			t.Fatalf("ListItems: %v", err)
		}
		for _, it := range items {
			if it.RelPath == "mcp.json" {
				t.Errorf("mcp.json should be absent when only [config] requested, got %+v", it)
			}
		}
		// opencode.json (config) must still be present.
		foundConfig := false
		for _, it := range items {
			if it.RelPath == "opencode.json" {
				foundConfig = true
				if it.Category != "config" {
					t.Errorf("opencode.json category = %q, want config", it.Category)
				}
			}
		}
		if !foundConfig {
			t.Error("opencode.json not found — should be included for [config]")
		}
	})

	t.Run("root scan runs once for multiple matching categories", func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
		home := writeMultiCatRoot(t)
		items, err := ga.ListItems(home, []string{"config", "mcp"})
		if err != nil {
			t.Fatalf("ListItems: %v", err)
		}
		// Both files present exactly once (no duplicates means the root
		// scan ran once, not once per requested category).
		counts := map[string]int{}
		gotCat := map[string]string{}
		for _, it := range items {
			counts[it.RelPath]++
			gotCat[it.RelPath] = it.Category
		}
		if counts["mcp.json"] != 1 {
			t.Errorf("mcp.json appears %d times, want exactly 1 (single root scan)", counts["mcp.json"])
		}
		if counts["opencode.json"] != 1 {
			t.Errorf("opencode.json appears %d times, want exactly 1 (single root scan)", counts["opencode.json"])
		}
		if gotCat["mcp.json"] != "mcp" {
			t.Errorf("mcp.json category = %q, want mcp", gotCat["mcp.json"])
		}
		if gotCat["opencode.json"] != "config" {
			t.Errorf("opencode.json category = %q, want config", gotCat["opencode.json"])
		}
	})
}

// TestGenericAdapter_MaxFileSizeRootFiles covers MaxFileSize applied to
// root files for both adapter shapes: the legacy nil-RootConfigFiles
// adapter (every root file is "config") and the multi-category adapter.
// In each case an oversized file is skipped with a stderr warning while a
// small file is included.
func TestGenericAdapter_MaxFileSizeRootFiles(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	tests := []struct {
		name       string
		adapter    adapters.GenericAdapter
		categories []string
		smallFile  string // included
		largeFile  string // skipped (oversized)
	}{
		{
			name:       "legacy config-only adapter",
			adapter:    newTestAdapter("maxsize-legacy"),
			categories: []string{"config"},
			smallFile:  "small.txt",
			largeFile:  "large.log",
		},
		{
			name:       "multi-category adapter",
			adapter:    newMultiCatAdapter("maxsize-multicat"),
			categories: []string{"config", "mcp"},
			smallFile:  "opencode.json",
			largeFile:  "mcp.json",
		},
	}

	for _, tt := range tests { //nolint:paralleltest // subtests share table/struct state
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
			home := t.TempDir()
			configDir := filepath.Join(home, ".test")
			if err := os.MkdirAll(configDir, 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(configDir, tt.smallFile), []byte("ok"), 0644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(configDir, tt.largeFile), bytes.Repeat([]byte("x"), 200), 0644); err != nil {
				t.Fatal(err)
			}

			ga := tt.adapter
			ga.ScanOpts = adapters.ScanOptions{MaxFileSize: 100}

			var items []adapters.Item
			stderr := captureStderr(t, func() {
				var err error
				items, err = ga.ListItems(home, tt.categories)
				if err != nil {
					t.Fatalf("ListItems: %v", err)
				}
			})

			foundSmall := false
			for _, it := range items {
				if it.RelPath == tt.largeFile {
					t.Errorf("oversized %q should have been skipped, got %+v", tt.largeFile, it)
				}
				if it.RelPath == tt.smallFile {
					foundSmall = true
				}
			}
			if !foundSmall {
				t.Errorf("small file %q not found — should have been included", tt.smallFile)
			}
			if !strings.Contains(stderr, tt.largeFile) {
				t.Errorf("expected stderr warning mentioning %q, got %q", tt.largeFile, stderr)
			}
		})
	}
}

// newTestAdapter returns a GenericAdapter configured for testing with
// a ".test" config directory under homeDir.
func newTestAdapter(name string) adapters.GenericAdapter {
	return adapters.GenericAdapter{
		AdapterName:   name,
		ConfigRelPath: ".test",
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

func TestGenericAdapter_Name(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	tests := []struct {
		name        string
		adapterName string
		want        string
	}{
		{name: "codex", adapterName: "codex", want: "codex"},
		{name: "claude-code", adapterName: "claude-code", want: "claude-code"},
		{name: "cursor", adapterName: "cursor", want: "cursor"},
	}

	for _, tt := range tests { //nolint:paralleltest // subtests share table/struct state
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
			ga := newTestAdapter(tt.adapterName)
			if got := ga.Name(); got != tt.want {
				t.Errorf("Name() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGenericAdapter_Detect(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	ga := newTestAdapter("test-tool")

	t.Run("installed", func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
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

	t.Run("not installed", func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
		home := t.TempDir()

		installed, _, err := ga.Detect(home)
		if err != nil {
			t.Fatalf("Detect: %v", err)
		}
		if installed {
			t.Error("expected installed=false when dir is missing")
		}
	})

	t.Run("exists but is file not dir", func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
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

	t.Run("stat error", func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
		home := t.TempDir()
		configDir := filepath.Join(home, ".test")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			t.Fatal(err)
		}

		gaStat := newTestAdapter("stat-test")
		gaStat.StatFn = func(path string) (os.FileInfo, error) {
			return nil, &os.PathError{Op: "stat", Path: path, Err: os.ErrPermission}
		}

		_, _, err := gaStat.Detect(home)
		if err == nil {
			t.Error("expected error from injected StatFn, got nil")
		}
	})

	t.Run("stat not exist via injection", func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
		home := t.TempDir()

		gaStat := newTestAdapter("notexist-test")
		gaStat.StatFn = func(path string) (os.FileInfo, error) {
			return nil, &os.PathError{Op: "stat", Path: path, Err: os.ErrNotExist}
		}

		installed, _, err := gaStat.Detect(home)
		if err != nil {
			t.Fatalf("Detect should not error on injected not-exist: %v", err)
		}
		if installed {
			t.Error("expected installed=false when StatFn returns not-exist")
		}
	})

	t.Run("path traversal blocked", func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
		evil := newTestAdapter("evil")
		evil.ConfigRelPath = "../../etc"
		home := t.TempDir()

		_, _, err := evil.Detect(home)
		if err == nil {
			t.Error("expected error for path traversal, got nil")
		}
	})
}

func TestGenericAdapter_ListItems(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	ga := newTestAdapter("test-tool")

	t.Run("config category returns root files", func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
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

	t.Run("scripts category returns dir contents", func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
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

	t.Run("rel path uses forward slashes", func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
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

	t.Run("all categories", func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
		home := setupConfigHome(t)
		items, err := ga.ListItems(home, []string{"config", "scripts"})
		if err != nil {
			t.Fatalf("ListItems: %v", err)
		}
		if len(items) < 3 {
			t.Fatalf("expected at least 3 items across categories, got %d", len(items))
		}
	})

	t.Run("empty categories returns empty slice", func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
		home := setupConfigHome(t)
		items, err := ga.ListItems(home, []string{})
		if err != nil {
			t.Fatalf("ListItems: %v", err)
		}
		if len(items) != 0 {
			t.Errorf("expected 0 items for empty categories, got %d", len(items))
		}
	})

	t.Run("unknown category returns empty slice", func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
		home := setupConfigHome(t)
		items, err := ga.ListItems(home, []string{"nonexistent"})
		if err != nil {
			t.Fatalf("ListItems: %v", err)
		}
		if len(items) != 0 {
			t.Errorf("expected 0 items for unknown category, got %d", len(items))
		}
	})

	t.Run("empty result for missing dirs", func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
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

func TestGenericAdapter_Backup(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	ga := newTestAdapter("test-tool")

	t.Run("copies file to backup dir", func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
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

	t.Run("creates directory for dir items", func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
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

	t.Run("copy error on missing source", func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
		home := t.TempDir()
		configDir := filepath.Join(home, ".test")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			t.Fatal(err)
		}

		// Do NOT create the source file — Backup copies items from
		// configDir, so a missing source triggers an open error in CopyFile.
		items := []adapters.Item{
			{Category: "config", SourcePath: "~/.test/missing.txt", RelPath: "missing.txt", IsDir: false, Hash: "sha256:abc", Size: 4},
		}

		backupDir := filepath.Join(t.TempDir(), "backup")
		err := ga.Backup(home, backupDir, items)
		if err == nil {
			t.Error("expected error for missing source file, got nil")
		}
	})
}

func TestGenericAdapter_Restore(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	ga := newTestAdapter("test-tool")

	t.Run("copies file from backup to home", func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
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

	t.Run("creates directory for dir items on restore", func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
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

	t.Run("copy error on restore", func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
		home := t.TempDir()
		// Create a file at the config path instead of a directory.
		// This causes os.MkdirAll inside copyItems to fail because
		// the parent is not a directory — works cross-platform.
		configPath := filepath.Join(home, ".test")
		if err := os.WriteFile(configPath, []byte("not-a-dir"), 0644); err != nil {
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
			t.Error("expected error for copy to file path, got nil")
		}
	})
}

// TestScanRootFiles_AppliesExcludes verifies that scanRootFiles honors
// ScanOptions (MatchExclude + MaxFileSize) when filtering root-level files.
func TestScanRootFiles_AppliesExcludes(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	t.Run("excludes sqlite files", func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
		home := t.TempDir()
		configDir := filepath.Join(home, ".test")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			t.Fatal(err)
		}
		// Create a config file (should be included).
		if err := os.WriteFile(filepath.Join(configDir, "settings.json"), []byte(`{"a":1}`), 0644); err != nil {
			t.Fatal(err)
		}
		// Create sqlite files (should be excluded).
		if err := os.WriteFile(filepath.Join(configDir, "logs.sqlite"), []byte("db"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(configDir, "state.sqlite-wal"), []byte("wal"), 0644); err != nil {
			t.Fatal(err)
		}

		ga := newTestAdapter("exclude-test")
		ga.ScanOpts = adapters.ScanOptions{
			Excludes: []string{"*.sqlite", "*.sqlite-wal"},
		}

		items, err := ga.ListItems(home, []string{"config"})
		if err != nil {
			t.Fatalf("ListItems: %v", err)
		}

		// Verify settings.json is included.
		foundSettings := false
		for _, item := range items {
			if item.RelPath == "settings.json" {
				foundSettings = true
				if item.Category != "config" {
					t.Errorf("settings.json category = %q, want config", item.Category)
				}
			}
		}
		if !foundSettings {
			t.Error("settings.json not found — should have been included")
		}

		// Verify sqlite files are excluded.
		for _, item := range items {
			if item.RelPath == "logs.sqlite" || item.RelPath == "state.sqlite-wal" {
				t.Errorf("sqlite file %q should have been excluded", item.RelPath)
			}
		}
	})

	t.Run("custom exclude patterns apply", func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
		home := t.TempDir()
		configDir := filepath.Join(home, ".test")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(configDir, "config.yml"), []byte("yaml"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(configDir, "data.tmp"), []byte("tmp"), 0644); err != nil {
			t.Fatal(err)
		}

		ga := newTestAdapter("custom-exclude-test")
		ga.ScanOpts = adapters.ScanOptions{
			Excludes: []string{"*.tmp"},
		}

		items, err := ga.ListItems(home, []string{"config"})
		if err != nil {
			t.Fatalf("ListItems: %v", err)
		}

		foundYml := false
		for _, item := range items {
			if item.RelPath == "config.yml" {
				foundYml = true
			}
			if item.RelPath == "data.tmp" {
				t.Error("data.tmp should have been excluded by custom pattern *.tmp")
			}
		}
		if !foundYml {
			t.Error("config.yml not found — should have been included")
		}
	})
}

// TestGenericAdapter_HashErrorBranches exercises the FileHash error branch in
// both scanDir (subdirectory file) and scanRootFiles (root file) using a
// chmod-000 fixture. It proves the error is wrapped with a lowercase context
// prefix and uses the relative path (not the absolute home path). Skipped on
// Windows where chmod 000 does not block reads.
func TestGenericAdapter_HashErrorBranches(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	if runtime.GOOS == "windows" {
		t.Skip("chmod 000 does not block file reads on Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("running as root bypasses chmod 000 permissions")
	}

	t.Run("scanDir wraps hash error with rel path", func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
		home := t.TempDir()
		configDir := filepath.Join(home, ".test")
		scriptsDir := filepath.Join(configDir, "scripts")
		if err := os.MkdirAll(scriptsDir, 0755); err != nil {
			t.Fatal(err)
		}
		unreadable := filepath.Join(scriptsDir, "locked.sh")
		if err := os.WriteFile(unreadable, []byte("#!/bin/sh"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(unreadable, 0000); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if cerr := os.Chmod(unreadable, 0644); cerr != nil {
				t.Logf("cleanup chmod: %v", cerr)
			}
		})

		ga := newTestAdapter("hash-err-dir")
		_, err := ga.ListItems(home, []string{"scripts"})
		if err == nil {
			t.Fatal("expected error from unreadable file in scanDir, got nil")
		}
		msg := err.Error()
		// The wrapping context prefix (bug-fix #4) must use the rel path,
		// not the absolute home path. The underlying FileHash error still
		// contains the absolute open path — that is FileHash's behavior and
		// out of scope here; we only assert the wrapping context prefix.
		if !strings.Contains(msg, "hash scripts/locked.sh:") {
			t.Errorf("error %q should wrap with rel-path context %q", msg, "hash scripts/locked.sh:")
		}
	})

	t.Run("scanRootFiles wraps hash error with entry name", func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
		home := t.TempDir()
		configDir := filepath.Join(home, ".test")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			t.Fatal(err)
		}
		unreadable := filepath.Join(configDir, "settings.json")
		if err := os.WriteFile(unreadable, []byte(`{"a":1}`), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(unreadable, 0000); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if cerr := os.Chmod(unreadable, 0644); cerr != nil {
				t.Logf("cleanup chmod: %v", cerr)
			}
		})

		ga := newTestAdapter("hash-err-root")
		_, err := ga.ListItems(home, []string{"config"})
		if err == nil {
			t.Fatal("expected error from unreadable root file in scanRootFiles, got nil")
		}
		msg := err.Error()
		// The wrapping context prefix (bug-fix #4) must use the entry name,
		// not the absolute home path. The underlying FileHash error still
		// contains the absolute open path — out of scope here.
		if !strings.Contains(msg, "hash settings.json:") {
			t.Errorf("error %q should wrap with entry-name context %q", msg, "hash settings.json:")
		}
	})
}

func TestGenericAdapter_InterfaceCompliance(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
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
