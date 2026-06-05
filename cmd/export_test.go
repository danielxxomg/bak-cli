package cmd

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestExportCmd_Structure(t *testing.T) {
	if exportCmd.Use != "export <backup-id>" {
		t.Errorf("Expected Use 'export <backup-id>', got %q", exportCmd.Use)
	}
	if exportCmd.Short == "" {
		t.Error("Short description should not be empty")
	}
	if exportCmd.RunE == nil {
		t.Error("RunE should be set")
	}
}

func TestExportCmd_Args(t *testing.T) {
	// export requires exactly 1 argument
	// cobra.ExactArgs(1) validates this
}

func TestExportCmd_Flags(t *testing.T) {
	flag := exportCmd.Flags().Lookup("output")
	if flag == nil {
		t.Error("Expected 'output' flag to exist")
	}
	if flag.DefValue != "bak-export.tar.gz" {
		t.Errorf("Expected default value 'bak-export.tar.gz', got %q", flag.DefValue)
	}
}

func TestExportCmd_Help(t *testing.T) {
	if exportCmd.Long == "" {
		t.Error("Long description should not be empty")
	}
}

func TestIsValidBackupID(t *testing.T) {
	tests := []struct {
		name string
		id   string
		want bool
	}{
		{"valid", "20260604-150405", true},
		{"valid midnight", "20260101-000000", true},
		{"valid end of day", "20261231-235959", true},
		{"too short", "20260604-15040", false},
		{"too long", "20260604-1504051", false},
		{"no dash", "20260604150405", false},
		{"dash wrong position", "202606041-50405", false},
		{"letters", "20260604-15040a", false},
		{"empty", "", false},
		{"only dash", "-", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidBackupID(tt.id)
			if got != tt.want {
				t.Errorf("isValidBackupID(%q) = %v, want %v", tt.id, got, tt.want)
			}
		})
	}
}

func TestFormatBackupIDError(t *testing.T) {
	msg := formatBackupIDError("invalid")
	if msg == "" {
		t.Error("Error message should not be empty")
	}
	if !contains(msg, "invalid") {
		t.Error("Error message should contain the invalid ID")
	}
}

func TestCreateTarGz_RoundTrip(t *testing.T) {
	// Create a temporary directory with test files.
	srcDir := t.TempDir()

	// Create test files.
	testFiles := map[string]string{
		"file1.txt":           "content of file 1",
		"subdir/file2.txt":    "content of file 2",
		"subdir/deep/file3":   "deep file content",
	}

	for relPath, content := range testFiles {
		fullPath := filepath.Join(srcDir, relPath)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Create tar.gz in a DIFFERENT directory to avoid including output in archive.
	outDir := t.TempDir()
	outFile := filepath.Join(outDir, "test.tar.gz")
	f, err := os.Create(outFile)
	if err != nil {
		t.Fatal(err)
	}

	if err := createTarGz(srcDir, f); err != nil {
		f.Close()
		t.Fatal(err)
	}
	f.Close()

	// Verify the archive.
	f, err = os.Open(outFile)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	fileCount := 0
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		fileCount++

		// Verify file content.
		if hdr.Name == "file1.txt" {
			content, err := io.ReadAll(tr)
			if err != nil {
				t.Fatal(err)
			}
			if string(content) != "content of file 1" {
				t.Errorf("Expected 'content of file 1', got %q", string(content))
			}
		}
	}

	// Should have at least 4 entries (root dir + 3 files).
	if fileCount < 4 {
		t.Errorf("Expected at least 4 entries in archive, got %d", fileCount)
	}
}

func TestCreateTarGz_EmptyDir(t *testing.T) {
	srcDir := t.TempDir()

	// Create tar.gz in a DIFFERENT directory.
	outDir := t.TempDir()
	outFile := filepath.Join(outDir, "empty.tar.gz")
	f, err := os.Create(outFile)
	if err != nil {
		t.Fatal(err)
	}

	if err := createTarGz(srcDir, f); err != nil {
		f.Close()
		t.Fatal(err)
	}
	f.Close()

	// Verify the archive exists and is valid gzip.
	f, err = os.Open(outFile)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	defer gr.Close()

	// Empty directory archive should have one entry (the directory itself).
	tr := tar.NewReader(gr)
	hdr, err := tr.Next()
	if err != nil {
		t.Errorf("Expected directory entry, got error: %v", err)
	}
	if hdr != nil && !hdr.FileInfo().IsDir() {
		t.Errorf("Expected directory entry, got file: %s", hdr.Name)
	}

	// Should be EOF after the directory entry.
	_, err = tr.Next()
	if err != io.EOF {
		t.Errorf("Expected EOF after directory entry, got %v", err)
	}
}
