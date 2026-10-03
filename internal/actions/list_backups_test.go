package actions

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	configtest "github.com/danielxxomg/bak-cli/internal/config/testutil"
)

func TestListBackupsAction_EmptyDirectory(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state isolation pending
	tests := []struct {
		name      string
		setup     func(t *testing.T, bakDir string)
		wantCount int
	}{
		{
			name:      "missing_backups_dir_returns_nil",
			setup:     func(t *testing.T, bakDir string) {},
			wantCount: 0,
		},
		{
			name: "empty_backups_dir_returns_empty",
			setup: func(t *testing.T, bakDir string) {
				if err := os.MkdirAll(filepath.Join(bakDir, "backups"), 0755); err != nil {
					t.Fatal(err)
				}
			},
			wantCount: 0,
		},
	}

	for _, tt := range tests { //nolint:paralleltest // subtests share table/struct state
		t.Run(tt.name, func(t *testing.T) {
			bakDir := t.TempDir()
			tt.setup(t, bakDir)

			action := ListBackupsAction{
				FS:     &OSFileSystem{},
				BakDir: bakDir,
			}
			backups, err := action.Run()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(backups) != tt.wantCount {
				t.Errorf("len(backups) = %d, want %d", len(backups), tt.wantCount)
			}
		})
	}
}

func TestListBackupsAction_CorruptManifest(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state isolation pending
	bakDir := t.TempDir()
	backupsDir := filepath.Join(bakDir, "backups")

	// Valid backup
	validDir := filepath.Join(backupsDir, "20260101-120000")
	if err := os.MkdirAll(validDir, 0755); err != nil {
		t.Fatal(err)
	}
	validManifest := `{
		"version": "0.3.0",
		"id": "20260101-120000",
		"created_at": "2026-01-01T12:00:00Z",
		"os_source": "linux",
		"bak_version": "1.0.0",
		"preset": "quick",
		"categories": ["config"],
		"adapters": {
			"opencode": {
				"config_dir": "~/.config/opencode",
				"items": []
			}
		},
		"secrets_excluded": false,
		"file_count": 5,
		"total_size": 1024
	}`
	if err := os.WriteFile(filepath.Join(validDir, "manifest.json"), []byte(validManifest), 0644); err != nil {
		t.Fatal(err)
	}

	// Corrupt manifest
	corruptDir := filepath.Join(backupsDir, "20260102-120000")
	if err := os.MkdirAll(corruptDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(corruptDir, "manifest.json"), []byte("invalid json {{"), 0644); err != nil {
		t.Fatal(err)
	}

	// Missing manifest
	missingDir := filepath.Join(backupsDir, "20260103-120000")
	if err := os.MkdirAll(missingDir, 0755); err != nil {
		t.Fatal(err)
	}

	action := ListBackupsAction{
		FS:     &OSFileSystem{},
		BakDir: bakDir,
	}
	backups, err := action.Run()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(backups) != 1 {
		t.Fatalf("len(backups) = %d, want 1 (corrupt and missing skipped)", len(backups))
	}
	if backups[0].ID != "20260101-120000" {
		t.Errorf("backups[0].ID = %q, want %q", backups[0].ID, "20260101-120000")
	}
	if backups[0].Cloud != "opencode" {
		t.Errorf("backups[0].Cloud = %q, want %q", backups[0].Cloud, "opencode")
	}
	if backups[0].Status != "ok" {
		t.Errorf("backups[0].Status = %q, want %q", backups[0].Status, "ok")
	}
	if backups[0].Size != "1.0 KB" {
		t.Errorf("backups[0].Size = %q, want %q", backups[0].Size, "1.0 KB")
	}
}

func TestListBackupsAction_InvalidBackupIDFiltering(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state isolation pending
	bakDir := t.TempDir()
	backupsDir := filepath.Join(bakDir, "backups")

	// Standard valid ID format: YYYYMMDD-HHMMSS
	validDir := filepath.Join(backupsDir, "20260101-120000")
	if err := os.MkdirAll(validDir, 0755); err != nil {
		t.Fatal(err)
	}
	validManifest := `{
		"version": "0.3.0",
		"id": "20260101-120000",
		"created_at": "2026-01-01T12:00:00Z",
		"os_source": "linux",
		"bak_version": "1.0.0",
		"preset": "quick",
		"categories": ["config"],
		"adapters": {},
		"secrets_excluded": false,
		"file_count": 1,
		"total_size": 512
	}`
	if err := os.WriteFile(filepath.Join(validDir, "manifest.json"), []byte(validManifest), 0644); err != nil {
		t.Fatal(err)
	}

	// Custom/non-standard ID format
	customDir := filepath.Join(backupsDir, "named-backup")
	if err := os.MkdirAll(customDir, 0755); err != nil {
		t.Fatal(err)
	}
	customManifest := `{
		"version": "0.3.0",
		"id": "named-backup",
		"created_at": "2026-01-01T12:00:00Z",
		"os_source": "linux",
		"bak_version": "1.0.0",
		"preset": "quick",
		"categories": ["config"],
		"adapters": {},
		"secrets_excluded": false,
		"file_count": 1,
		"total_size": 512
	}`
	if err := os.WriteFile(filepath.Join(customDir, "manifest.json"), []byte(customManifest), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name             string
		filterInvalidIDs bool
		wantIDs          []string
	}{
		{
			name:             "without_filter_returns_all_valid_manifests",
			filterInvalidIDs: false,
			wantIDs:          []string{"20260101-120000", "named-backup"},
		},
		{
			name:             "with_filter_returns_only_valid_backup_ids",
			filterInvalidIDs: true,
			wantIDs:          []string{"20260101-120000"},
		},
	}

	for _, tt := range tests { //nolint:paralleltest // subtests share table/struct state
		t.Run(tt.name, func(t *testing.T) {
			action := ListBackupsAction{
				FS:               &OSFileSystem{},
				BakDir:           bakDir,
				FilterInvalidIDs: tt.filterInvalidIDs,
			}
			backups, err := action.Run()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(backups) != len(tt.wantIDs) {
				t.Fatalf("len(backups) = %d, want %d", len(backups), len(tt.wantIDs))
			}
			for i, want := range tt.wantIDs {
				if backups[i].ID != want {
					t.Errorf("backups[%d].ID = %q, want %q", i, backups[i].ID, want)
				}
			}
		})
	}
}

func TestListBackupsAction_DeterministicOrdering(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state isolation pending
	bakDir := t.TempDir()
	backupsDir := filepath.Join(bakDir, "backups")

	ids := []string{"20260102-120000", "20260101-120000", "20260103-120000"}
	for _, id := range ids {
		dir := filepath.Join(backupsDir, id)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		manifestContent := `{"version":"0.3.0","id":"` + id + `","adapters":{}}`
		if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(manifestContent), 0644); err != nil {
			t.Fatal(err)
		}
	}

	tests := []struct {
		name           string
		sortDescending bool
		wantOrder      []string
	}{
		{
			name:           "default_ascending_order",
			sortDescending: false,
			wantOrder:      []string{"20260101-120000", "20260102-120000", "20260103-120000"},
		},
		{
			name:           "descending_order",
			sortDescending: true,
			wantOrder:      []string{"20260103-120000", "20260102-120000", "20260101-120000"},
		},
	}

	for _, tt := range tests { //nolint:paralleltest // subtests share table/struct state
		t.Run(tt.name, func(t *testing.T) {
			action := ListBackupsAction{
				FS:             &OSFileSystem{},
				BakDir:         bakDir,
				SortDescending: tt.sortDescending,
			}
			backups, err := action.Run()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(backups) != len(tt.wantOrder) {
				t.Fatalf("len(backups) = %d, want %d", len(backups), len(tt.wantOrder))
			}
			for i, want := range tt.wantOrder {
				if backups[i].ID != want {
					t.Errorf("backups[%d].ID = %q, want %q", i, backups[i].ID, want)
				}
			}
		})
	}
}

func TestListBackupsAction_AdapterCloudHintDeterministic(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state isolation pending
	bakDir := t.TempDir()
	backupsDir := filepath.Join(bakDir, "backups")

	// Multi-adapter backup: ghostty and neovim
	multiDir := filepath.Join(backupsDir, "20260101-120000")
	if err := os.MkdirAll(multiDir, 0755); err != nil {
		t.Fatal(err)
	}
	multiManifest := `{
		"version": "0.3.0",
		"id": "20260101-120000",
		"adapters": {
			"neovim": {"config_dir": "~/.config/nvim", "items": []},
			"ghostty": {"config_dir": "~/.config/ghostty", "items": []}
		}
	}`
	if err := os.WriteFile(filepath.Join(multiDir, "manifest.json"), []byte(multiManifest), 0644); err != nil {
		t.Fatal(err)
	}

	// Zero-adapter backup
	zeroDir := filepath.Join(backupsDir, "20260102-120000")
	if err := os.MkdirAll(zeroDir, 0755); err != nil {
		t.Fatal(err)
	}
	zeroManifest := `{"version": "0.3.0", "id": "20260102-120000", "adapters": {}}`
	if err := os.WriteFile(filepath.Join(zeroDir, "manifest.json"), []byte(zeroManifest), 0644); err != nil {
		t.Fatal(err)
	}

	action := ListBackupsAction{
		FS:     &OSFileSystem{},
		BakDir: bakDir,
	}
	backups, err := action.Run()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(backups) != 2 {
		t.Fatalf("len(backups) = %d, want 2", len(backups))
	}

	// Alphabetically sorted adapter names -> "ghostty" comes before "neovim"
	if backups[0].Cloud != "ghostty" {
		t.Errorf("backups[0].Cloud = %q, want %q", backups[0].Cloud, "ghostty")
	}
	if backups[1].Cloud != "none" {
		t.Errorf("backups[1].Cloud = %q, want %q", backups[1].Cloud, "none")
	}
}

func TestListBackupsAction_SkipsNonDirectories(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state isolation pending
	bakDir := t.TempDir()
	backupsDir := filepath.Join(bakDir, "backups")
	if err := os.MkdirAll(backupsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(backupsDir, "not-a-dir.txt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	action := ListBackupsAction{
		FS:     &OSFileSystem{},
		BakDir: bakDir,
	}
	backups, err := action.Run()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(backups) != 0 {
		t.Errorf("len(backups) = %d, want 0", len(backups))
	}
}

func TestListBackupsAction_ReadDirError(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state isolation pending
	mockFS := &MockFileSystem{
		ReadDirErrors: map[string]error{
			filepath.Join("/mock/bak", "backups"): errors.New("simulated I/O failure"),
		},
	}

	action := ListBackupsAction{
		FS:     mockFS,
		BakDir: "/mock/bak",
	}
	_, err := action.Run()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestListBackupsAction_ZeroValueUsable(t *testing.T) { //nolint:paralleltest // sets environment variables via SetConfigHome
	home := t.TempDir()
	configtest.SetConfigHome(t, home)

	var action ListBackupsAction
	backups, err := action.Run()
	if err != nil {
		t.Fatalf("unexpected error for zero-value action: %v", err)
	}
	if len(backups) != 0 {
		t.Errorf("expected 0 backups on fresh home, got %d", len(backups))
	}
}
