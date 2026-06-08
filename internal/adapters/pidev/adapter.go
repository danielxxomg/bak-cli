// Package pidev implements the Adapter interface for pi.dev (the AI
// coding platform). It discovers, inventories, backs up, and restores
// configuration files from ~/.pi/.
package pidev

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

const adapterName = "pidev"
const configRelPath = ".pi"

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
		return false, configDir, fmt.Errorf("stat pidev config dir: %w", err)
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
	"agents": {subPath: "agents", isDir: true},
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
		hash, _, hashErr := fileHash(absPath)
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
		if err := copyFile(src, dst); err != nil {
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
		if err := copyFile(src, dst); err != nil {
			return fmt.Errorf("copy %s → %s: %w", src, dst, err)
		}
	}
	return nil
}

func copyFile(src, dst string) error {
	sf, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open src: %w", err)
	}
	defer func() { _ = sf.Close() }()
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	df, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create dst: %w", err)
	}
	defer func() { _ = df.Close() }()
	if _, err := io.Copy(df, sf); err != nil {
		return fmt.Errorf("copy: %w", err)
	}
	return df.Close()
}

func fileHash(path string) (hash string, size int64, err error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer func() { _ = f.Close() }()
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
