package actions

import (
	"io/fs"
	"os"
	"path/filepath"
)

// MockFileSystem implements FileSystem with configurable behavior for
// testing. Every method can be controlled via the struct fields.
//
// Usage:
//
//	m := &MockFileSystem{
//	    HomeDir: "/home/test",
//	    Files:   map[string][]byte{"/config.yaml": []byte("name: test")},
//	    StatResult: map[string]MockStatResult{
//	        "/exists": {Info: mockFileInfo(...)},
//	    },
//	}
type MockFileSystem struct {
	// HomeDir is the value returned by UserHomeDir().
	HomeDir string

	// StatResult maps path → (Info, Err). Missing keys return os.ErrNotExist.
	StatResult map[string]MockStatResult

	// DirEntries maps dir paths to their entries. Missing keys return empty.
	DirEntries map[string][]os.DirEntry

	// ReadDirErrors maps dir paths to explicit errors.
	ReadDirErrors map[string]error

	// Files maps file paths to their byte content.
	Files map[string][]byte

	// ReadFileErrors maps file paths to explicit errors. Missing keys in
	// both Files and ReadFileErrors return os.ErrNotExist.
	ReadFileErrors map[string]error

	// MkdirErrors maps paths to explicit errors.
	MkdirErrors map[string]error

	// CopyErrors maps src paths to explicit errors.
	CopyErrors map[string]error

	// RemoveErrors maps paths to explicit errors.
	RemoveErrors map[string]error

	// WalkErrors maps root paths to explicit errors.
	WalkErrors map[string]error

	// WriteErrors maps paths to explicit errors.
	WriteErrors map[string]error
}

// Compile-time check.
var _ FileSystem = (*MockFileSystem)(nil)

// MockStatResult pairs a file info with an optional error for Stat().
type MockStatResult struct {
	Info os.FileInfo
	Err  error
}

// UserHomeDir returns the configured HomeDir.
func (m *MockFileSystem) UserHomeDir() (string, error) {
	return m.HomeDir, nil
}

// Stat returns file info from the StatResult map, or os.ErrNotExist if the
// path is not configured.
func (m *MockFileSystem) Stat(path string) (os.FileInfo, error) {
	if r, ok := m.StatResult[path]; ok {
		return r.Info, r.Err
	}
	return nil, os.ErrNotExist
}

// ReadDir returns the configured DirEntries for a path, or an empty slice
// if the path is not configured. Explicit errors in ReadDirErrors take
// priority.
func (m *MockFileSystem) ReadDir(dirname string) ([]os.DirEntry, error) {
	if err, ok := m.ReadDirErrors[dirname]; ok {
		return nil, err
	}
	entries, ok := m.DirEntries[dirname]
	if !ok {
		return nil, nil
	}
	return entries, nil
}

// ReadFile returns file content from the Files map. Returns os.ErrNotExist
// if the path is not configured. Explicit errors in ReadFileErrors take
// priority.
func (m *MockFileSystem) ReadFile(filename string) ([]byte, error) {
	if err, ok := m.ReadFileErrors[filename]; ok {
		return nil, err
	}
	data, ok := m.Files[filename]
	if !ok {
		return nil, os.ErrNotExist
	}
	return data, nil
}

// MkdirAll returns a configured error for the path, or nil.
func (m *MockFileSystem) MkdirAll(path string, _ os.FileMode) error {
	if err, ok := m.MkdirErrors[path]; ok {
		return err
	}
	return nil
}

// CopyFile copies file content from src Files map to dst in the Files map.
// Returns a configured error for src if present. When src is not in Files,
// the copy is a no-op (succeeds without writing dst).
func (m *MockFileSystem) CopyFile(src, dst string) error {
	if err, ok := m.CopyErrors[src]; ok {
		return err
	}
	data, ok := m.Files[src]
	if !ok {
		// No source content configured — succeed silently.
		return nil
	}
	if m.Files == nil {
		m.Files = make(map[string][]byte)
	}
	m.Files[dst] = data
	return nil
}

// RemoveAll returns a configured error for the path, or nil.
func (m *MockFileSystem) RemoveAll(path string) error {
	if err, ok := m.RemoveErrors[path]; ok {
		return err
	}
	return nil
}

// WalkDir walks the configured DirEntries for the root path. For each
// entry, it constructs a path and calls fn. Directories are walked with
// a single visit. Returns a configured error for root if present.
func (m *MockFileSystem) WalkDir(root string, fn fs.WalkDirFunc) error {
	if err, ok := m.WalkErrors[root]; ok {
		return err
	}
	entries, ok := m.DirEntries[root]
	if !ok {
		return nil
	}
	for _, entry := range entries {
		path := filepath.Join(root, entry.Name())
		if err := fn(path, entry, nil); err != nil {
			return err
		}
	}
	return nil
}

// WriteFile stores data in the Files map for the given path. Returns a
// configured error for the path if present.
func (m *MockFileSystem) WriteFile(filename string, data []byte, _ os.FileMode) error {
	if err, ok := m.WriteErrors[filename]; ok {
		return err
	}
	if m.Files == nil {
		m.Files = make(map[string][]byte)
	}
	m.Files[filename] = data
	return nil
}

// --- MockConfigLoader --------------------------------------------------------

// MockConfigLoader implements ConfigLoader with configurable behavior for
// testing. Every method can be controlled via struct fields.
type MockConfigLoader struct {
	// ConfigResult is the config returned by Load(). If nil, a default
	// empty Config is returned.
	ConfigResult *Config

	// LoadErr is the error returned by Load(). Takes priority over
	// ConfigResult.
	LoadErr error
}

// Compile-time check.
var _ ConfigLoader = (*MockConfigLoader)(nil)

// Load returns ConfigResult if LoadErr is nil. Returns a default empty
// Config if ConfigResult is also nil.
func (m *MockConfigLoader) Load() (*Config, error) {
	if m.LoadErr != nil {
		return nil, m.LoadErr
	}
	if m.ConfigResult != nil {
		return m.ConfigResult, nil
	}
	return &Config{}, nil
}

// --- config import bridge ----------------------------------------------------

// configLoad bridges the RealConfigLoader to the config package without an
// import cycle. It is assigned in os_impl_config.go (a separate file that
// imports config.Config).
var configLoad = defaultConfigLoad

// defaultConfigLoad returns an empty Config, used as fallback until the
// real bridge is wired in Phase 3.
func defaultConfigLoad() (*Config, error) {
	return &Config{}, nil
}

// MockConfig is a type alias for Config used by tests for convenience.
// It mirrors the Config struct for test readability.
type MockConfig = Config
