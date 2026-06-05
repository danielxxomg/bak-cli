package actions

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// OSFileSystem implements FileSystem using the real operating system.
type OSFileSystem struct{}

// Compile-time check.
var _ FileSystem = (*OSFileSystem)(nil)

func (o *OSFileSystem) UserHomeDir() (string, error) {
	dir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("user home dir: %w", err)
	}
	return dir, nil
}

func (o *OSFileSystem) Stat(path string) (os.FileInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat: %w", err)
	}
	return info, nil
}

func (o *OSFileSystem) ReadDir(dirname string) ([]os.DirEntry, error) {
	entries, err := os.ReadDir(dirname)
	if err != nil {
		return nil, fmt.Errorf("read dir: %w", err)
	}
	return entries, nil
}

func (o *OSFileSystem) ReadFile(filename string) ([]byte, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	return data, nil
}

func (o *OSFileSystem) MkdirAll(path string, perm os.FileMode) error {
	if err := os.MkdirAll(path, perm); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	return nil
}

func (o *OSFileSystem) CopyFile(src, dst string) error {
	sf, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source: %w", err)
	}
	defer sf.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("mkdir destination: %w", err)
	}

	df, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create destination: %w", err)
	}
	defer df.Close()

	if _, err := io.Copy(df, sf); err != nil {
		return fmt.Errorf("copy: %w", err)
	}

	return df.Close()
}

func (o *OSFileSystem) RemoveAll(path string) error {
	if err := os.RemoveAll(path); err != nil {
		return fmt.Errorf("remove: %w", err)
	}
	return nil
}

func (o *OSFileSystem) WalkDir(root string, fn fs.WalkDirFunc) error {
	if err := filepath.WalkDir(root, fn); err != nil {
		return fmt.Errorf("walk dir: %w", err)
	}
	return nil
}

func (o *OSFileSystem) WriteFile(filename string, data []byte, perm os.FileMode) error {
	if err := os.WriteFile(filename, data, perm); err != nil {
		return fmt.Errorf("write file: %w", err)
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
		return nil, err
	}
	return &Config{
		SchemaVersion: cfg.SchemaVersion,
	}, nil
}
