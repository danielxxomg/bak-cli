// Package opencode implements the Adapter interface for the OpenCode
// AI coding tool. It discovers, inventories, backs up, and restores
// configuration files from ~/.config/opencode/.
//
// v1 is the only adapter; the package is structured as a separate
// sub-package so future agents (claude-code, cursor, etc.) can follow
// the same pattern without touching the core adapter interface.
package opencode

import (
	"crypto/sha256"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/danielxxomg/bak-cli/internal/adapters"
	"github.com/danielxxomg/bak-cli/internal/paths"
)

// Adapter name exposed to the registry and CLI.
const adapterName = "opencode"

// configRelPath is the OpenCode config directory relative to the user home.
const configRelPath = ".config/opencode"

// Adapter implements adapters.Adapter for OpenCode.
type Adapter struct{}

// Ensure Adapter satisfies the interface at compile time.
var _ adapters.Adapter = (*Adapter)(nil)

// Name returns the adapter identifier.
func (a *Adapter) Name() string { return adapterName }

// Detect checks whether ~/.config/opencode/ exists on disk.
func (a *Adapter) Detect(homeDir string) (installed bool, configDir string, err error) {
	configDir = filepath.Join(homeDir, configRelPath)
	info, err := os.Stat(configDir)
	if err != nil {
		if os.IsNotExist(err) {
			return false, configDir, nil
		}
		return false, configDir, fmt.Errorf("stat opencode config dir: %w", err)
	}
	if !info.IsDir() {
		return false, configDir, nil
	}
	return true, configDir, nil
}

// --- category → directory mapping -------------------------------------------

// categoryDir maps a category name to the subdirectory/file pattern it
// represents under the OpenCode config root.
type categoryDir struct {
	subPath string // relative path under configDir; empty = root
	isDir   bool   // true when subPath is a directory to scan
}

var categoryMap = map[string]categoryDir{
	"skills":   {subPath: "skills", isDir: true},
	"commands": {subPath: "commands", isDir: true},
	"config":   {subPath: "", isDir: false}, // root-level config files
	"mcp":      {subPath: "", isDir: false}, // root-level mcp.json
	"plugins":  {subPath: "plugins", isDir: true},
	"agents":   {subPath: "agent", isDir: true},
}

// rootConfigFiles lists the file names under the config root that belong
// to the "config" and "mcp" categories.
var rootConfigFiles = map[string]string{
	// config category
	"opencode.jsonc": "config",
	"opencode.json":  "config",
	"config.json":    "config",
	"AGENTS.md":      "config",
	"tui.json":       "config",
	// mcp category
	"mcp.json": "mcp",
}

// ListItems enumerates files and directories belonging to the requested
// categories. It computes SHA-256 hashes and populates Size for every
// regular file. Directories receive a zero hash.
func (a *Adapter) ListItems(homeDir string, categories []string) ([]adapters.Item, error) {
	configDir := filepath.Join(homeDir, configRelPath)

	// Build a set for O(1) lookups.
	catSet := make(map[string]bool, len(categories))
	for _, c := range categories {
		catSet[c] = true
	}

	var items []adapters.Item

	for _, cat := range categories {
		info, ok := categoryMap[cat]
		if !ok {
			continue // unknown category — skip silently
		}

		if info.isDir {
			dir := filepath.Join(configDir, info.subPath)
			dirItems, err := scanDir(dir, cat, configDir, homeDir)
			if err != nil {
				if os.IsNotExist(err) {
					continue // optional directory
				}
				return nil, fmt.Errorf("scan %s: %w", cat, err)
			}
			items = append(items, dirItems...)
		}
	}

	// Root-level files: scan once even if both "config" and "mcp" are requested.
	if catSet["config"] || catSet["mcp"] {
		rootItems, err := scanRootFiles(configDir, homeDir, catSet)
		if err != nil {
			return nil, fmt.Errorf("scan root files: %w", err)
		}
		items = append(items, rootItems...)
	}

	return items, nil
}

// scanDir recursively walks a directory and returns an Item for every
// file and subdirectory found. Directories receive a zero hash and size.
func scanDir(dir, category, configDir, homeDir string) ([]adapters.Item, error) {
	var items []adapters.Item

	err := filepath.WalkDir(dir, func(absPath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// Skip the root directory entry itself.
		if absPath == dir {
			return nil
		}

		relPath, relErr := filepath.Rel(configDir, absPath)
		if relErr != nil {
			return relErr
		}

		canonical := paths.ToCanonical(absPath)

		item := adapters.Item{
			Category:   category,
			SourcePath: canonical,
			RelPath:    filepath.ToSlash(relPath),
			IsDir:      d.IsDir(),
		}

		if !d.IsDir() {
			hash, sz, hashErr := fileHash(absPath)
			if hashErr != nil {
				return fmt.Errorf("hash %s: %w", absPath, hashErr)
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
// whose file name is recognized in rootConfigFiles and whose category is
// in catSet.
func scanRootFiles(configDir, homeDir string, catSet map[string]bool) ([]adapters.Item, error) {
	entries, err := os.ReadDir(configDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var items []adapters.Item
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		cat, recognized := rootConfigFiles[e.Name()]
		if !recognized || !catSet[cat] {
			continue
		}

		absPath := filepath.Join(configDir, e.Name())
		canonical := paths.ToCanonical(absPath)

		info, infoErr := e.Info()
		if infoErr != nil {
			return nil, infoErr
		}

		hash, _, hashErr := fileHash(absPath)
		if hashErr != nil {
			return nil, fmt.Errorf("hash %s: %w", absPath, hashErr)
		}

		items = append(items, adapters.Item{
			Category:   cat,
			SourcePath: canonical,
			RelPath:    e.Name(),
			IsDir:      false,
			Hash:       hash,
			Size:       info.Size(),
		})
	}

	return items, nil
}

// Backup copies items from their source locations into the backup
// directory, preserving the relative structure under an "opencode/"
// adapter prefix.
func (a *Adapter) Backup(homeDir, backupDir string, items []adapters.Item) error {
	configDir := filepath.Join(homeDir, configRelPath)

	for _, item := range items {
		src := filepath.Join(configDir, filepath.FromSlash(item.RelPath))
		dst := filepath.Join(backupDir, adapterName, filepath.FromSlash(item.RelPath))

		if item.IsDir {
			if err := os.MkdirAll(dst, 0755); err != nil {
				return fmt.Errorf("create dir %s: %w", dst, err)
			}
			continue
		}

		if err := copyFile(src, dst); err != nil {
			return fmt.Errorf("copy %s → %s: %w", src, dst, err)
		}
	}

	return nil
}

// Restore copies items from the backup directory back to the user's
// home, placing them under ~/.config/opencode/.
func (a *Adapter) Restore(backupDir, homeDir string, items []adapters.Item) error {
	configDir := filepath.Join(homeDir, configRelPath)

	for _, item := range items {
		src := filepath.Join(backupDir, adapterName, filepath.FromSlash(item.RelPath))
		dst := filepath.Join(configDir, filepath.FromSlash(item.RelPath))

		if item.IsDir {
			if err := os.MkdirAll(dst, 0755); err != nil {
				return fmt.Errorf("create dir %s: %w", dst, err)
			}
			continue
		}

		if err := copyFile(src, dst); err != nil {
			return fmt.Errorf("copy %s → %s: %w", src, dst, err)
		}
	}

	return nil
}

// --- helpers ----------------------------------------------------------------

// copyFile copies a regular file from src to dst, creating parent
// directories as needed. It preserves no metadata beyond file content.
func copyFile(src, dst string) error {
	sf, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open src: %w", err)
	}
	defer sf.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	df, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create dst: %w", err)
	}
	defer df.Close()

	if _, err := io.Copy(df, sf); err != nil {
		return fmt.Errorf("copy: %w", err)
	}

	return df.Close()
}

// fileHash computes the SHA-256 hex digest and file size for the given
// regular file path.
func fileHash(path string) (hash string, size int64, err error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return "", 0, err
	}

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", 0, err
	}

	return fmt.Sprintf("sha256:%x", h.Sum(nil)), info.Size(), nil
}
