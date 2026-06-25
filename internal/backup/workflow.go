package backup

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/danielxxomg/bak-cli/internal/adapters"
	"github.com/danielxxomg/bak-cli/internal/manifest"
	"github.com/danielxxomg/bak-cli/internal/paths"
	"github.com/danielxxomg/bak-cli/internal/presets"
)

// FS is the filesystem contract the consolidated backup workflow depends on.
// It is a subset of actions.FileSystem; the concrete actions.OSFileSystem,
// actions.MockFileSystem, and any struct embedding them satisfy backup.FS
// structurally (identical method names and signatures).
type FS interface {
	UserHomeDir() (string, error)
	Stat(path string) (os.FileInfo, error)
	ReadDir(dirname string) ([]os.DirEntry, error)
	MkdirAll(path string, perm os.FileMode) error
	RemoveAll(path string) error
	WalkDir(root string, fn fs.WalkDirFunc) error
	WriteFile(filename string, data []byte, perm os.FileMode) error
}

// osFS implements FS using the real operating system. It is the default
// filesystem for callers (Engine) that do not inject one.
type osFS struct{}

func (osFS) UserHomeDir() (string, error)                 { return os.UserHomeDir() }
func (osFS) Stat(path string) (os.FileInfo, error)        { return os.Stat(path) }
func (osFS) ReadDir(name string) ([]os.DirEntry, error)   { return os.ReadDir(name) }
func (osFS) MkdirAll(path string, perm os.FileMode) error { return os.MkdirAll(path, perm) }
func (osFS) RemoveAll(path string) error                  { return os.RemoveAll(path) }
func (osFS) WalkDir(root string, fn fs.WalkDirFunc) error { return filepath.WalkDir(root, fn) }
func (osFS) WriteFile(name string, data []byte, perm os.FileMode) error {
	return os.WriteFile(name, data, perm)
}

// Context bundles every input the canonical backup workflow needs. Both
// BackupAction.Run (CLI) and Engine.Run (TUI) build a Context and delegate to
// Run so that both paths use byte-identical implementation.
type Context struct {
	// FS performs all filesystem operations. nil falls back to osFS at Run
	// time so callers that leave it zero-valued (existing &Engine{...}
	// instantiations) keep working unchanged.
	FS FS

	// HomeDir and BakDir are the user home directory and the ~/.bak storage
	// root. Both must be resolved by the caller; Run does not re-derive them.
	HomeDir string
	BakDir  string

	Registry         *adapters.Registry
	Preset           string
	AdapterFilter    []string
	BakVersion       string
	Verbose          bool
	SecretPatterns   []*regexp.Regexp
	CustomCategories []string

	// ProgressFn is an optional callback invoked once per non-directory file
	// during the manifest-building pass. nil means no progress reporting.
	ProgressFn ProgressFn

	// ExcludesLoader returns scan options applied to ScanConfigurable
	// adapters before ListItems is called. nil means no exclusions.
	ExcludesLoader func() (adapters.ScanOptions, error)

	// HostnameFn returns the current hostname. nil falls back to os.Hostname.
	HostnameFn func() (string, error)

	// Stderr receives verbose warnings and diagnostics. nil falls back to
	// os.Stderr at Run time.
	Stderr io.Writer
}

// Run executes the canonical 8-phase backup workflow:
//
//  1. resolve categories (custom override or preset)
//  2. detect adapters (filtered or all)
//  3. apply exclusion rules to ScanConfigurable adapters
//  4. create the timestamped backup directory
//  5. build the initial manifest (hostname resolution, fail-fast save)
//  6. collect items, backup, scan+remove secret files, build manifest items
//  7. generate .env.example when secrets were detected
//  8. save the final manifest and return the summary
//
// Secret handling adopts the Engine.Run canonical behavior: secret-bearing
// files are removed from the backup directory via FS.RemoveAll AND excluded
// from the manifest's Items via a secretRelPaths skip-map, so no dangling
// references remain. This fixes the pre-consolidation CLI bug where secret
// files were removed from disk but left in the manifest.
func Run(ctx Context) (*Result, error) {
	fsys := ctx.FS
	if fsys == nil {
		fsys = osFS{}
	}
	stderr := ctx.Stderr
	if stderr == nil {
		stderr = os.Stderr
	}

	// --- 1. Resolve categories -------------------------------------------
	categories, err := resolveCategories(ctx)
	if err != nil {
		return nil, err
	}

	// --- 2. Identify which adapters to run --------------------------------
	detected, err := detectAdapters(ctx.Registry, ctx.HomeDir, ctx.AdapterFilter)
	if err != nil {
		return nil, err
	}
	if len(detected) == 0 {
		return nil, fmt.Errorf("no installed adapters detected")
	}

	// --- 3. Apply exclusion rules (if loader is set) ----------------------
	if err := applyExcludes(ctx, detected); err != nil {
		return nil, err
	}

	// --- 4. Create backup directory ---------------------------------------
	backupID := time.Now().UTC().Format("20060102-150405")
	backupDir := filepath.Join(ctx.BakDir, "backups", backupID)
	if err := fsys.MkdirAll(backupDir, 0755); err != nil {
		return nil, fmt.Errorf("create backup dir: %w", err)
	}

	// --- 5. Build manifest (hostname + fail-fast save) --------------------
	hostname := resolveHostname(ctx.HostnameFn, ctx.Verbose, stderr)
	m := manifest.New(backupID, runtime.GOOS, hostname, ctx.BakVersion, ctx.Preset, categories)
	if err := saveManifest(fsys, m, backupDir); err != nil {
		return nil, fmt.Errorf("save manifest (fail-fast): %w", err)
	}

	// Roll back the backup directory on error.
	cleanupOnError := true
	defer func() {
		if cleanupOnError {
			if removeErr := fsys.RemoveAll(backupDir); removeErr != nil && ctx.Verbose {
				fmt.Fprintf(stderr, "warning: cleanup failed: %v\n", removeErr) //nolint:errcheck // non-critical diagnostic
			}
		}
	}()

	patterns := ctx.SecretPatterns
	if patterns == nil {
		patterns = DefaultPatterns()
	}

	// --- 6. Collect items, backup, scan secrets, build manifest items -----
	// First pass: collect items from every adapter to compute the total file
	// count for progress reporting.
	allItems, filesTotal, err := collectAdapterItems(detected, ctx.HomeDir, categories)
	if err != nil {
		return nil, err
	}

	// Second pass: backup, scan for secrets, remove secret files, and build
	// the manifest items with the secretRelPaths skip-map.
	allSecretFiles, totalFiles, totalSize, err := backupAndBuildManifest(
		ctx, fsys, allItems, backupDir, patterns, filesTotal, m, stderr,
	)
	if err != nil {
		return nil, err
	}

	// --- 7. Generate .env.example when secrets were detected -------------
	secretsExcluded := len(allSecretFiles) > 0
	if secretsExcluded {
		if err := GenerateEnvExample(allSecretFiles, patterns, backupDir); err != nil {
			return nil, fmt.Errorf("generate .env.example: %w", err)
		}
	}
	m.SecretsExcluded = secretsExcluded

	// --- 8. Save final manifest -------------------------------------------
	if err := saveManifest(fsys, m, backupDir); err != nil {
		return nil, fmt.Errorf("save manifest: %w", err)
	}

	cleanupOnError = false

	return &Result{
		ID:              backupID,
		BackupDir:       backupDir,
		FileCount:       totalFiles,
		TotalSize:       totalSize,
		Secrets:         len(allSecretFiles),
		SecretsExcluded: secretsExcluded,
		AdaptersRun:     len(detected),
		Preset:          ctx.Preset,
	}, nil
}

// resolveCategories returns the custom categories when provided, otherwise
// resolves them from the named preset.
func resolveCategories(ctx Context) ([]string, error) {
	if len(ctx.CustomCategories) > 0 {
		return ctx.CustomCategories, nil
	}
	cats, err := presets.Resolve(ctx.Preset)
	if err != nil {
		return nil, fmt.Errorf("resolve preset %q: %w", ctx.Preset, err)
	}
	return cats, nil
}

// applyExcludes loads scan options (when an ExcludesLoader is configured) and
// applies them to every ScanConfigurable adapter before ListItems runs. nil
// loader means no exclusions.
func applyExcludes(ctx Context, detected []adapters.DetectedAdapter) error {
	if ctx.ExcludesLoader == nil {
		return nil
	}
	opts, err := ctx.ExcludesLoader()
	if err != nil {
		return fmt.Errorf("load excludes: %w", err)
	}
	for _, d := range detected {
		if sc, ok := d.Adapter.(adapters.ScanConfigurable); ok {
			sc.SetScanOptions(opts)
		}
	}
	return nil
}

// adapterItems pairs a detected adapter with the items its ListItems call
// returned, so the second pass can iterate without re-listing.
type adapterItems struct {
	adapter adapters.DetectedAdapter
	items   []adapters.Item
}

// collectAdapterItems runs the first pass: ask every detected adapter for its
// items and tally the non-directory files so progress reporting knows the
// total upfront.
func collectAdapterItems(detected []adapters.DetectedAdapter, homeDir string, categories []string) ([]adapterItems, int, error) {
	var allItems []adapterItems
	filesTotal := 0
	for _, d := range detected {
		items, err := d.Adapter.ListItems(homeDir, categories)
		if err != nil {
			return nil, 0, fmt.Errorf("list items for %q: %w", d.Adapter.Name(), err)
		}
		allItems = append(allItems, adapterItems{adapter: d, items: items})
		for _, item := range items {
			if !item.IsDir {
				filesTotal++
			}
		}
	}
	return allItems, filesTotal, nil
}

// backupAndBuildManifest runs the second pass: for each adapter it backs the
// items up, scans for secrets, removes secret files from the backup dir, and
// builds the manifest entries with the secretRelPaths skip-map so no dangling
// references remain. It mutates m by adding one adapter section per pass.
func backupAndBuildManifest(
	ctx Context,
	fsys FS,
	allItems []adapterItems,
	backupDir string,
	patterns []*regexp.Regexp,
	filesTotal int,
	m *manifest.Manifest,
	stderr io.Writer,
) (allSecretFiles []string, totalFiles int, totalSize int64, err error) {
	filesDone := 0
	for _, entry := range allItems {
		d := entry.adapter

		if err := d.Adapter.Backup(ctx.HomeDir, backupDir, entry.items); err != nil {
			return nil, 0, 0, fmt.Errorf("backup %q: %w", d.Adapter.Name(), err)
		}

		adapterBackupDir := filepath.Join(backupDir, d.Adapter.Name())
		secretFiles := scanBackupForSecretsFS(fsys, adapterBackupDir, patterns, ctx.Verbose, stderr)
		allSecretFiles = append(allSecretFiles, secretFiles...)

		secretRelPaths := removeSecretFiles(fsys, secretFiles, backupDir, ctx.Verbose, stderr)

		items, files, size, fdone, berr := buildAdapterManifestItems(entry, ctx, d, backupDir, secretRelPaths, filesDone, filesTotal)
		if berr != nil {
			return nil, 0, 0, berr
		}
		filesDone = fdone
		totalFiles += files
		totalSize += size

		m.AddAdapter(d.Adapter.Name(), "", paths.ToCanonical(d.ConfigDir), items)
	}
	return allSecretFiles, totalFiles, totalSize, nil
}

// removeSecretFiles builds the secretRelPaths skip-map and removes each
// secret-bearing file from the backup directory via FS.RemoveAll (handles
// directories containing only secrets). The skip-map keys are backup-relative
// slash paths so the manifest builder can match item.BackupPath exactly.
func removeSecretFiles(fsys FS, secretFiles []string, backupDir string, verbose bool, stderr io.Writer) map[string]bool {
	secretRelPaths := make(map[string]bool)
	for _, secretFile := range secretFiles {
		if rel, relErr := filepath.Rel(backupDir, secretFile); relErr == nil {
			secretRelPaths[paths.Slash(rel)] = true
		}
		if rmErr := fsys.RemoveAll(secretFile); rmErr != nil && verbose {
			fmt.Fprintf(stderr, "warning: could not remove secret file: %v\n", rmErr) //nolint:errcheck // non-critical diagnostic
		}
	}
	return secretRelPaths
}

// buildAdapterManifestItems walks one adapter's items, advancing progress,
// skipping removed-secret entries, validating source paths stay under the
// home dir, and assembling the manifest.Item slice. It returns the per-adapter
// item count and size deltas plus the updated running filesDone counter.
func buildAdapterManifestItems(
	entry adapterItems,
	ctx Context,
	d adapters.DetectedAdapter,
	backupDir string,
	secretRelPaths map[string]bool,
	filesDone, filesTotal int,
) (items []manifest.Item, files int, size int64, done int, err error) {
	items = make([]manifest.Item, 0, len(entry.items))
	for _, item := range entry.items {
		if item.IsDir {
			continue // manifest tracks files only
		}

		filesDone++
		if ctx.ProgressFn != nil {
			ctx.ProgressFn(item.RelPath, filesDone, filesTotal)
		}

		backupPath := paths.Slash(filepath.Join(d.Adapter.Name(), item.RelPath))

		// Skip items whose backed-up file was removed (contained secrets) so
		// the manifest never carries dangling references.
		if secretRelPaths[backupPath] {
			continue
		}

		// Security: validate the source path stays under the home dir.
		absSource := item.SourcePath
		if strings.HasPrefix(absSource, "~/") {
			absSource = paths.FromCanonical(absSource, ctx.HomeDir)
		}
		cleanSource := paths.CanonicalPath(absSource)
		cleanHome := paths.CanonicalPath(ctx.HomeDir) + "/"
		if !strings.HasPrefix(strings.ToLower(cleanSource), strings.ToLower(cleanHome)) &&
			!strings.EqualFold(cleanSource, paths.CanonicalPath(ctx.HomeDir)) {
			return nil, 0, 0, filesDone, fmt.Errorf("adapter %q returned source path outside home directory", d.Adapter.Name())
		}

		items = append(items, manifest.Item{
			Category:   item.Category,
			SourcePath: item.SourcePath,
			BackupPath: backupPath,
			Hash:       item.Hash,
			Size:       item.Size,
		})
		files++
		size += item.Size
	}
	return items, files, size, filesDone, nil
}

// detectAdapters resolves the detected adapter set, honoring an explicit
// AdapterFilter when provided and falling back to registry auto-discovery.
func detectAdapters(reg *adapters.Registry, homeDir string, filter []string) ([]adapters.DetectedAdapter, error) {
	if reg == nil {
		return nil, nil
	}
	if len(filter) == 0 {
		return reg.DetectAll(homeDir), nil
	}

	var detected []adapters.DetectedAdapter
	for _, name := range filter {
		a, ok := reg.Get(name)
		if !ok {
			return nil, fmt.Errorf("adapter %q not registered", name)
		}
		installed, configDir, err := a.Detect(homeDir)
		if err != nil {
			return nil, fmt.Errorf("detect %q: %w", name, err)
		}
		if !installed {
			return nil, fmt.Errorf("adapter %q: config directory not found", name)
		}
		detected = append(detected, adapters.DetectedAdapter{
			Adapter:   a,
			ConfigDir: configDir,
		})
	}
	return detected, nil
}

// resolveHostname returns the hostname via the injected function, falling
// back to os.Hostname when fn is nil. Errors default to "unknown" and, when
// verbose is set, emit a warning to stderr.
func resolveHostname(fn func() (string, error), verbose bool, stderr io.Writer) string {
	hostnameFn := fn
	if hostnameFn == nil {
		hostnameFn = os.Hostname
	}
	hostname, err := hostnameFn()
	if err != nil {
		if verbose {
			fmt.Fprintf(stderr, "warning: could not get hostname: %v\n", err) //nolint:errcheck // non-critical diagnostic
		}
		return "unknown"
	}
	return hostname
}

// saveManifest serializes m as indented JSON and writes it through fs. Keeping
// the marshaling in one place guarantees byte-identical manifest output across
// the CLI and TUI paths.
func saveManifest(fs FS, m *manifest.Manifest, dir string) error {
	path := filepath.Join(dir, "manifest.json")
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	if err := fs.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}
	return nil
}

// scanBackupForSecretsFS walks the adapter's backup directory via fs and
// collects absolute paths of files that contain secrets. Per-entry access
// and scan errors are warned (when verbose) and skipped so the walk keeps
// scanning siblings; a top-level walk error (e.g. missing root) is surfaced
// via a verbose warning rather than silently dropped.
func scanBackupForSecretsFS(fsys FS, adapterBackupDir string, patterns []*regexp.Regexp, verbose bool, stderr io.Writer) []string {
	var secretFiles []string

	err := fsys.WalkDir(adapterBackupDir, func(fpath string, d os.DirEntry, err error) error {
		if err != nil {
			if verbose {
				fmt.Fprintf(stderr, "warning: walk %s: %v\n", fpath, err) //nolint:errcheck // non-critical diagnostic
			}
			// Skip entries with access errors and continue walking.
			return nil //nolint:nilerr // intentional: continue the walk
		}
		if d.IsDir() {
			return nil
		}
		results, scanErr := ScanFile(fpath, patterns)
		if scanErr != nil {
			if verbose {
				fmt.Fprintf(stderr, "warning: scan %s: %v\n", fpath, scanErr) //nolint:errcheck // non-critical diagnostic
			}
			return nil //nolint:nilerr // intentional: keep scanning siblings
		}
		if len(results) > 0 {
			secretFiles = append(secretFiles, fpath)
		}
		return nil
	})
	if err != nil && verbose {
		fmt.Fprintf(stderr, "warning: secret scan walk %s: %v\n", adapterBackupDir, err) //nolint:errcheck // non-critical diagnostic
	}

	return secretFiles
}
