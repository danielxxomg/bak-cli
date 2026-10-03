package actions

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
)

// osName is the runtime OS identifier, centralized so platform-specific
// behavior has a single reference per package.
var osName = runtime.GOOS

// isWindows reports whether the current platform is Windows.
func isWindows() bool { return osName == "windows" }

// OSFileSystem implements FileSystem using the real operating system.
type OSFileSystem struct{}

// Compile-time check.
var _ FileSystem = (*OSFileSystem)(nil)

// UserHomeDir returns the current user's home directory.
func (o *OSFileSystem) UserHomeDir() (string, error) {
	dir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("user home dir: %w", err)
	}
	return dir, nil
}

// Stat returns file info for path, following symlinks.
func (o *OSFileSystem) Stat(path string) (os.FileInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat: %w", err)
	}
	return info, nil
}

// Lstat returns file info for path without following symlinks.
func (o *OSFileSystem) Lstat(path string) (os.FileInfo, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, fmt.Errorf("lstat: %w", err)
	}
	return info, nil
}

// ReadDir lists the entries of dirname, sorted by filename.
func (o *OSFileSystem) ReadDir(dirname string) ([]os.DirEntry, error) {
	entries, err := os.ReadDir(dirname)
	if err != nil {
		return nil, fmt.Errorf("read dir: %w", err)
	}
	return entries, nil
}

// ReadFile returns the contents of filename.
func (o *OSFileSystem) ReadFile(filename string) ([]byte, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	return data, nil
}

// MkdirAll creates dirname and any missing parents with the given permissions.
func (o *OSFileSystem) MkdirAll(path string, perm os.FileMode) error {
	if err := os.MkdirAll(path, perm); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	return nil
}

// CopyFile copies src to dst, creating parent directories as needed and
// preserving the source permission bits. Close errors are propagated when no
// earlier error occurred.
func (o *OSFileSystem) CopyFile(src, dst string) (retErr error) {
	sf, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source: %w", err)
	}
	defer func() {
		if closeErr := sf.Close(); closeErr != nil && retErr == nil {
			retErr = fmt.Errorf("close source: %w", closeErr)
		}
	}()

	info, err := sf.Stat()
	if err != nil {
		return fmt.Errorf("stat source: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("mkdir destination: %w", err)
	}

	df, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return fmt.Errorf("create destination: %w", err)
	}
	defer func() {
		if closeErr := df.Close(); closeErr != nil && retErr == nil {
			retErr = fmt.Errorf("close destination: %w", closeErr)
		}
	}()

	if _, err := io.Copy(df, sf); err != nil {
		return fmt.Errorf("copy: %w", err)
	}

	if err := os.Chmod(dst, info.Mode().Perm()); err != nil && !isWindows() {
		return fmt.Errorf("chmod destination: %w", err)
	}

	return nil
}

// Remove deletes a single file or empty directory.
func (o *OSFileSystem) Remove(name string) error {
	if err := os.Remove(name); err != nil {
		return fmt.Errorf("remove: %w", err)
	}
	return nil
}

// RemoveAll deletes path and any children. Callers must validate path safety
// before invoking it.
func (o *OSFileSystem) RemoveAll(path string) error {
	if err := os.RemoveAll(path); err != nil {
		return fmt.Errorf("remove: %w", err)
	}
	return nil
}

// WalkDir walks the file tree rooted at root, calling fn for each entry.
func (o *OSFileSystem) WalkDir(root string, fn fs.WalkDirFunc) error {
	if err := filepath.WalkDir(root, fn); err != nil {
		return fmt.Errorf("walk dir: %w", err)
	}
	return nil
}

// WriteFile writes data to filename with the given permissions.
func (o *OSFileSystem) WriteFile(filename string, data []byte, perm os.FileMode) error {
	if err := os.WriteFile(filename, data, perm); err != nil {
		return fmt.Errorf("write file: %w", err)
	}
	return nil
}

// Chmod sets the permission bits of name.
func (o *OSFileSystem) Chmod(name string, mode os.FileMode) error {
	if err := os.Chmod(name, mode); err != nil {
		return fmt.Errorf("chmod: %w", err)
	}
	return nil
}

// RealConfigLoader implements ConfigLoader using the real configuration
// system from internal/config.
//
// It wraps config.Load() and returns an actions.Config with the
// SchemaVersion mapped.
type RealConfigLoader struct{}

// Compile-time check.
var _ ConfigLoader = (*RealConfigLoader)(nil)

// Load reads the real bak-cli configuration from disk.
// It delegates to config.Load() and translates to actions.Config.
func (r *RealConfigLoader) Load() (*Config, error) {
	cfg, err := configLoad()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	return &Config{
		SchemaVersion: cfg.SchemaVersion,
	}, nil
}
