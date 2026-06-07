package actions

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunExport_HappyPath(t *testing.T) {
	homeDir := t.TempDir()
	backupID := "20260101-120000"

	// Create source backup directory with a file.
	sourceDir := filepath.Join(homeDir, ".bak", "backups", backupID)
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "test.txt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	outputPath := filepath.Join(t.TempDir(), "export.tar.gz")
	var out strings.Builder

	err := RunExport(homeDir, backupID, outputPath, &out)
	if err != nil {
		t.Fatalf("RunExport: %v", err)
	}

	// Verify output file exists and is a valid tar.gz.
	if _, err := os.Stat(outputPath); err != nil {
		t.Fatalf("output file not created: %v", err)
	}

	f, err := os.Open(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatalf("gzip reader: %v", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	// Skip directory entry (first).
	_, err = tr.Next()
	if err != nil {
		t.Fatalf("tar next (dir): %v", err)
	}

	// Read file entry (second).
	hdr, err := tr.Next()
	if err != nil {
		t.Fatalf("tar next (file): %v", err)
	}
	if !strings.HasSuffix(hdr.Name, "test.txt") {
		t.Errorf("entry name should end with test.txt, got %q", hdr.Name)
	}

	body, err := io.ReadAll(tr)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "hello" {
		t.Errorf("file content = %q, want hello", string(body))
	}

	// Confirm success message.
	if !strings.Contains(out.String(), backupID) {
		t.Errorf("output should mention backup ID: %q", out.String())
	}
}

func TestRunExport_BackupNotFound(t *testing.T) {
	homeDir := t.TempDir()
	outputPath := filepath.Join(t.TempDir(), "export.tar.gz")
	var out strings.Builder

	err := RunExport(homeDir, "20260101-120000", outputPath, &out)
	if err == nil {
		t.Fatal("expected error for missing backup")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention not found: %v", err)
	}
}

func TestRunExport_NotADirectory(t *testing.T) {
	homeDir := t.TempDir()
	backupID := "20260101-120000"

	// Create a file instead of a directory at the expected backup path.
	backupsDir := filepath.Join(homeDir, ".bak", "backups", backupID)
	if err := os.MkdirAll(filepath.Dir(backupsDir), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(backupsDir, []byte("not a dir"), 0644); err != nil {
		t.Fatal(err)
	}

	outputPath := filepath.Join(t.TempDir(), "export.tar.gz")
	var out strings.Builder

	err := RunExport(homeDir, backupID, outputPath, &out)
	if err == nil {
		t.Fatal("expected error for non-directory path")
	}
	if !strings.Contains(err.Error(), "not a directory") {
		t.Errorf("error should mention not a directory: %v", err)
	}
}
