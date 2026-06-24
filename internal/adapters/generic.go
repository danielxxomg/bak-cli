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

	// ScanOpts holds optional filtering for ListItems. The zero value
	// preserves current behavior (all files included).
	ScanOpts ScanOptions

	// RootConfigFiles is an optional whitelist mapping root entry names
	// to their category. When non-nil, scanRootFiles skips any root
	// entry whose name is not in this map. This is the belt alongside
	// DefaultExcludes as suspenders: whitelist prevents future runtime
	// files from leaking; excludes catch them even without a whitelist.
	RootConfigFiles map[string]string

	// StatFn replaces os.Stat in Detect. When nil, Detect falls back
	// to os.Stat. Inject a custom function to simulate stat failures
	// in tests without relying on OS-level permissions (chmod).
	StatFn func(string) (os.FileInfo, error)
}

// Compile-time check: GenericAdapter satisfies the Adapter interface.
var _ Adapter = (*GenericAdapter)(nil)

// Compile-time check: GenericAdapter satisfies ScanConfigurable.
var _ ScanConfigurable = (*GenericAdapter)(nil)

// Name returns the adapter identifier.
func (ga *GenericAdapter) Name() string { return ga.AdapterName }

// SetScanOptions applies the given scanning options to the adapter.
func (ga *GenericAdapter) SetScanOptions(opts ScanOptions) {
	ga.ScanOpts = opts
}

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
			dirItems, err := scanDir(dir, cat, configDir, ga.ScanOpts)
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				return nil, fmt.Errorf("scan %s: %w", cat, err)
			}
			items = append(items, dirItems...)
		}
	}

	if rootScanRequested(catSet, ga.RootConfigFiles) {
		rootItems, err := scanRootFiles(configDir, catSet, ga.ScanOpts, ga.RootConfigFiles)
		if err != nil {
			return nil, fmt.Errorf("scan root files: %w", err)
		}
		items = append(items, rootItems...)
	}

	return items, nil
}

// rootScanRequested reports whether the root-file scan should run for the
// requested category set. When rootConfigFiles is nil, the legacy behavior
// applies: root files belong to "config" only. When non-nil, the scan runs
// if any mapped category is in catSet (so an mcp-only request still scans
// the root for mcp.json without pulling in config files).
func rootScanRequested(catSet map[string]bool, rootConfigFiles map[string]string) bool {
	if rootConfigFiles == nil {
		return catSet["config"]
	}
	for _, cat := range rootConfigFiles {
		if catSet[cat] {
			return true
		}
	}
	return false
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
// When opts is non-zero, entries matching exclude patterns or exceeding
// MaxFileSize are skipped.
func scanDir(dir, category, configDir string, opts ScanOptions) ([]Item, error) {
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

		// Normalize for matching.
		rel := strings.ReplaceAll(relPath, "\\", "/")

		// Check exclude patterns.
		if matchesExclude(d.Name(), rel, d.IsDir(), opts) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Check MaxFileSize for regular files.
		if !d.IsDir() && opts.MaxFileSize > 0 {
			info, statErr := d.Info()
			if statErr != nil {
				return fmt.Errorf("stat %s: %w", relPath, statErr)
			}
			if info.Size() > opts.MaxFileSize {
				emitOversizeWarning(info.Size(), opts.MaxFileSize, rel)
				return nil
			}
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

// MatchExclude checks whether a file or directory matches an exclude pattern.
// It supports basic gitignore-style matching:
//   - "dir/" matches directories named "dir" anywhere in the path
//   - "*.ext" matches files ending in ".ext" anywhere in the path
//   - "name" matches files or directories named "name" anywhere in the path
func MatchExclude(pattern, name, relPath string, isDir bool) bool {
	// Parse the pattern.
	dirOnly := false
	raw := pattern
	if len(raw) > 0 && raw[len(raw)-1] == '/' {
		dirOnly = true
		raw = raw[:len(raw)-1]
	}
	negate := false
	if len(raw) > 0 && raw[0] == '!' {
		negate = true
		raw = raw[1:]
	}

	// Directory-only patterns only match directories.
	if dirOnly && !isDir {
		return false
	}

	// Match the entry name against the pattern.
	var matched bool
	if strings.Contains(raw, "*") {
		// A malformed glob (filepath.ErrBadPattern) cannot match anything;
		// treat it as a non-match rather than discarding the error blindly.
		m, err := filepath.Match(raw, name)
		matched = err == nil && m
	} else {
		matched = (name == raw)
	}

	// Negation: re-include.
	if negate && matched {
		return false
	}

	return matched
}

// matchesExclude reports whether an entry matches any pattern in opts.Excludes.
// It is the shared exclude gate used by both scanDir and scanRootFiles so the
// matching loop lives in one place.
func matchesExclude(name, relPath string, isDir bool, opts ScanOptions) bool {
	for _, pat := range opts.Excludes {
		if MatchExclude(pat, name, relPath, isDir) {
			return true
		}
	}
	return false
}

// emitOversizeWarning writes the shared "skipping large file" warning to
// stderr. Both scanDir and scanRootFiles use it so the message format stays
// identical in one place.
func emitOversizeWarning(size, max int64, rel string) {
	fmt.Fprintf(os.Stderr, "warning: skipping large file (%d bytes exceeds max %d): %s\n",
		size, max, rel)
}

// scanRootFiles reads the top-level config directory and returns items
// for all regular files that belong to categories in catSet. When opts
// is non-zero, entries matching exclude patterns or exceeding MaxFileSize
// are skipped — mirroring scanDir's filtering. When rootConfigFiles is
// non-nil, only file names present in the map are included (whitelist).
func scanRootFiles(configDir string, catSet map[string]bool, opts ScanOptions, rootConfigFiles map[string]string) ([]Item, error) {
	entries, err := os.ReadDir(configDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read config dir: %w", err)
	}

	var items []Item
	for _, e := range entries {
		if e.IsDir() {
			continue
		}

		entryName := e.Name()

		// Per-entry category resolution. When rootConfigFiles is nil the
		// legacy behavior applies: every root file belongs to "config".
		// When non-nil it acts as a whitelist (skip unrecognized names)
		// and maps each name to its real category.
		cat := "config"
		if rootConfigFiles != nil {
			mapped, ok := rootConfigFiles[entryName]
			if !ok {
				continue
			}
			cat = mapped
		}
		if !catSet[cat] {
			continue
		}

		absPath := filepath.Join(configDir, entryName)

		// Check exclude patterns.
		if matchesExclude(entryName, entryName, false, opts) {
			continue
		}

		info, infoErr := e.Info()
		if infoErr != nil {
			return nil, fmt.Errorf("stat %s: %w", entryName, infoErr)
		}

		// Check MaxFileSize for regular files.
		if opts.MaxFileSize > 0 && info.Size() > opts.MaxFileSize {
			emitOversizeWarning(info.Size(), opts.MaxFileSize, entryName)
			continue
		}

		canonical := paths.ToCanonical(absPath)

		hash, _, hashErr := FileHash(absPath)
		if hashErr != nil {
			return nil, fmt.Errorf("hash %s: %w", entryName, hashErr)
		}

		items = append(items, Item{
			Category:   cat,
			SourcePath: canonical,
			RelPath:    entryName,
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
