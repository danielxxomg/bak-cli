// Package adapters provides the GenericAdapter base struct for AI coding
// tool adapters that follow the standard scan-dir + scan-root-files pattern.
package adapters

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/danielxxomg/bak-cli/internal/paths"
)

// CategoryDir maps a category name to the subdirectory pattern it represents
// under the adapter's config root.
type CategoryDir struct {
	SubPath string // relative path under configDir; empty = root
	IsDir   bool   // true when SubPath is a directory to scan
}

// GenericAdapter implements Adapter for tools that follow the standard
// scan-dir + scan-root-files pattern. Each adapter package constructs
// a GenericAdapter with its own constants and delegates interface methods.
type GenericAdapter struct {
	AdapterName      string
	ConfigRelPath    string
	Categories       map[string]CategoryDir
	DetectErrContext string // e.g. "stat codex config dir"

	// StatFn replaces os.Stat in Detect. When nil, Detect falls back
	// to os.Stat. Inject a custom function to simulate stat failures
	// in tests without relying on OS-level permissions (chmod).
	StatFn func(string) (os.FileInfo, error)
}

// Compile-time check: GenericAdapter satisfies the Adapter interface.
var _ Adapter = (*GenericAdapter)(nil)

// Name returns the adapter identifier.
func (ga *GenericAdapter) Name() string { return ga.AdapterName }

// Detect checks whether the adapter's config directory exists under homeDir.
func (ga *GenericAdapter) Detect(homeDir string) (installed bool, configDir string, err error) {
	configDir = filepath.Join(homeDir, ga.ConfigRelPath)
	if !pathUnderHome(configDir, homeDir) {
		return false, configDir, fmt.Errorf("config path escapes home: %s", ga.ConfigRelPath)
	}

	statFn := ga.StatFn
	if statFn == nil {
		statFn = os.Stat
	}
	info, err := statFn(configDir)
	if err != nil {
		if os.IsNotExist(err) {
			return false, configDir, nil
		}
		return false, configDir, fmt.Errorf("%s: %w", ga.DetectErrContext, err)
	}
	if !info.IsDir() {
		return false, configDir, nil
	}
	return true, configDir, nil
}

// ListItems enumerates files and directories belonging to the requested
// categories. It computes SHA-256 hashes for regular files.
func (ga *GenericAdapter) ListItems(homeDir string, categories []string) ([]Item, error) {
	configDir := filepath.Join(homeDir, ga.ConfigRelPath)
	if !pathUnderHome(configDir, homeDir) {
		return nil, fmt.Errorf("config path escapes home: %s", ga.ConfigRelPath)
	}

	catSet := make(map[string]bool, len(categories))
	for _, c := range categories {
		catSet[c] = true
	}

	var items []Item

	for _, cat := range categories {
		info, ok := ga.Categories[cat]
		if !ok {
			continue
		}

		if info.IsDir {
			dir := filepath.Join(configDir, info.SubPath)
			dirItems, err := scanDir(dir, cat, configDir)
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				return nil, fmt.Errorf("scan %s: %w", cat, err)
			}
			items = append(items, dirItems...)
		}
	}

	if catSet["config"] {
		rootItems, err := scanRootFiles(configDir, catSet)
		if err != nil {
			return nil, fmt.Errorf("scan root files: %w", err)
		}
		items = append(items, rootItems...)
	}

	return items, nil
}

// Backup copies items from their source locations into the backup directory,
// preserving the relative structure under an "<AdapterName>/" prefix.
func (ga *GenericAdapter) Backup(homeDir, backupDir string, items []Item) error {
	configDir := filepath.Join(homeDir, ga.ConfigRelPath)
	if !pathUnderHome(configDir, homeDir) {
		return fmt.Errorf("config path escapes home: %s", ga.ConfigRelPath)
	}
	dstBase := filepath.Join(backupDir, ga.AdapterName)
	return copyItems(items, configDir, dstBase)
}

// Restore copies items from the backup directory back to the user's home,
// placing them under the adapter's config directory.
func (ga *GenericAdapter) Restore(backupDir, homeDir string, items []Item) error {
	configDir := filepath.Join(homeDir, ga.ConfigRelPath)
	if !pathUnderHome(configDir, homeDir) {
		return fmt.Errorf("config path escapes home: %s", ga.ConfigRelPath)
	}
	srcBase := filepath.Join(backupDir, ga.AdapterName)
	return copyItems(items, srcBase, configDir)
}

// copyItems is the shared implementation for Backup and Restore. It iterates
// over items, creates directories via os.MkdirAll, and copies files via
// CopyFile. Error messages use item.RelPath to avoid leaking absolute paths.
func copyItems(items []Item, srcBase, dstBase string) error {
	for _, item := range items {
		rel := filepath.FromSlash(item.RelPath)
		src := filepath.Join(srcBase, rel)
		dst := filepath.Join(dstBase, rel)

		if item.IsDir {
			if err := os.MkdirAll(dst, 0755); err != nil {
				return fmt.Errorf("create dir %s: %w", item.RelPath, err)
			}
			continue
		}

		if err := CopyFile(src, dst); err != nil {
			return fmt.Errorf("copy %s: %w", item.RelPath, err)
		}
	}
	return nil
}

// scanDir recursively walks a directory and returns an Item for every
// file and subdirectory found. Directories receive a zero hash and size.
func scanDir(dir, category, configDir string) ([]Item, error) {
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
			return fmt.Errorf("compute relative path: %w", relErr)
		}

		canonical := paths.ToCanonical(absPath)

		item := Item{
			Category:   category,
			SourcePath: canonical,
			RelPath:    strings.ReplaceAll(relPath, "\\", "/"),
			IsDir:      d.IsDir(),
		}

		if !d.IsDir() {
			hash, sz, hashErr := FileHash(absPath)
			if hashErr != nil {
				return fmt.Errorf("hash %s: %w", strings.ReplaceAll(relPath, "\\", "/"), hashErr)
			}
			item.Hash = hash
			item.Size = sz
		}

		items = append(items, item)
		return nil
	})

	return items, err
}

// scanRootFiles reads the top-level config directory and returns items
// for all regular files that belong to categories in catSet.
func scanRootFiles(configDir string, catSet map[string]bool) ([]Item, error) {
	entries, err := os.ReadDir(configDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read config dir: %w", err)
	}

	var items []Item
	for _, e := range entries {
		if e.IsDir() || !catSet["config"] {
			continue
		}

		absPath := filepath.Join(configDir, e.Name())
		canonical := paths.ToCanonical(absPath)

		info, infoErr := e.Info()
		if infoErr != nil {
			return nil, fmt.Errorf("stat %s: %w", e.Name(), infoErr)
		}

		hash, _, hashErr := FileHash(absPath)
		if hashErr != nil {
			return nil, fmt.Errorf("hash %s: %w", e.Name(), hashErr)
		}

		items = append(items, Item{
			Category:   "config",
			SourcePath: canonical,
			RelPath:    e.Name(),
			IsDir:      false,
			Hash:       hash,
			Size:       info.Size(),
		})
	}

	return items, nil
}

// pathUnderHome returns true when resolved does not escape homeDir.
// It uses path.Clean with forward-slash normalization per AGENTS.md.
func pathUnderHome(resolved, homeDir string) bool {
	c := path.Clean(strings.ReplaceAll(resolved, "\\", "/"))
	h := path.Clean(strings.ReplaceAll(homeDir, "\\", "/"))
	if c == h {
		return true
	}
	return strings.HasPrefix(c, h+"/")
}
