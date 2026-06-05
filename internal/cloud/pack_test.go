package cloud

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTarGz_RoundTrip(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	// Create files and directories in source.
	mustWrite(t, filepath.Join(srcDir, "manifest.json"), `{"version": "1.0"}`)
	mustWrite(t, filepath.Join(srcDir, "data", "config.json"), `{"key": "value"}`)
	mustWrite(t, filepath.Join(srcDir, "data", "nested", "deep.txt"), "deep content")
	mustWrite(t, filepath.Join(srcDir, ".env.example"), "SECRET=<YOUR_SECRET>")

	// Tar.gz the source.
	encoded, err := TarGzDirectory(srcDir)
	if err != nil {
		t.Fatalf("TarGzDirectory: %v", err)
	}
	if encoded == "" {
		t.Fatal("expected non-empty base64 output")
	}
	if !isBase64(encoded) {
		t.Error("output should be valid base64")
	}

	// Untar into destination.
	if err := UntarGz(encoded, dstDir); err != nil {
		t.Fatalf("UntarGz: %v", err)
	}

	// Verify all files exist with correct content.
	verifyFile(t, dstDir, "manifest.json", `{"version": "1.0"}`)
	verifyFile(t, dstDir, filepath.Join("data", "config.json"), `{"key": "value"}`)
	verifyFile(t, dstDir, filepath.Join("data", "nested", "deep.txt"), "deep content")
	verifyFile(t, dstDir, ".env.example", "SECRET=<YOUR_SECRET>")
}

func TestUntarGz_InvalidBase64(t *testing.T) {
	err := UntarGz("!!!not-valid-base64!!!", t.TempDir())
	if err == nil {
		t.Fatal("expected error for invalid base64")
	}
}

func TestUntarGz_EmptyArchive(t *testing.T) {
	// Create an empty tar.gz manually.
	encoded, err := TarGzDirectory(t.TempDir())
	if err != nil {
		t.Fatalf("TarGzDirectory empty dir: %v", err)
	}

	dstDir := t.TempDir()
	if err := UntarGz(encoded, dstDir); err != nil {
		t.Fatalf("UntarGz empty archive: %v", err)
	}
}

func TestTarGz_DirectoryWithSubdirs(t *testing.T) {
	srcDir := t.TempDir()

	// Create a nested directory structure with no files in one subdir.
	mustMkdir(t, filepath.Join(srcDir, "empty-dir"))
	mustWrite(t, filepath.Join(srcDir, "a", "b", "c.txt"), "nested")
	mustWrite(t, filepath.Join(srcDir, "root.txt"), "root")

	encoded, err := TarGzDirectory(srcDir)
	if err != nil {
		t.Fatalf("TarGzDirectory: %v", err)
	}

	dstDir := t.TempDir()
	if err := UntarGz(encoded, dstDir); err != nil {
		t.Fatalf("UntarGz: %v", err)
	}

	// Verify directories are preserved.
	if _, err := os.Stat(filepath.Join(dstDir, "empty-dir")); os.IsNotExist(err) {
		t.Error("empty-dir should exist")
	}
	verifyFile(t, dstDir, filepath.Join("a", "b", "c.txt"), "nested")
	verifyFile(t, dstDir, "root.txt", "root")
}

func TestTarGz_NonexistentDir(t *testing.T) {
	_, err := TarGzDirectory("/nonexistent/path/for/testing")
	if err == nil {
		t.Fatal("expected error for nonexistent directory")
	}
}

// --- helpers ---

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}

func verifyFile(t *testing.T, dir, relPath, wantContent string) {
	t.Helper()
	abs := filepath.Join(dir, relPath)
	data, err := os.ReadFile(abs)
	if err != nil {
		t.Errorf("read %s: %v", relPath, err)
		return
	}
	if string(data) != wantContent {
		t.Errorf("%s: content = %q, want %q", relPath, string(data), wantContent)
	}
}

func isBase64(s string) bool {
	for _, c := range s {
		if !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') ||
			(c >= '0' && c <= '9') || c == '+' || c == '/' || c == '=') {
			return false
		}
	}
	return len(s) > 0
}

func TestBase64_Detector(t *testing.T) {
	if !isBase64("SGVsbG8gV29ybGQ=") {
		t.Error("valid base64 should pass")
	}
	if isBase64("hello world!!!") {
		t.Error("invalid base64 should fail")
	}
	if isBase64("") {
		t.Error("empty string should fail")
	}
}
