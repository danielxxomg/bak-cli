// Package kilocode implements the Adapter interface for KiloCode
// (an AI coding agent). It discovers, inventories, backs up, and
// restores configuration files from ~/.kilocode/.
package kilocode

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/danielxxomg/bak-cli/internal/adapters"
	"github.com/danielxxomg/bak-cli/internal/paths"
)

const adapterName = "kilocode"
const configRelPath = ".kilocode"

type Adapter struct{}

var _ adapters.Adapter = (*Adapter)(nil)

func (a *Adapter) Name() string { return adapterName }

func (a *Adapter) Detect(homeDir string) (installed bool, configDir string, err error) {
	configDir = filepath.Join(homeDir, configRelPath)
	info, err := os.Stat(configDir)
	if err != nil {
		if os.IsNotExist(err) {
			return false, configDir, nil
		}
		return false, configDir, fmt.Errorf("stat kilocode config dir: %w", err)
	}
	if !info.IsDir() {
		return false, configDir, nil
	}
	return true, configDir, nil
}

type categoryDir struct {
	subPath string
	isDir   bool
}

var categoryMap = map[string]categoryDir{
	"config": {subPath: "", isDir: false},
	"rules":  {subPath: "rules", isDir: true},
}

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
	if catSet["config"] {
		rootItems, err := scanRootFiles(configDir, homeDir, catSet)
		if err != nil {
			return nil, fmt.Errorf("scan root files: %w", err)
		}
		items = append(items, rootItems...)
	}
	return items, nil
}

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
		if e.IsDir() || !catSet["config"] {
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
