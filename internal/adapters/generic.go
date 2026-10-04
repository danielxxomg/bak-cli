// Package adapters provides the GenericAdapter base struct for AI coding
// tool adapters that follow the standard scan-dir + scan-root-files pattern.
package adapters

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/danielxxomg/bak-cli/internal/paths"
)

// osName is the runtime OS identifier, injected for testability and
// centralized so platform-specific behavior has a single reference.
var osName = runtime.GOOS

// isWindows reports whether the current platform is Windows.
func isWindows() bool { return osName == "windows" }

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

	// Verbose gates diagnostic warnings during scanning (e.g. skipped broken symlinks).
	Verbose bool

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

// SetVerbose sets the verbose flag on GenericAdapter.
func (ga *GenericAdapter) SetVerbose(v bool) {
	ga.Verbose = v
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
			dirItems, err := scanDir(dir, cat, configDir, homeDir, ga.ScanOpts, ga.Verbose)
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
		rootItems, err := scanRootFiles(configDir, homeDir, catSet, ga.ScanOpts, ga.RootConfigFiles, ga.Verbose)
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
			perm := os.FileMode(0755)
			if item.Mode != 0 {
				perm = os.FileMode(item.Mode)
			}
			if err := os.MkdirAll(dst, 0755); err != nil {
				return fmt.Errorf("create dir %s: %w", item.RelPath, err)
			}
			if item.Mode != 0 {
				if err := os.Chmod(dst, perm); err != nil && !isWindows() {
					return fmt.Errorf("chmod dir %s: %w", item.RelPath, err)
				}
			}
			continue
		}

		if err := CopyFile(src, dst); err != nil {
			return fmt.Errorf("copy %s: %w", item.RelPath, err)
		}
		if item.Mode != 0 {
			if err := os.Chmod(dst, os.FileMode(item.Mode)); err != nil && !isWindows() {
				return fmt.Errorf("chmod %s: %w", item.RelPath, err)
			}
		}
	}
	return nil
}

// initAncestorDirs records all real directory paths from dir up to homeDir
// to prevent symlinks from looping back to parent directories.
func initAncestorDirs(dir, homeDir string) map[string]bool {
	visited := make(map[string]bool)
	for p := dir; ; {
		if realP, err := filepath.EvalSymlinks(p); err == nil {
			visited[paths.CanonicalPath(realP)] = true
		}
		if p == homeDir || p == filepath.Dir(p) {
			break
		}
		p = filepath.Dir(p)
	}
	return visited
}

// resolveSymlink resolves a symlink entry with os.Stat and filepath.EvalSymlinks,
// verifying that its target stays within homeDir. Returns ok=false on any error
// or containment violation (with diagnostic warning if verbose is enabled).
func resolveSymlink(absPath, rel, homeDir string, verbose bool) (fs.FileInfo, string, bool) {
	targetStat, statErr := os.Stat(absPath)
	if statErr != nil {
		if verbose {
			if warnErr := emitSymlinkWarning(rel, statErr); warnErr != nil {
				fmt.Fprintf(os.Stderr, "warning: %v\n", warnErr)
			}
		}
		return nil, "", false
	}

	targetReal, evalErr := filepath.EvalSymlinks(absPath)
	if evalErr != nil {
		if verbose {
			if warnErr := emitSymlinkWarning(rel, evalErr); warnErr != nil {
				fmt.Fprintf(os.Stderr, "warning: %v\n", warnErr)
			}
		}
		return nil, "", false
	}

	if !pathUnderHome(targetReal, homeDir) {
		if verbose {
			if warnErr := emitSymlinkWarning(rel, fmt.Errorf("target escapes home: %s", targetReal)); warnErr != nil {
				fmt.Fprintf(os.Stderr, "warning: %v\n", warnErr)
			}
		}
		return nil, "", false
	}

	return targetStat, targetReal, true
}

// isOversized checks whether size exceeds maxFileSize, emitting a warning if so.
func isOversized(size, max int64, rel string) bool {
	if max > 0 && size > max {
		if warnErr := emitOversizeWarning(size, max, rel); warnErr != nil {
			fmt.Fprintf(os.Stderr, "warning: %v\n", warnErr)
		}
		return true
	}
	return false
}

// createFileItem constructs a non-directory Item with computed hash and size.
func createFileItem(absPath, rel, category string, info fs.FileInfo) (Item, error) {
	hash, sz, hashErr := FileHash(absPath)
	if hashErr != nil {
		return Item{}, fmt.Errorf("hash %s: %w", rel, hashErr)
	}
	return Item{
		Category:   category,
		SourcePath: paths.ToCanonical(absPath),
		RelPath:    rel,
		IsDir:      false,
		Hash:       hash,
		Size:       sz,
		Mode:       uint32(info.Mode().Perm()),
	}, nil
}

// createDirItem constructs a directory Item with mode metadata.
func createDirItem(absPath, rel, category string, info fs.FileInfo) Item {
	return Item{
		Category:   category,
		SourcePath: paths.ToCanonical(absPath),
		RelPath:    rel,
		IsDir:      true,
		Mode:       uint32(info.Mode().Perm()),
	}
}

// dirScanner encapsulates directory traversal state for scanDir.
type dirScanner struct {
	category    string
	configDir   string
	homeDir     string
	opts        ScanOptions
	verbose     bool
	visitedDirs map[string]bool
	items       []Item
}

func (s *dirScanner) walk(currentDir string) error {
	entries, err := os.ReadDir(currentDir)
	if err != nil {
		return err
	}

	for _, d := range entries {
		absPath := filepath.Join(currentDir, d.Name())
		relPath, relErr := filepath.Rel(s.configDir, absPath)
		if relErr != nil {
			return fmt.Errorf("compute relative path: %w", relErr)
		}
		rel := strings.ReplaceAll(relPath, "\\", "/")

		if d.Type()&fs.ModeSymlink != 0 {
			if err := s.handleSymlink(d, absPath, rel); err != nil {
				return err
			}
			continue
		}

		if err := s.handleRegular(d, absPath, rel); err != nil {
			return err
		}
	}
	return nil
}

func (s *dirScanner) handleSymlink(d fs.DirEntry, absPath, rel string) error {
	targetStat, targetReal, ok := resolveSymlink(absPath, rel, s.homeDir, s.verbose)
	if !ok {
		return nil
	}
	if matchesExclude(d.Name(), rel, targetStat.IsDir(), s.opts) {
		return nil
	}
	if targetStat.IsDir() {
		canonical := paths.CanonicalPath(targetReal)
		if s.visitedDirs[canonical] {
			if s.verbose {
				_ = emitSymlinkWarning(rel, fmt.Errorf("cycle detected: %s", targetReal))
			}
			return nil
		}
		s.visitedDirs[canonical] = true
		s.items = append(s.items, createDirItem(absPath, rel, s.category, targetStat))
		return s.walk(absPath)
	}

	if isOversized(targetStat.Size(), s.opts.MaxFileSize, rel) {
		return nil
	}
	item, err := createFileItem(absPath, rel, s.category, targetStat)
	if err != nil {
		return err
	}
	s.items = append(s.items, item)
	return nil
}

func (s *dirScanner) handleRegular(d fs.DirEntry, absPath, rel string) error {
	if matchesExclude(d.Name(), rel, d.IsDir(), s.opts) {
		return nil
	}

	info, statErr := d.Info()
	if statErr != nil {
		return fmt.Errorf("stat %s: %w", rel, statErr)
	}

	if d.IsDir() {
		if realSub, err := filepath.EvalSymlinks(absPath); err == nil {
			s.visitedDirs[paths.CanonicalPath(realSub)] = true
		}
		s.items = append(s.items, createDirItem(absPath, rel, s.category, info))
		return s.walk(absPath)
	}

	if isOversized(info.Size(), s.opts.MaxFileSize, rel) {
		return nil
	}

	item, err := createFileItem(absPath, rel, s.category, info)
	if err != nil {
		return err
	}
	s.items = append(s.items, item)
	return nil
}

// scanDir recursively walks a directory and returns an Item for every
// file and subdirectory found. Directories receive a zero hash and size.
// Symlinks pointing within the user home are resolved: directory targets are
// traversed recursively, while regular file targets are hashed and recorded.
// Broken symlinks or symlinks escaping the home directory are skipped (with a
// warning when verbose is true). When opts is non-zero, entries matching
// exclude patterns or exceeding MaxFileSize are skipped.
func scanDir(dir, category, configDir, homeDir string, opts ScanOptions, verbose bool) ([]Item, error) {
	if _, err := os.Stat(dir); err != nil {
		return nil, err
	}

	s := &dirScanner{
		category:    category,
		configDir:   configDir,
		homeDir:     homeDir,
		opts:        opts,
		verbose:     verbose,
		visitedDirs: initAncestorDirs(dir, homeDir),
	}

	if err := s.walk(dir); err != nil {
		return nil, err
	}
	return s.items, nil
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

// stderrWriter is the sink for oversize warnings. It delegates to os.Stderr
// on every Write so tests that swap os.Stderr (captureStderr) still capture
// output, while tests that need a failing writer can replace stderrWriter
// directly to verify the scan continues when stderr writes fail (spec
// scenario "stderr write failure does not abort scan"). Mirrors the AGENTS.md
// injection-via-var pattern (e.g. var execCommand = exec.Command).
type stderrSink struct{}

func (stderrSink) Write(p []byte) (int, error) { return os.Stderr.Write(p) }

var stderrWriter io.Writer = stderrSink{}

// emitOversizeWarning writes the shared "skipping large file" warning to
// stderrWriter. Both scanDir and scanRootFiles use it so the message format
// stays identical in one place. It returns the write error (if any) so callers
// can log it to verbose output and continue rather than aborting the walk.
func emitOversizeWarning(size, max int64, rel string) error {
	_, err := fmt.Fprintf(stderrWriter, "warning: skipping large file (%d bytes exceeds max %d): %s\n",
		size, max, rel)
	return err
}

// emitSymlinkWarning writes the shared "skipping symlink" warning to
// stderrWriter. Both scanDir and scanRootFiles use it so the message format
// stays identical in one place.
func emitSymlinkWarning(rel string, err error) error {
	_, writeErr := fmt.Fprintf(stderrWriter, "warning: skipping symlink %s: %v\n", rel, err)
	return writeErr
}

// scanRootFiles reads the top-level config directory and returns items
// for all regular files that belong to categories in catSet. When opts
// is non-zero, entries matching exclude patterns or exceeding MaxFileSize
// are skipped — mirroring scanDir's filtering. When rootConfigFiles is
// non-nil, only file names present in the map are included (whitelist).
// Symlinks to regular files within the user home are followed; broken links
// or links escaping home are skipped with a warning when verbose is true.
func scanRootFiles(configDir, homeDir string, catSet map[string]bool, opts ScanOptions, rootConfigFiles map[string]string, verbose bool) ([]Item, error) {
	entries, err := os.ReadDir(configDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read config dir: %w", err)
	}

	var items []Item
	for _, e := range entries {
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

		var info fs.FileInfo
		if e.Type()&fs.ModeSymlink != 0 {
			targetStat, _, ok := resolveSymlink(absPath, entryName, homeDir, verbose)
			if !ok || targetStat.IsDir() {
				continue
			}
			info = targetStat
		} else {
			if e.IsDir() {
				continue
			}
			var infoErr error
			info, infoErr = e.Info()
			if infoErr != nil {
				return nil, fmt.Errorf("stat %s: %w", entryName, infoErr)
			}
		}

		if isOversized(info.Size(), opts.MaxFileSize, entryName) {
			continue
		}

		item, err := createFileItem(absPath, entryName, cat, info)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
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
