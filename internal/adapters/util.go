package adapters

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// CopyFile copies a regular file from src to dst, creating parent
// directories as needed. It preserves permission bits from src.
//
// Callers MUST validate that src and dst paths are within the expected
// boundary (e.g., under the user's home directory) before calling this
// function — no path traversal check is performed here.
func CopyFile(src, dst string) error {
	sf, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open src: %w", err)
	}
	defer func() { _ = sf.Close() }()

	info, err := sf.Stat()
	if err != nil {
		return fmt.Errorf("stat src: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	df, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return fmt.Errorf("create dst: %w", err)
	}

	// Copy content first, then close to capture flush errors.
	if _, err := io.Copy(df, sf); err != nil {
		_ = df.Close()
		return fmt.Errorf("copy: %w", err)
	}

	if err := df.Close(); err != nil {
		return fmt.Errorf("close dst: %w", err)
	}

	// Explicitly apply permission bits to bypass umask. On Windows, Chmod
	// is a no-op for exec bits and must not error for mode-only reasons.
	if err := os.Chmod(dst, info.Mode().Perm()); err != nil && !isWindows() {
		return fmt.Errorf("chmod dst: %w", err)
	}
	return nil
}

// FileHash computes the SHA-256 hex digest and file size for the given
// regular file path.
func FileHash(path string) (hash string, size int64, err error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, fmt.Errorf("open file: %w", err)
	}
	defer func() { _ = f.Close() }()

	info, err := f.Stat()
	if err != nil {
		return "", 0, fmt.Errorf("stat file: %w", err)
	}

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", 0, fmt.Errorf("hash content: %w", err)
	}

	return fmt.Sprintf("sha256:%x", h.Sum(nil)), info.Size(), nil
}
