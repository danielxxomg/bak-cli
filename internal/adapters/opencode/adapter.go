// Package opencode implements the Adapter interface for the OpenCode
// AI coding tool. It discovers, inventories, backs up, and restores
// configuration files from ~/.config/opencode/.
//
// v1 is the only adapter; the package is structured as a separate
// sub-package so future agents (claude-code, cursor, etc.) can follow
// the same pattern without touching the core adapter interface.
package opencode

import (
	"fmt"
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
type Adapter struct {
	// ScanOpts holds optional filtering for ListItems. The zero value
	// preserves current behavior (all files included).
	ScanOpts adapters.ScanOptions
}

// Compile-time check: Adapter satisfies the interface at compile time.
var _ adapters.Adapter = (*Adapter)(nil)

// Compile-time check: Adapter satisfies ScanConfigurable.
var _ adapters.ScanConfigurable = (*Adapter)(nil)

// Name returns the adapter identifier.
func (a *Adapter) Name() string { return adapterName }

// SetScanOptions applies the given scanning options to the adapter.
func (a *Adapter) SetScanOptions(opts adapters.ScanOptions) {
	a.ScanOpts = opts
}

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
			dirItems, err := scanDir(dir, cat, configDir, homeDir, a.ScanOpts)
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
		rootItems, err := scanRootFiles(configDir, homeDir, catSet, a.ScanOpts)
		if err != nil {
			return nil, fmt.Errorf("scan root files: %w", err)
		}
		items = append(items, rootItems...)
	}

	return items, nil
}

// scanDir recursively walks a directory and returns an Item for every
// file and subdirectory found. Directories receive a zero hash and size.
// When opts is non-zero, entries matching exclude patterns or exceeding
// MaxFileSize are skipped.
func scanDir(dir, category, configDir, homeDir string, opts adapters.ScanOptions) ([]adapters.Item, error) {
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

		// Normalize for matching.
		rel := paths.Slash(relPath)

		// Check exclude patterns.
		if len(opts.Excludes) > 0 {
			entryName := d.Name()
			for _, pat := range opts.Excludes {
				if adapters.MatchExclude(pat, entryName, rel, d.IsDir()) {
					if d.IsDir() {
						return filepath.SkipDir
					}
					return nil
				}
			}
		}

		// Check MaxFileSize for regular files.
		if !d.IsDir() && opts.MaxFileSize > 0 {
			info, statErr := d.Info()
			if statErr != nil {
				return fmt.Errorf("stat %s: %w", relPath, statErr)
			}
			if info.Size() > opts.MaxFileSize {
				warning := fmt.Sprintf("warning: skipping large file (%d bytes exceeds max %d): %s\n",
					info.Size(), opts.MaxFileSize, rel)
				// Write to stderr via os.Stderr for CLI visibility.
				if _, werr := os.Stderr.WriteString(warning); werr != nil {
					return fmt.Errorf("write stderr warning: %w", werr)
				}
				return nil
			}
		}

		canonical := paths.ToCanonical(absPath)

		item := adapters.Item{
			Category:   category,
			SourcePath: canonical,
			RelPath:    paths.Slash(relPath),
			IsDir:      d.IsDir(),
		}

		if !d.IsDir() {
			hash, sz, hashErr := adapters.FileHash(absPath)
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
// in catSet. When opts is non-zero, entries matching exclude patterns or
// exceeding MaxFileSize are skipped — mirroring scanDir's filtering.
func scanRootFiles(configDir, homeDir string, catSet map[string]bool, opts adapters.ScanOptions) ([]adapters.Item, error) {
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

		entryName := e.Name()
		absPath := filepath.Join(configDir, entryName)

		// Check exclude patterns (mirror scanDir).
		if len(opts.Excludes) > 0 {
			skipped := false
			for _, pat := range opts.Excludes {
				if adapters.MatchExclude(pat, entryName, entryName, false) {
					skipped = true
					break
				}
			}
			if skipped {
				continue
			}
		}

		canonical := paths.ToCanonical(absPath)

		info, infoErr := e.Info()
		if infoErr != nil {
			return nil, infoErr
		}

		item := adapters.Item{
			Category:   cat,
			SourcePath: canonical,
			RelPath:    e.Name(),
			IsDir:      false,
			Hash:       "",
			Size:       info.Size(),
		}

		hash, _, hashErr := adapters.FileHash(absPath)
		if hashErr != nil {
			return nil, fmt.Errorf("hash %s: %w", absPath, hashErr)
		}
		item.Hash = hash

		items = append(items, item)
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

		if err := adapters.CopyFile(src, dst); err != nil {
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

		if err := adapters.CopyFile(src, dst); err != nil {
			return fmt.Errorf("copy %s → %s: %w", src, dst, err)
		}
	}

	return nil
}
