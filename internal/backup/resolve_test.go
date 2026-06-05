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
