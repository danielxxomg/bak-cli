package cloud

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestTarGz_RoundTrip(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
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

func TestUntarGz_InvalidBase64(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	err := UntarGz("!!!not-valid-base64!!!", t.TempDir())
	if err == nil {
		t.Fatal("expected error for invalid base64")
	}
}

func TestUntarGz_EmptyArchive(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
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

func TestTarGz_DirectoryWithSubdirs(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
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

func TestTarGz_NonexistentDir(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
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
		if (c < 'A' || c > 'Z') && (c < 'a' || c > 'z') &&
			(c < '0' || c > '9') && c != '+' && c != '/' && c != '=' {
			return false
		}
	}
	return len(s) > 0
}

func TestBase64_Detector(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
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

// tarEntry describes a member of a tar archive for testing.
type tarEntry struct {
	Name     string // entry name in archive
	Body     string // file content (ignored for symlinks and dirs)
	Typeflag byte   // tar.TypeReg, tar.TypeDir, tar.TypeSymlink
	Linkname string // symlink target (only for TypeSymlink)
	Mode     int64  // file mode; defaults to 0644 for files, 0755 for dirs
}

// buildTarGz creates a base64-encoded tar.gz archive from the given
// entries. Useful for constructing archives with specific metadata
// (symlinks, traversal payloads, etc.) without touching the filesystem.
func buildTarGz(t *testing.T, entries []tarEntry) string {
	t.Helper()

	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	for _, e := range entries {
		mode := e.Mode
		if mode == 0 {
			switch e.Typeflag {
			case tar.TypeDir:
				mode = 0755
			case tar.TypeSymlink:
				mode = 0644
			default:
				mode = 0644
			}
		}

		hdr := &tar.Header{
			Name:     e.Name,
			Size:     int64(len(e.Body)),
			Typeflag: e.Typeflag,
			Mode:     mode,
		}

		if e.Typeflag == tar.TypeSymlink {
			hdr.Linkname = e.Linkname
		}
		if e.Typeflag == tar.TypeDir {
			if !strings.HasSuffix(hdr.Name, "/") {
				hdr.Name += "/"
			}
			hdr.Size = 0
		}

		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("buildTarGz: write header %q: %v", e.Name, err)
		}
		if e.Typeflag == tar.TypeReg && e.Body != "" {
			if _, err := tw.Write([]byte(e.Body)); err != nil {
				t.Fatalf("buildTarGz: write body %q: %v", e.Name, err)
			}
		}
	}

	if err := tw.Close(); err != nil {
		t.Fatalf("buildTarGz: close tar: %v", err)
	}
	if err := gw.Close(); err != nil {
		t.Fatalf("buildTarGz: close gzip: %v", err)
	}

	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

func TestUntarGzDir_Symlink(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	if runtime.GOOS == "windows" {
		t.Skip("symlink tests require Unix-like filesystem support")
	}

	srcDir := t.TempDir()
	dstDir := t.TempDir()

	// Create a real file and a symlink to it.
	targetFile := filepath.Join(srcDir, "real-config.json")
	mustWrite(t, targetFile, `{"theme":"dark"}`)

	linkPath := filepath.Join(srcDir, "current.json")
	if err := os.Symlink("real-config.json", linkPath); err != nil {
		t.Fatalf("setup symlink: %v", err)
	}

	// Tar.gz the source directory.
	encoded, err := TarGzDirectory(srcDir)
	if err != nil {
		t.Fatalf("TarGzDirectory: %v", err)
	}

	// Untar into destination.
	if err := UntarGz(encoded, dstDir); err != nil {
		t.Fatalf("UntarGz: %v", err)
	}

	// Verify the real file was restored.
	verifyFile(t, dstDir, "real-config.json", `{"theme":"dark"}`)

	// Verify the symlink was restored and resolves correctly.
	restoredLink := filepath.Join(dstDir, "current.json")
	target, err := os.Readlink(restoredLink)
	if err != nil {
		t.Fatalf("symlink not restored: %v", err)
	}
	if target != "real-config.json" {
		t.Fatalf("symlink target = %q, want %q", target, "real-config.json")
	}
}

func TestUntarGzDir_PathTraversal(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	// absEntry returns an OS-specific absolute path that triggers traversal.
	absEntry := func() string {
		if runtime.GOOS == "windows" {
			// Use a different drive letter so filepath.Join discards the
			// target directory and produces a path outside it.
			return `X:\Windows\System32\malicious.exe`
		}
		return "/etc/passwd"
	}()

	tests := []struct {
		name      string
		entryName string
	}{
		{
			name:      "direct traversal — ../etc/passwd",
			entryName: "../etc/passwd",
		},
		{
			name:      "nested traversal — foo/../../etc/passwd",
			entryName: "foo/../../etc/passwd",
		},
		{
			name:      "absolute path",
			entryName: absEntry,
		},
	}

	for _, tt := range tests { //nolint:paralleltest // subtests share table/struct state
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
			encoded := buildTarGz(t, []tarEntry{
				{
					Name:     tt.entryName,
					Body:     "malicious",
					Typeflag: tar.TypeReg,
				},
			})

			dstDir := t.TempDir()
			err := UntarGz(encoded, dstDir)
			if err == nil {
				t.Fatal("expected error for traversal entry, got nil")
			}

			// Verify no file escaped the target directory.
			verifyNoEscape(t, dstDir, tt.entryName)
		})
	}
}

func TestTarGzDir_WalkError(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	if runtime.GOOS == "windows" {
		t.Skip("chmod 0000 does not prevent reading on Windows")
	}

	srcDir := t.TempDir()

	// Create a subdirectory and make it unreadable.
	lockedDir := filepath.Join(srcDir, "locked")
	if err := os.MkdirAll(lockedDir, 0755); err != nil {
		t.Fatalf("setup mkdir: %v", err)
	}
	// Create a file inside the locked dir.
	mustWrite(t, filepath.Join(lockedDir, "secret.txt"), "classified")

	// Make the locked directory unreadable so WalkDir fails.
	if err := os.Chmod(lockedDir, 0000); err != nil {
		t.Fatalf("setup chmod: %v", err)
	}
	t.Cleanup(func() {
		// Restore permissions so TempDir cleanup works.
		_ = os.Chmod(lockedDir, 0755)
	})

	_, err := TarGzDirectory(srcDir)
	if err == nil {
		t.Fatal("expected error when walking unreadable directory, got nil")
	}
}

// verifyNoEscape checks that no file with the given traversal name was
// created outside the target directory.
func verifyNoEscape(t *testing.T, dir, traversalName string) {
	t.Helper()

	// Ensure the traversal path does not exist inside the dir.
	cleanName := filepath.Clean(traversalName)
	abs := filepath.Join(dir, cleanName)
	if _, err := os.Stat(abs); err == nil {
		t.Errorf("traversal file %q was created inside target dir", abs)
	}
}
