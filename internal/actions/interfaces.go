// Package actions provides the core backup, restore, push, and pull
// workflows as injectable action structs with FileSystem and ConfigLoader
// dependencies, enabling full unit-test coverage.
package actions

import (
	"io/fs"
	"os"
)

// FileSystem abstracts OS file operations for testability.
type FileSystem interface {
	// UserHomeDir returns the user's home directory.
	UserHomeDir() (string, error)

	// Stat returns file info for the given path.
	Stat(path string) (os.FileInfo, error)

	// ReadDir reads the directory named by dirname.
	ReadDir(dirname string) ([]os.DirEntry, error)

	// ReadFile reads the file named by filename.
	ReadFile(filename string) ([]byte, error)

	// MkdirAll creates a directory path and all parents.
	MkdirAll(path string, perm os.FileMode) error

	// CopyFile copies a file from src to dst.
	CopyFile(src, dst string) error

	// RemoveAll removes path and any children it contains.
	RemoveAll(path string) error

	// WalkDir walks the file tree rooted at root.
	WalkDir(root string, fn fs.WalkDirFunc) error

	// WriteFile writes data to a file named by filename.
	WriteFile(filename string, data []byte, perm os.FileMode) error
}

// ConfigLoader abstracts configuration loading for testability.
type ConfigLoader interface {
	// Load reads and parses the bak-cli configuration.
	Load() (*Config, error)
}

// Config is a simplified config type used by the actions package.
// It mirrors the config.Config type from internal/config without
// introducing an import cycle.
type Config struct {
	SchemaVersion string
}
