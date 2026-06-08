// Package claudecode implements the Adapter interface for Claude Code
// (Anthropic's CLI coding agent). It discovers, inventories, backs up, and
// restores configuration files from ~/.claude/.
package claudecode

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/danielxxomg/bak-cli/internal/adapters"
	"github.com/danielxxomg/bak-cli/internal/paths"
)

// adapterName is the identifier exposed to the registry and CLI.
const adapterName = "claude-code"

// configRelPath is the Claude Code config directory relative to the user home.
const configRelPath = ".claude"

// Adapter implements adapters.Adapter for Claude Code.
type Adapter struct{}

// Ensure Adapter satisfies the interface at compile time.
var _ adapters.Adapter = (*Adapter)(nil)

// Name returns the adapter identifier.
func (a *Adapter) Name() string { return adapterName }

// Detect checks whether ~/.claude/ exists on disk.
func (a *Adapter) Detect(homeDir string) (installed bool, configDir string, err error) {
	configDir = filepath.Join(homeDir, configRelPath)
	info, err := os.Stat(configDir)
	if err != nil {
		if os.IsNotExist(err) {
			return false, configDir, nil
		}
		return false, configDir, fmt.Errorf("stat claude-code config dir: %w", err)
	}
	if !info.IsDir() {
		return false, configDir, nil
	}
	return true, configDir, nil
}

// categoryDir maps a category name to the subdirectory/file pattern it
// represents under the Claude Code config root.
type categoryDir struct {
	subPath string
	isDir   bool
}

var categoryMap = map[string]categoryDir{
	"config":   {subPath: "", isDir: false}, // root-level config files
	"skills":   {subPath: "skills", isDir: true},
	"commands": {subPath: "commands", isDir: true},
}

// ListItems enumerates files and directories belonging to the requested
// categories. It computes SHA-256 hashes and populates Size for every
// regular file. Directories receive a zero hash.
func (a *Adapter) ListItems(homeDir string, categories []string) ([]adapters.Item, error) {
	configDir := filepath.Join(homeDir, configRelPath)

	catSet := make(map[string]bool, len(categories))
	for _, c := range categories {
		catSet[c] = true
	}

	var items []adapters.Item

	for _, cat := range categories {
		info, ok := categoryMap[cat]
		if !ok {
			continue
		}

		if info.isDir {
			dir := filepath.Join(configDir, info.subPath)
			dirItems, err := scanDir(dir, cat, configDir, homeDir)
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				return nil, fmt.Errorf("scan %s: %w", cat, err)
			}
			items = append(items, dirItems...)
		}
	}

	// Root-level config files.
	if catSet["config"] {
		rootItems, err := scanRootFiles(configDir, homeDir, catSet)
		if err != nil {
			return nil, fmt.Errorf("scan root files: %w", err)
		}
		items = append(items, rootItems...)
	}

	return items, nil
}

// scanDir recursively walks a directory and returns an Item for every
// file and subdirectory found.
func scanDir(dir, category, configDir, homeDir string) ([]adapters.Item, error) {
	var items []adapters.Item

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

		canonical := paths.ToCanonical(absPath)

		item := adapters.Item{
			Category:   category,
			SourcePath: canonical,
			RelPath:    filepath.ToSlash(relPath),
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
// for all regular files that belong to categories in catSet.
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
		if !catSet["config"] {
			continue
		}

		absPath := filepath.Join(configDir, e.Name())
		canonical := paths.ToCanonical(absPath)

		info, infoErr := e.Info()
		if infoErr != nil {
			return nil, infoErr
		}

		hash, _, hashErr := adapters.FileHash(absPath)
		if hashErr != nil {
			return nil, fmt.Errorf("hash %s: %w", absPath, hashErr)
		}

		items = append(items, adapters.Item{
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

// Backup copies items from their source locations into the backup
// directory, preserving the relative structure under the adapter prefix.
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
// home, placing them under ~/.claude/.
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
