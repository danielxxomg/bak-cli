package actions

import (
	"encoding/json"
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
