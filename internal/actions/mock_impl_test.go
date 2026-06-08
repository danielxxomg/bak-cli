package actions

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/danielxxomg/bak-cli/internal/cloud"
)

// MockFileSystem implements FileSystem with configurable behavior for
// testing. Every method can be controlled via the struct fields.
type MockFileSystem struct {
	HomeDir        string
	StatResult     map[string]MockStatResult
	DirEntries     map[string][]os.DirEntry
	ReadDirErrors  map[string]error
	Files          map[string][]byte
	ReadFileErrors map[string]error
	MkdirErrors    map[string]error
	CopyErrors     map[string]error
	RemoveErrors   map[string]error
	WalkErrors     map[string]error
	WriteErrors    map[string]error
}

var _ FileSystem = (*MockFileSystem)(nil)

// MockStatResult pairs a file info with an optional error for Stat().
type MockStatResult struct {
	Info os.FileInfo
	Err  error
}

func (m *MockFileSystem) UserHomeDir() (string, error) {
	return m.HomeDir, nil
}

func (m *MockFileSystem) Stat(path string) (os.FileInfo, error) {
	if r, ok := m.StatResult[path]; ok {
		return r.Info, r.Err
	}
	return nil, os.ErrNotExist
}

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

func (m *MockFileSystem) MkdirAll(path string, _ os.FileMode) error {
	if err, ok := m.MkdirErrors[path]; ok {
		return err
	}
	return nil
}

func (m *MockFileSystem) CopyFile(src, dst string) error {
	if err, ok := m.CopyErrors[src]; ok {
		return err
	}
	data, ok := m.Files[src]
	if !ok {
		return nil
	}
	if m.Files == nil {
		m.Files = make(map[string][]byte)
	}
	m.Files[dst] = data
	return nil
}

func (m *MockFileSystem) RemoveAll(path string) error {
	if err, ok := m.RemoveErrors[path]; ok {
		return err
	}
	return nil
}

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

// MockConfigLoader implements ConfigLoader with configurable behavior.
type MockConfigLoader struct {
	ConfigResult *Config
	LoadErr      error
}

var _ ConfigLoader = (*MockConfigLoader)(nil)

func (m *MockConfigLoader) Load() (*Config, error) {
	if m.LoadErr != nil {
		return nil, m.LoadErr
	}
	if m.ConfigResult != nil {
		return m.ConfigResult, nil
	}
	return &Config{}, nil
}

// MockProviderFactory implements ProviderFactory with configurable behavior.
type MockProviderFactory struct {
	Providers map[string]cloud.Provider
	Err       error
}

var _ ProviderFactory = (*MockProviderFactory)(nil)

func (m *MockProviderFactory) CreateProvider(name string) (cloud.Provider, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	p, ok := m.Providers[name]
	if !ok {
		return nil, fmt.Errorf("unknown provider: %q", name)
	}
	return p, nil
}

// MockProvider implements cloud.Provider with configurable function fields.
type MockProvider struct {
	MockName string
	PushFn   func(archive []byte, meta cloud.PushMeta) (string, error)
	PullFn   func(id string) ([]byte, error)
	ListFn   func() ([]cloud.BackupMeta, error)
}

var _ cloud.Provider = (*MockProvider)(nil)

func (m *MockProvider) Name() string { return m.MockName }
func (m *MockProvider) Push(archive []byte, meta cloud.PushMeta) (string, error) {
	return m.PushFn(archive, meta)
}
func (m *MockProvider) Pull(id string) ([]byte, error)    { return m.PullFn(id) }
func (m *MockProvider) List() ([]cloud.BackupMeta, error) { return m.ListFn() }
