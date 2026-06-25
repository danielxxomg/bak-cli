package backup

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestResolveBackupID(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(tmpDir string) string // returns backupID to test
		wantErr   bool
		errSubstr string
		wantDir   func(tmpDir string) string // expected resolved dir path
	}{
		{
			name: "valid backup ID",
			setup: func(tmpDir string) string {
				backupsDir := filepath.Join(tmpDir, ".bak", "backups")
				backupID := "20250101-120000"
				os.MkdirAll(filepath.Join(backupsDir, backupID), 0755)
				return backupID
			},
			wantErr: false,
			wantDir: func(tmpDir string) string {
				return filepath.Join(tmpDir, ".bak", "backups", "20250101-120000")
			},
		},
		{
			name: "missing backup ID",
			setup: func(tmpDir string) string {
				os.MkdirAll(filepath.Join(tmpDir, ".bak", "backups"), 0755)
				return "nonexistent"
			},
			wantErr:   true,
			errSubstr: "not found",
		},
		{
			name: "dot-dot traversal blocked",
			setup: func(tmpDir string) string {
				os.MkdirAll(filepath.Join(tmpDir, ".bak", "backups"), 0755)
				return "../etc"
			},
			wantErr:   true,
			errSubstr: "outside",
		},
		{
			name: "nested dot-dot traversal blocked",
			setup: func(tmpDir string) string {
				os.MkdirAll(filepath.Join(tmpDir, ".bak", "backups"), 0755)
				return "../../etc"
			},
			wantErr:   true,
			errSubstr: "outside",
		},
		{
			name: "dot-dot with valid suffix blocked",
			setup: func(tmpDir string) string {
				os.MkdirAll(filepath.Join(tmpDir, ".bak", "backups"), 0755)
				return "../valid-id"
			},
			wantErr:   true,
			errSubstr: "outside",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			// Override home directory so BakDir() resolves to tmpDir/.bak.
			switch runtime.GOOS {
			case "windows":
				t.Setenv("USERPROFILE", tmpDir)
			default:
				t.Setenv("HOME", tmpDir)
			}

			backupID := tt.setup(tmpDir)
			gotDir, err := ResolveBackupID(backupID)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil (dir=%q)", tt.errSubstr, gotDir)
				}
				if tt.errSubstr != "" && !strings.Contains(err.Error(), tt.errSubstr) {
					t.Errorf("error = %v, want substring %q", err, tt.errSubstr)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			wantDir := tt.wantDir(tmpDir)
			if gotDir != wantDir {
				t.Errorf("dir = %q, want %q", gotDir, wantDir)
			}
		})
	}
}

// =============================================================================
// Phase 6: resolveBackupID consolidation — RED (ListBackupIDs/LatestBackupID absent)
// =============================================================================

func TestListBackupIDs_DescendingSort(t *testing.T) {
	dir := t.TempDir()
	for _, id := range []string{"20260101-120000", "20260102-130000", "20260103-140000"} {
		if err := os.MkdirAll(filepath.Join(dir, id), 0755); err != nil {
			t.Fatal(err)
		}
	}
	// A non-directory entry must be ignored.
	if err := os.WriteFile(filepath.Join(dir, "README.txt"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	ids, err := ListBackupIDs(dir)
	if err != nil {
		t.Fatalf("ListBackupIDs: %v", err)
	}
	if len(ids) != 3 {
		t.Fatalf("len(ids) = %d, want 3 (non-dir entry ignored)", len(ids))
	}
	want := []string{"20260103-140000", "20260102-130000", "20260101-120000"}
	for i, w := range want {
		if ids[i] != w {
			t.Errorf("ids[%d] = %q, want %q (descending)", i, ids[i], w)
		}
	}
}

func TestListBackupIDs_EmptyDirReturnsEmpty(t *testing.T) {
	ids, err := ListBackupIDs(t.TempDir())
	if err != nil {
		t.Fatalf("ListBackupIDs on empty dir: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("len(ids) = %d, want 0", len(ids))
	}
}

func TestListBackupIDs_ReadDirError(t *testing.T) {
	_, err := ListBackupIDs(filepath.Join(t.TempDir(), "does-not-exist"))
	if err == nil {
		t.Fatal("ListBackupIDs on missing dir: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "read backups dir") {
		t.Errorf("error should mention reading backups dir: %v", err)
	}
}

func TestLatestBackupID_ReturnsMostRecent(t *testing.T) {
	dir := t.TempDir()
	for _, id := range []string{"20260101-120000", "20260102-130000", "20260103-140000"} {
		if err := os.MkdirAll(filepath.Join(dir, id), 0755); err != nil {
			t.Fatal(err)
		}
	}
	latest, err := LatestBackupID(dir)
	if err != nil {
		t.Fatalf("LatestBackupID: %v", err)
	}
	if latest != "20260103-140000" {
		t.Errorf("LatestBackupID = %q, want %q", latest, "20260103-140000")
	}
}

func TestLatestBackupID_EmptyDirReturnsError(t *testing.T) {
	_, err := LatestBackupID(t.TempDir())
	if err == nil {
		t.Fatal("LatestBackupID on empty dir: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no backups found") {
		t.Errorf("error should mention no backups found: %v", err)
	}
}

func TestLatestBackupID_ReadDirError(t *testing.T) {
	_, err := LatestBackupID(filepath.Join(t.TempDir(), "nope"))
	if err == nil {
		t.Fatal("LatestBackupID on missing dir: expected error, got nil")
	}
}
