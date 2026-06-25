package actions

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/danielxxomg/bak-cli/internal/manifest"
)

func setupBackupDir(t *testing.T, bakDir string, backupID string, m *manifest.Manifest) string {
	t.Helper()

	backupsDir := filepath.Join(bakDir, "backups")
	if err := os.MkdirAll(backupsDir, 0755); err != nil {
		t.Fatal(err)
	}

	backupDir := filepath.Join(backupsDir, backupID)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatal(err)
	}

	data, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(backupDir, "manifest.json"), data, 0644); err != nil {
		t.Fatal(err)
	}

	return backupDir
}

func TestRunListLocal_EmptyBackupsDir(t *testing.T) {
	bakDir := t.TempDir()
	// No backups/ directory at all.
	var out strings.Builder
	err := RunListLocal(bakDir, false, &out, &strings.Builder{})
	if err != nil {
		t.Fatalf("RunListLocal: %v", err)
	}
	if !strings.Contains(out.String(), "No backups found") {
		t.Errorf("expected 'No backups found', got: %q", out.String())
	}
}

func TestRunListLocal_NoBackups(t *testing.T) {
	bakDir := t.TempDir()
	backupsDir := filepath.Join(bakDir, "backups")
	if err := os.MkdirAll(backupsDir, 0755); err != nil {
		t.Fatal(err)
	}

	var out strings.Builder
	err := RunListLocal(bakDir, false, &out, &strings.Builder{})
	if err != nil {
		t.Fatalf("RunListLocal: %v", err)
	}
	if !strings.Contains(out.String(), "No backups found") {
		t.Errorf("expected 'No backups found', got: %q", out.String())
	}
}

func TestRunListLocal_SingleBackup(t *testing.T) {
	bakDir := t.TempDir()
	m := &manifest.Manifest{
		ID:        "20260101-120000",
		Preset:    "quick",
		TotalSize: 2048,
		FileCount: 2,
		Adapters: map[string]manifest.AdapterManifest{
			"opencode": {Items: make([]manifest.Item, 2)},
		},
	}
	setupBackupDir(t, bakDir, "20260101-120000", m)

	var out strings.Builder
	err := RunListLocal(bakDir, false, &out, &strings.Builder{})
	if err != nil {
		t.Fatalf("RunListLocal: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "20260101-120000") {
		t.Errorf("output should contain backup ID, got: %q", output)
	}
	if !strings.Contains(output, "quick") {
		t.Errorf("output should contain preset, got: %q", output)
	}
	if !strings.Contains(output, "2") {
		t.Errorf("output should contain file count, got: %q", output)
	}
}

func TestRunListLocal_MultipleBackups(t *testing.T) {
	bakDir := t.TempDir()
	m1 := &manifest.Manifest{
		ID:        "20260101-120000",
		Preset:    "quick",
		TotalSize: 1024,
		FileCount: 1,
		Adapters: map[string]manifest.AdapterManifest{
			"opencode": {Items: make([]manifest.Item, 1)},
		},
	}
	m2 := &manifest.Manifest{
		ID:        "20260102-080000",
		Preset:    "full",
		TotalSize: 4096,
		FileCount: 5,
		Adapters: map[string]manifest.AdapterManifest{
			"opencode": {Items: make([]manifest.Item, 3)},
			"cursor":   {Items: make([]manifest.Item, 2)},
		},
	}
	setupBackupDir(t, bakDir, "20260101-120000", m1)
	setupBackupDir(t, bakDir, "20260102-080000", m2)

	var out strings.Builder
	err := RunListLocal(bakDir, false, &out, &strings.Builder{})
	if err != nil {
		t.Fatalf("RunListLocal: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "20260101-120000") {
		t.Error("output should contain first backup ID")
	}
	if !strings.Contains(output, "20260102-080000") {
		t.Error("output should contain second backup ID")
	}
	if !strings.Contains(output, "quick") {
		t.Error("output should contain quick preset")
	}
	if !strings.Contains(output, "full") {
		t.Error("output should contain full preset")
	}
}

func TestRunListLocal_SkipsInvalidManifest(t *testing.T) {
	bakDir := t.TempDir()
	// Backup directory without a valid manifest.
	backupsDir := filepath.Join(bakDir, "backups", "corrupt-backup")
	if err := os.MkdirAll(backupsDir, 0755); err != nil {
		t.Fatal(err)
	}

	var out strings.Builder
	err := RunListLocal(bakDir, false, &out, &strings.Builder{})
	if err != nil {
		t.Fatalf("RunListLocal: %v", err)
	}
	// Table header is printed but no rows (corrupt backup skipped).
	if !strings.Contains(out.String(), "ID") {
		t.Errorf("expected table header, got: %q", out.String())
	}
	if strings.Contains(out.String(), "corrupt-backup") {
		t.Errorf("corrupt backup should not appear in output: %q", out.String())
	}
}

func TestRunListLocal_VerboseWarns(t *testing.T) {
	bakDir := t.TempDir()
	// Create a corrupt backup directory (no manifest) to trigger verbose warning.
	backupsDir := filepath.Join(bakDir, "backups", "corrupt-backup")
	if err := os.MkdirAll(backupsDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Also create a valid backup to ensure it still lists.
	m := &manifest.Manifest{
		ID:        "20260101-120000",
		Preset:    "quick",
		TotalSize: 1024,
		FileCount: 1,
		Adapters: map[string]manifest.AdapterManifest{
			"opencode": {Items: make([]manifest.Item, 1)},
		},
	}
	setupBackupDir(t, bakDir, "20260101-120000", m)

	var out, errOut strings.Builder
	err := RunListLocal(bakDir, true, &out, &errOut)
	if err != nil {
		t.Fatalf("RunListLocal: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "20260101-120000") {
		t.Errorf("valid backup should still be listed: %q", output)
	}
	// Verbose warning about corrupt backup goes to errOut, not out.
	if strings.Contains(output, "corrupt-backup") {
		t.Error("corrupt backup should not appear in main output")
	}
	if !strings.Contains(errOut.String(), "corrupt-backup") {
		t.Errorf("verbose warning should mention corrupt backup, errOut=%q", errOut.String())
	}
}

func TestFormatSizeBytes(t *testing.T) {
	tests := []struct {
		name  string
		bytes int64
		want  string
	}{
		// Zero and sub-KB range.
		{name: "zero", bytes: 0, want: "0 B"},
		{name: "one byte", bytes: 1, want: "1 B"},
		{name: "sub-KB max", bytes: 1023, want: "1023 B"},

		// KB boundary.
		{name: "exactly 1 KB", bytes: 1024, want: "1.0 KB"},
		{name: "just above 1 KB", bytes: 1025, want: "1.0 KB"},
		{name: "1.5 KB", bytes: 1536, want: "1.5 KB"},

		// MB boundary.
		{name: "exactly 1 MB", bytes: 1048576, want: "1.0 MB"},
		{name: "just below 1 MB", bytes: 1048575, want: "1024.0 KB"},
		{name: "just above 1 MB", bytes: 1048577, want: "1.0 MB"},

		// GB boundary.
		{name: "exactly 1 GB", bytes: 1073741824, want: "1.0 GB"},
		{name: "just below 1 GB", bytes: 1073741823, want: "1024.0 MB"},
		{name: "just above 1 GB", bytes: 1073741825, want: "1.0 GB"},

		// TB — exercises code beyond GB (RED: code only goes to GB).
		{name: "exactly 1 TB", bytes: 1099511627776, want: "1.0 TB"},
		{name: "just below 1 TB", bytes: 1099511627775, want: "1024.0 GB"},
		{name: "1.5 TB", bytes: 1649267441664, want: "1.5 TB"},

		// PB — exercises higher magnitudes.
		{name: "exactly 1 PB", bytes: 1125899906842624, want: "1.0 PB"},

		// Edge cases.
		{name: "negative value", bytes: -500, want: "-500 B"},
		{name: "max int64", bytes: math.MaxInt64, want: "8.0 EB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatSizeBytes(tt.bytes)
			if got != tt.want {
				t.Errorf("FormatSizeBytes(%d) = %q, want %q", tt.bytes, got, tt.want)
			}
		})
	}
}

// --- extracted helper tests (Phase 9) ---

// TestFormatBackupDate verifies the backup-ID-to-date parser.
func TestFormatBackupDate(t *testing.T) {
	tests := []struct {
		name     string
		backupID string
		want     string
	}{
		{
			name:     "full timestamp",
			backupID: "20260625-143012",
			want:     "2026-06-25 14:30:12",
		},
		{
			name:     "too short returns empty",
			backupID: "2026062",
			want:     "",
		},
		{
			name:     "exactly 15 chars parses",
			backupID: "20260101-000000",
			want:     "2026-01-01 00:00:00",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatBackupDate(tt.backupID); got != tt.want {
				t.Errorf("formatBackupDate(%q) = %q, want %q", tt.backupID, got, tt.want)
			}
		})
	}
}

// TestFormatBackupRow verifies the row formatter produces a tab-separated
// line with sorted adapters and the parsed date.
func TestFormatBackupRow(t *testing.T) {
	m := &manifest.Manifest{
		Preset:    "full",
		FileCount: 7,
		TotalSize: 2048,
		Adapters: map[string]manifest.AdapterManifest{
			"cursor":   {},
			"opencode": {},
			"codex":    {},
		},
	}

	row := formatBackupRow("20260625-143012", m)

	if !strings.HasPrefix(row, "20260625-143012\t") {
		t.Errorf("row should start with backup ID; got: %q", row)
	}
	if !strings.Contains(row, "2026-06-25 14:30:12") {
		t.Errorf("row should contain parsed date; got: %q", row)
	}
	if !strings.Contains(row, "full") {
		t.Errorf("row should contain preset; got: %q", row)
	}
	if !strings.Contains(row, "\t7\t") {
		t.Errorf("row should contain file count 7; got: %q", row)
	}
	// Adapters must be sorted alphabetically: codex, cursor, opencode.
	if !strings.Contains(row, "codex, cursor, opencode") {
		t.Errorf("row should contain sorted adapters; got: %q", row)
	}
	if !strings.HasSuffix(row, "\n") {
		t.Errorf("row should end with newline; got: %q", row)
	}
}
