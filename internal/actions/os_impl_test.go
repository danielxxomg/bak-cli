package actions

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOSFileSystem_Stat_HappyPath(t *testing.T) {
	fsys := &OSFileSystem{}
	tmpDir := t.TempDir()
	info, err := fsys.Stat(tmpDir)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected directory")
	}
}

func TestOSFileSystem_Stat_NotFound(t *testing.T) {
	fsys := &OSFileSystem{}
	_, err := fsys.Stat(filepath.Join(t.TempDir(), "nonexistent"))
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestOSFileSystem_ReadDir_HappyPath(t *testing.T) {
	fsys := &OSFileSystem{}
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "a.txt"), []byte("a"), 0644); err != nil {
		t.Fatal(err)
	}
	entries, err := fsys.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(entries))
	}
}

func TestOSFileSystem_ReadDir_NotFound(t *testing.T) {
	fsys := &OSFileSystem{}
	_, err := fsys.ReadDir(filepath.Join(t.TempDir(), "nonexistent"))
	if err == nil {
		t.Fatal("expected error for nonexistent dir")
	}
}

func TestOSFileSystem_ReadFile_HappyPath(t *testing.T) {
	fsys := &OSFileSystem{}
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(path, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	data, err := fsys.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "hello" {
		t.Errorf("data = %q, want hello", string(data))
	}
}

func TestOSFileSystem_ReadFile_NotFound(t *testing.T) {
	fsys := &OSFileSystem{}
	_, err := fsys.ReadFile(filepath.Join(t.TempDir(), "nonexistent"))
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestOSFileSystem_MkdirAll(t *testing.T) {
	fsys := &OSFileSystem{}
	path := filepath.Join(t.TempDir(), "a", "b", "c")
	if err := fsys.MkdirAll(path, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
}

func TestOSFileSystem_CopyFile_HappyPath(t *testing.T) {
	fsys := &OSFileSystem{}
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "src.txt")
	if err := os.WriteFile(src, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(tmpDir, "dst.txt")
	if err := fsys.CopyFile(src, dst); err != nil {
		t.Fatalf("CopyFile: %v", err)
	}
	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "content" {
		t.Errorf("data = %q, want content", string(data))
	}
}

func TestOSFileSystem_CopyFile_SourceNotFound(t *testing.T) {
	fsys := &OSFileSystem{}
	err := fsys.CopyFile(filepath.Join(t.TempDir(), "nonexistent"), filepath.Join(t.TempDir(), "dst.txt"))
	if err == nil {
		t.Fatal("expected error for missing source")
	}
}

func TestOSFileSystem_RemoveAll(t *testing.T) {
	fsys := &OSFileSystem{}
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "to-remove")
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatal(err)
	}
	if err := fsys.RemoveAll(path); err != nil {
		t.Fatalf("RemoveAll: %v", err)
	}
}

func TestOSFileSystem_WalkDir(t *testing.T) {
	fsys := &OSFileSystem{}
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "a.txt"), []byte("a"), 0644); err != nil {
		t.Fatal(err)
	}
	count := 0
	err := fsys.WalkDir(tmpDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		count++
		return nil
	})
	if err != nil {
		t.Fatalf("WalkDir: %v", err)
	}
	if count < 2 { // root + a.txt
		t.Errorf("expected at least 2 entries, got %d", count)
	}
}

func TestOSFileSystem_WriteFile(t *testing.T) {
	fsys := &OSFileSystem{}
	path := filepath.Join(t.TempDir(), "test.txt")
	if err := fsys.WriteFile(path, []byte("data"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "data" {
		t.Errorf("data = %q, want data", string(data))
	}
}
