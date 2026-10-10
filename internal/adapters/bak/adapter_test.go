package bak

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/danielxxomg/bak-cli/internal/adapters"
)

func TestBakAdapter_Name(t *testing.T) { //nolint:paralleltest // shared state isolation
	a := &Adapter{}
	if got := a.Name(); got != "bak" {
		t.Errorf("Name() = %q, want %q", got, "bak")
	}
}

func TestBakAdapter_Detect(t *testing.T) { //nolint:paralleltest // subtests share struct state
	a := &Adapter{}

	tests := []struct {
		name          string
		setup         func(t *testing.T, home string)
		wantInstalled bool
	}{
		{
			name: "installed directory exists",
			setup: func(t *testing.T, home string) {
				t.Helper()
				dir := filepath.Join(home, ".config", "bak")
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatal(err)
				}
			},
			wantInstalled: true,
		},
		{
			name: "not installed directory missing",
			setup: func(t *testing.T, home string) {
				t.Helper()
			},
			wantInstalled: false,
		},
		{
			name: "exists but is file not directory",
			setup: func(t *testing.T, home string) {
				t.Helper()
				parent := filepath.Join(home, ".config")
				if err := os.MkdirAll(parent, 0755); err != nil {
					t.Fatal(err)
				}
				filePath := filepath.Join(parent, "bak")
				if err := os.WriteFile(filePath, []byte("file"), 0644); err != nil {
					t.Fatal(err)
				}
			},
			wantInstalled: false,
		},
	}

	for _, tt := range tests { //nolint:paralleltest
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest
			home := t.TempDir()
			tt.setup(t, home)

			installed, gotDir, err := a.Detect(home)
			if err != nil {
				t.Fatalf("Detect error: %v", err)
			}
			if installed != tt.wantInstalled {
				t.Errorf("Detect() installed = %v, want %v", installed, tt.wantInstalled)
			}
			expectedDir := filepath.Join(home, ".config", "bak")
			if gotDir != expectedDir {
				t.Errorf("Detect() gotDir = %q, want %q", gotDir, expectedDir)
			}
		})
	}
}

func TestBakAdapter_ListItems(t *testing.T) { //nolint:paralleltest // subtests share struct state
	a := &Adapter{}

	setupFixture := func(t *testing.T) string {
		t.Helper()
		home := t.TempDir()
		configDir := filepath.Join(home, ".config", "bak")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			t.Fatal(err)
		}

		// Populated config.json
		cfgContent := `{"schema_version":"0.3.0","settings":{"default_preset":"quick"}}`
		if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte(cfgContent), 0644); err != nil {
			t.Fatal(err)
		}

		// Migration leftovers and unlisted files (must not be listed)
		if err := os.WriteFile(filepath.Join(configDir, "config.json.v010.bak"), []byte(`{"legacy":true}`), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(configDir, "config.json.v020.bak"), []byte(`{"legacy2":true}`), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(configDir, "unrelated.txt"), []byte("not in allowlist"), 0644); err != nil {
			t.Fatal(err)
		}

		// Custom adapters directory with a YAML definition
		adaptersDir := filepath.Join(configDir, "adapters")
		if err := os.MkdirAll(adaptersDir, 0755); err != nil {
			t.Fatal(err)
		}
		yamlContent := "name: custom-agent\nconfig_path: .config/custom\n"
		if err := os.WriteFile(filepath.Join(adaptersDir, "custom.yaml"), []byte(yamlContent), 0644); err != nil {
			t.Fatal(err)
		}

		// Backup repository path ~/.bak (must never be captured as input)
		bakRepoDir := filepath.Join(home, ".bak")
		if err := os.MkdirAll(bakRepoDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(bakRepoDir, "repo-artifact.txt"), []byte("output repo content"), 0644); err != nil {
			t.Fatal(err)
		}

		return home
	}

	tests := []struct {
		name           string
		categories     []string
		wantRelPaths   []string
		wantCategories map[string]string
		dontWantPaths  []string
	}{
		{
			name:         "config category lists only config.json",
			categories:   []string{"config"},
			wantRelPaths: []string{"config.json"},
			wantCategories: map[string]string{
				"config.json": "config",
			},
			dontWantPaths: []string{
				"config.json.v010.bak",
				"config.json.v020.bak",
				"unrelated.txt",
				"adapters/custom.yaml",
				"repo-artifact.txt",
			},
		},
		{
			name:         "adapters category lists custom YAML definitions",
			categories:   []string{"adapters"},
			wantRelPaths: []string{"adapters/custom.yaml"},
			wantCategories: map[string]string{
				"adapters/custom.yaml": "adapters",
			},
			dontWantPaths: []string{
				"config.json",
				"config.json.v010.bak",
				"unrelated.txt",
				"repo-artifact.txt",
			},
		},
		{
			name:         "both categories list config.json and adapters",
			categories:   []string{"config", "adapters"},
			wantRelPaths: []string{"config.json", "adapters/custom.yaml"},
			wantCategories: map[string]string{
				"config.json":          "config",
				"adapters/custom.yaml": "adapters",
			},
			dontWantPaths: []string{
				"config.json.v010.bak",
				"config.json.v020.bak",
				"unrelated.txt",
				"repo-artifact.txt",
			},
		},
	}

	for _, tt := range tests { //nolint:paralleltest
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest
			home := setupFixture(t)
			items, err := a.ListItems(home, tt.categories)
			if err != nil {
				t.Fatalf("ListItems: %v", err)
			}

			itemMap := make(map[string]adapters.Item)
			for _, it := range items {
				itemMap[it.RelPath] = it
				// Ensure nothing under the backup repository ~/.bak is ever captured
				if it.RelPath == "repo-artifact.txt" || it.RelPath == ".bak" {
					t.Fatalf("backup repository path was erroneously captured: %+v", it)
				}
			}

			for _, wantPath := range tt.wantRelPaths {
				it, found := itemMap[wantPath]
				if !found {
					t.Errorf("missing expected item %q in %v", wantPath, items)
					continue
				}
				if wantCat, ok := tt.wantCategories[wantPath]; ok {
					if it.Category != wantCat {
						t.Errorf("item %q has Category = %q, want %q", wantPath, it.Category, wantCat)
					}
				}
			}

			for _, dontWant := range tt.dontWantPaths {
				if _, found := itemMap[dontWant]; found {
					t.Errorf("item %q was found in items, but should be excluded by allowlist", dontWant)
				}
			}
		})
	}
}

func TestBakAdapter_BackupAndRestore(t *testing.T) { //nolint:paralleltest // subtests share struct state
	a := &Adapter{}
	home := t.TempDir()
	configDir := filepath.Join(home, ".config", "bak")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfgContent := `{"schema_version":"0.3.0"}`
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte(cfgContent), 0644); err != nil {
		t.Fatal(err)
	}

	items := []adapters.Item{
		{Category: "config", SourcePath: "~/.config/bak/config.json", RelPath: "config.json", IsDir: false, Hash: "sha256:abc", Size: int64(len(cfgContent))},
	}

	backupDir := filepath.Join(t.TempDir(), "backup")
	if err := a.Backup(home, backupDir, items); err != nil {
		t.Fatalf("Backup error: %v", err)
	}

	dstFile := filepath.Join(backupDir, "bak", "config.json")
	data, err := os.ReadFile(dstFile)
	if err != nil {
		t.Fatalf("read backup file: %v", err)
	}
	if string(data) != cfgContent {
		t.Errorf("backup content = %q, want %q", string(data), cfgContent)
	}

	// Now test restore to a fresh target home
	restoreHome := t.TempDir()
	if err := a.Restore(backupDir, restoreHome, items); err != nil {
		t.Fatalf("Restore error: %v", err)
	}

	restoredFile := filepath.Join(restoreHome, ".config", "bak", "config.json")
	restoredData, err := os.ReadFile(restoredFile)
	if err != nil {
		t.Fatalf("read restored file: %v", err)
	}
	if string(restoredData) != cfgContent {
		t.Errorf("restored content = %q, want %q", string(restoredData), cfgContent)
	}
}

func TestBakAdapter_InterfaceCompliance(t *testing.T) { //nolint:paralleltest // shared state isolation
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

	// Verify ScanConfigurable
	a.SetScanOptions(adapters.ScanOptions{Excludes: []string{"*.tmp"}, MaxFileSize: 1024})
}

func TestBakAdapter_Backup_CopyError(t *testing.T) { //nolint:paralleltest // shared state isolation
	if runtime.GOOS == "windows" {
		t.Skip("chmod not applicable on Windows")
	}
	a := &Adapter{}
	home := t.TempDir()
	configDir := filepath.Join(home, ".config", "bak")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}

	unreadableFile := filepath.Join(configDir, "config.json")
	if err := os.WriteFile(unreadableFile, []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(unreadableFile, 0000); err != nil {
		t.Fatal(err)
	}

	items := []adapters.Item{
		{Category: "config", SourcePath: "~/.config/bak/config.json", RelPath: "config.json", IsDir: false, Hash: "sha256:abc", Size: 4},
	}

	backupDir := filepath.Join(t.TempDir(), "backup")
	err := a.Backup(home, backupDir, items)
	if err == nil {
		t.Error("expected error for unreadable file, got nil")
	}
}

func TestBakAdapter_Restore_CopyError(t *testing.T) { //nolint:paralleltest // shared state isolation
	if runtime.GOOS == "windows" {
		t.Skip("chmod not applicable on Windows")
	}
	a := &Adapter{}
	home := t.TempDir()
	configDir := filepath.Join(home, ".config", "bak")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(configDir, 0500); err != nil {
		t.Fatal(err)
	}

	backupDir := filepath.Join(t.TempDir(), "backup")
	srcFile := filepath.Join(backupDir, "bak", "config.json")
	if err := os.MkdirAll(filepath.Dir(srcFile), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(srcFile, []byte(`{"key":"val"}`), 0644); err != nil {
		t.Fatal(err)
	}

	items := []adapters.Item{
		{Category: "config", SourcePath: "~/.config/bak/config.json", RelPath: "config.json", IsDir: false, Hash: "sha256:xyz", Size: 13},
	}

	err := a.Restore(backupDir, home, items)
	if err == nil {
		t.Error("expected error for copy to read-only dir, got nil")
	}
}
