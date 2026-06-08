// Package adapters defines the Adapter interface and provides both
// built-in and YAML-configurable adapter implementations for bak-cli.
package adapters

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ConfigAdapter implements the Adapter interface for tools discovered via
// YAML declarations. It supports arbitrary config paths and category
// patterns, making it suitable for any tool with a discoverable config root.
type ConfigAdapter struct {
	def YAMLAdapter
}

// Ensure ConfigAdapter satisfies the Adapter interface at compile time.
var _ Adapter = (*ConfigAdapter)(nil)

// --- Adapter interface -------------------------------------------------------

// Name returns the adapter identifier from the YAML definition.
func (a *ConfigAdapter) Name() string {
	return a.def.Name
}

// Detect checks whether the configuration directory exists under homeDir.
func (a *ConfigAdapter) Detect(homeDir string) (installed bool, configDir string, err error) {
	configDir = filepath.Join(homeDir, filepath.FromSlash(a.def.ConfigPath))

	// Security: validate configDir stays under homeDir.
	cleanCfg := path.Clean(filepath.ToSlash(configDir))
	cleanHome := path.Clean(filepath.ToSlash(homeDir))
	if !strings.HasPrefix(cleanCfg, cleanHome+"/") && cleanCfg != cleanHome {
		return false, configDir, fmt.Errorf("path traversal: config path %q escapes home dir", a.def.ConfigPath)
	}

	info, err := os.Stat(configDir)
	if err != nil {
		if os.IsNotExist(err) {
			return false, configDir, nil
		}
		return false, configDir, fmt.Errorf("stat config dir: %w", err)
	}
	if !info.IsDir() {
		return false, configDir, nil
	}
	return true, configDir, nil
}

// ListItems enumerates files and directories belonging to the requested
// categories, using the YAML-defined patterns to discover items.
func (a *ConfigAdapter) ListItems(homeDir string, categories []string) ([]Item, error) {
	configDir := filepath.Join(homeDir, filepath.FromSlash(a.def.ConfigPath))

	// Security: validate configDir stays under homeDir.
	cleanCfg := path.Clean(filepath.ToSlash(configDir))
	cleanHome := path.Clean(filepath.ToSlash(homeDir))
	if !strings.HasPrefix(cleanCfg, cleanHome+"/") && cleanCfg != cleanHome {
		return nil, fmt.Errorf("path traversal: config path %q escapes home dir", a.def.ConfigPath)
	}

	catSet := make(map[string]bool, len(categories))
	for _, c := range categories {
		catSet[c] = true
	}

	var items []Item
	catMap := a.categoryMap()

	for _, cat := range categories {
		cp, ok := catMap[cat]
		if !ok {
			continue
		}

		if cp.IsDir {
			dir := filepath.Join(configDir, filepath.FromSlash(cp.SubPath))
			dirItems, err := scanCategoryDir(dir, cat, configDir)
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				return nil, fmt.Errorf("scan %s: %w", cat, err)
			}
			items = append(items, dirItems...)
		} else {
			// Non-directory category — use RootFiles or scan root.
			for _, fname := range cp.RootFiles {
				absPath := filepath.Join(configDir, fname)
				info, err := os.Stat(absPath)
				if err != nil {
					if os.IsNotExist(err) {
						continue
					}
					return nil, fmt.Errorf("stat %s: %w", fname, err)
				}
				hash, sz, hashErr := FileHash(absPath)
				if hashErr != nil {
					return nil, fmt.Errorf("hash %s: %w", fname, hashErr)
				}
				items = append(items, Item{
					Category:   cat,
					SourcePath: absPath,
					RelPath:    fname,
					IsDir:      info.IsDir(),
					Hash:       hash,
					Size:       sz,
				})
			}
		}
	}

	return items, nil
}

// Backup copies items from their source locations into the backup
// directory, preserving relative structure under an adapter-named prefix.
func (a *ConfigAdapter) Backup(homeDir, backupDir string, items []Item) error {
	configDir := filepath.Join(homeDir, filepath.FromSlash(a.def.ConfigPath))
	cleanBackup := path.Clean(filepath.ToSlash(backupDir))

	for _, item := range items {
		src := filepath.Join(configDir, filepath.FromSlash(item.RelPath))
		dst := filepath.Join(backupDir, a.def.Name, filepath.FromSlash(item.RelPath))

		// Security: validate dst stays under backupDir.
		cleanDst := path.Clean(filepath.ToSlash(dst))
		if !strings.HasPrefix(cleanDst, cleanBackup+"/") && cleanDst != cleanBackup {
			return fmt.Errorf("path traversal: item %q escapes backup dir", item.RelPath)
		}

		if item.IsDir {
			if err := os.MkdirAll(dst, 0755); err != nil {
				return fmt.Errorf("create dir: %w", err)
			}
			continue
		}

		if err := CopyFile(src, dst); err != nil {
			return fmt.Errorf("copy file: %w", err)
		}
	}

	return nil
}

// Restore copies items from the backup directory back to the user's home.
func (a *ConfigAdapter) Restore(backupDir, homeDir string, items []Item) error {
	configDir := filepath.Join(homeDir, filepath.FromSlash(a.def.ConfigPath))
	cleanHome := path.Clean(filepath.ToSlash(homeDir))

	for _, item := range items {
		src := filepath.Join(backupDir, a.def.Name, filepath.FromSlash(item.RelPath))
		dst := filepath.Join(configDir, filepath.FromSlash(item.RelPath))

		// Security: validate dst stays under homeDir.
		cleanDst := path.Clean(filepath.ToSlash(dst))
		if !strings.HasPrefix(cleanDst, cleanHome+"/") && cleanDst != cleanHome {
			return fmt.Errorf("path traversal: item %q escapes home dir", item.RelPath)
		}

		if item.IsDir {
			if err := os.MkdirAll(dst, 0755); err != nil {
				return fmt.Errorf("create dir: %w", err)
			}
			continue
		}

		if err := CopyFile(src, dst); err != nil {
			return fmt.Errorf("copy file: %w", err)
		}
	}

	return nil
}

// --- YAML loading ------------------------------------------------------------

// LoadYAMLAdapters scans a directory for *.yaml files and parses them as
// adapter definitions. Files with invalid YAML or missing required fields
// are rejected with an error. Returns an empty slice when the directory
// does not exist.
func LoadYAMLAdapters(dir, homeDir string) ([]*ConfigAdapter, error) {
	// Security: validate path stays under user home BEFORE any filesystem access.
	cleanDir := path.Clean(filepath.ToSlash(dir))
	cleanHome := path.Clean(filepath.ToSlash(homeDir))
	if !strings.HasPrefix(cleanDir, cleanHome+"/") && cleanDir != cleanHome {
		return nil, fmt.Errorf("path traversal: directory outside home")
	}

	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read adapters dir: %w", err)
	}

	var adapters []*ConfigAdapter
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := strings.ToLower(entry.Name())
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			continue
		}

		filePath := filepath.Join(dir, entry.Name())
		ca, err := loadAdapterFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("load adapter %q: %w", entry.Name(), err)
		}
		adapters = append(adapters, ca)
	}

	return adapters, nil
}

// loadAdapterFile reads and validates a single YAML adapter file.
func loadAdapterFile(filePath string) (*ConfigAdapter, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	var def YAMLAdapter
	if err := yaml.Unmarshal(data, &def); err != nil {
		return nil, fmt.Errorf("yaml parse: %w", err)
	}

	if def.Name == "" {
		return nil, fmt.Errorf("missing required field \"name\"")
	}
	if def.ConfigPath == "" {
		return nil, fmt.Errorf("missing required field \"config_path\"")
	}

	return &ConfigAdapter{def: def}, nil
}

// --- helpers ----------------------------------------------------------------

// categoryMap builds a lookup from category name to pattern definition.
func (a *ConfigAdapter) categoryMap() map[string]YAMLCategoryPattern {
	m := make(map[string]YAMLCategoryPattern, len(a.def.Categories))
	for _, cp := range a.def.Categories {
		m[cp.Name] = cp
	}
	return m
}

// scanCategoryDir recursively walks a directory and returns an Item for
// every file and subdirectory found.
func scanCategoryDir(dir, category, configDir string) ([]Item, error) {
	var items []Item

	err := filepath.WalkDir(dir, func(absPath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if absPath == dir {
			return nil
		}

		relPath, relErr := filepath.Rel(configDir, absPath)
		if relErr != nil {
			return relErr
		}

		item := Item{
			Category:   category,
			SourcePath: absPath,
			RelPath:    filepath.ToSlash(relPath),
			IsDir:      d.IsDir(),
		}

		if !d.IsDir() {
			hash, sz, hashErr := FileHash(absPath)
			if hashErr != nil {
				return fmt.Errorf("hash file: %w", hashErr)
			}
			item.Hash = hash
			item.Size = sz
		}

		items = append(items, item)
		return nil
	})

	return items, err
}
