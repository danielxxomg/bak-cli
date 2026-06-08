package adapters

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestFileHash(t *testing.T) {
	h := sha256.New()
	h.Write([]byte("hello world"))
	knownHash := fmt.Sprintf("sha256:%x", h.Sum(nil))

	tests := []struct {
		name     string
		content  []byte
		wantSize int64
		wantHash string
		wantErr  bool
	}{
		{
			name:     "known content",
			content:  []byte("hello world"),
			wantSize: 11,
			wantHash: knownHash,
		},
		{
			name:     "empty file",
			content:  []byte{},
			wantSize: 0,
			wantHash: "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			fpath := filepath.Join(dir, "test.txt")
			if err := os.WriteFile(fpath, tt.content, 0644); err != nil {
				t.Fatal(err)
			}

			hash, size, err := FileHash(fpath)
			if err != nil {
				t.Fatalf("FileHash: %v", err)
			}
			if size != tt.wantSize {
				t.Errorf("size = %d, want %d", size, tt.wantSize)
			}
			if hash != tt.wantHash {
				t.Errorf("hash = %q, want %q", hash, tt.wantHash)
			}
		})
	}

	t.Run("nonexistent file returns error", func(t *testing.T) {
		_, _, err := FileHash(filepath.Join(t.TempDir(), "missing.txt"))
		if err == nil {
			t.Error("expected error for nonexistent file")
		}
	})
}

func TestCopyFile(t *testing.T) {
	tests := []struct {
		name    string
		content string
		dstDir  string
		wantErr bool
	}{
		{
			name:    "copies content to nested destination",
			content: "copy me",
			dstDir:  "sub",
		},
		{
			name:    "copies to same directory",
			content: "same dir",
			dstDir:  ".",
		},
		{
			name:    "copies empty file",
			content: "",
			dstDir:  ".",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			src := filepath.Join(dir, "src.txt")
			dst := filepath.Join(dir, tt.dstDir, "dst.txt")

			if err := os.WriteFile(src, []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}

			if err := CopyFile(src, dst); err != nil {
				t.Fatalf("CopyFile: %v", err)
			}

			data, err := os.ReadFile(dst)
			if err != nil {
				t.Fatalf("read dst: %v", err)
			}
			if string(data) != tt.content {
				t.Errorf("dst content = %q, want %q", string(data), tt.content)
			}
		})
	}

	t.Run("nonexistent source returns error", func(t *testing.T) {
		dir := t.TempDir()
		err := CopyFile(filepath.Join(dir, "missing.txt"), filepath.Join(dir, "dst.txt"))
		if err == nil {
			t.Error("expected error for nonexistent source")
		}
	})
}
