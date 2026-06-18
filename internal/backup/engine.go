package backup

import (
	"fmt"
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

// BakDir returns the bak storage directory (~/.bak).
func BakDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("home dir: %w", err)
	}
	return filepath.Join(home, ".bak"), nil
}

// Engine orchestrates the backup workflow: detect adapters, resolve
// presets, copy files, scan secrets, and produce a manifest.
type Engine struct {
	HomeDir          string             // user home directory
	BakDir           string             // ~/.bak storage root
	Registry         *adapters.Registry // adapter registry
	Preset           string             // preset name (quick, full, skills)
	AdapterFilter    []string           // optional: run only these adapters
	BakVersion       string             // bak binary version
	Verbose          bool               // enable verbose output
	SecretPatterns   []*regexp.Regexp   // patterns for secret detection; nil = defaults
	CustomCategories []string           // custom categories from TUI picker; overrides preset

	// ProgressFn is an optional callback invoked once per file during backup.
	// When nil (default), no progress is reported.
	ProgressFn func(currentFile string, filesDone int, filesTotal int)
}

// Result summarizes a completed backup operation.
type Result struct {
	ID          string // backup ID (timestamp)
	BackupDir   string // full path to backup directory
	FileCount   int    // total files backed up
	TotalSize   int64  // total bytes
	Secrets     int    // number of secret-bearing files detected
	AdaptersRun int    // number of adapters that contributed
}

// Run executes the full backup flow and returns a summary.
func (e *Engine) Run() (*Result, error) {
	// --- 1. Resolve preset ------------------------------------------------
	// --- 1. Resolve categories -------------------------------------------
	var categories []string
	if len(e.CustomCategories) > 0 {
		// Use custom categories from TUI picker (overrides preset).
		categories = e.CustomCategories
	} else {
		var err error
		categories, err = presets.Resolve(e.Preset)
		if err != nil {
			return nil, fmt.Errorf("resolve preset %q: %w", e.Preset, err)
		}
	}

	// --- 2. Identify which adapters to run --------------------------------
	var detected []adapters.DetectedAdapter

	if len(e.AdapterFilter) > 0 {
		for _, filterName := range e.AdapterFilter {
			a, ok := e.Registry.Get(filterName)
			if !ok {
				return nil, fmt.Errorf("adapter %q not registered", filterName)
			}
			installed, configDir, err := a.Detect(e.HomeDir)
			if err != nil {
				return nil, fmt.Errorf("detect %q: %w", filterName, err)
			}
			if !installed {
				return nil, fmt.Errorf("adapter %q: config directory not found", filterName)
			}
			detected = append(detected, adapters.DetectedAdapter{
				Adapter:   a,
				ConfigDir: configDir,
			})
		}
	} else {
		detected = e.Registry.DetectAll(e.HomeDir)
	}

	if len(detected) == 0 {
		return nil, fmt.Errorf("no installed adapters detected")
	}

	// --- 3. Create backup directory ---------------------------------------
	backupID := time.Now().UTC().Format("20060102-150405")
	backupDir := filepath.Join(e.BakDir, "backups", backupID)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return nil, fmt.Errorf("create backup dir: %w", err)
	}

	// --- 4. Build manifest ------------------------------------------------
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
		if e.Verbose {
			fmt.Fprintf(os.Stderr, "warning: could not get hostname: %v\n", err)
		}
	}
	m := manifest.New(backupID, runtime.GOOS, hostname, e.BakVersion, e.Preset, categories)

	// --- 5. Validate manifest is writable (fail-fast) ---------------------
	if err := m.Save(backupDir); err != nil {
		return nil, fmt.Errorf("save manifest (fail-fast): %w", err)
	}

	// Fallback on cleanup; caller should inspect Result to know whether to keep.
	var cleanupOnError = true
	defer func() {
		if cleanupOnError {
			if removeErr := os.RemoveAll(backupDir); removeErr != nil && e.Verbose {
				fmt.Fprintf(os.Stderr, "warning: cleanup failed: %v\n", removeErr)
			}
		}
	}()

	// --- 5. Collect items and backup ------------------------------------
	patterns := e.SecretPatterns
	if patterns == nil {
		patterns = DefaultPatterns()
	}

	// First pass: collect items from all adapters to compute total.
	type adapterItems struct {
		adapter adapters.DetectedAdapter
		items   []adapters.Item
	}
	var allItems []adapterItems
	filesTotal := 0
	for _, d := range detected {
		items, err := d.Adapter.ListItems(e.HomeDir, categories)
		if err != nil {
			return nil, fmt.Errorf("list items for %q: %w", d.Adapter.Name(), err)
		}
		allItems = append(allItems, adapterItems{adapter: d, items: items})
		for _, item := range items {
			if !item.IsDir {
				filesTotal++
			}
		}
	}

	// Second pass: backup, scan secrets, and build manifest with progress.
	var allSecretFiles []string
	totalFiles := 0
	var totalSize int64
	filesDone := 0

	for _, entry := range allItems {
		d := entry.adapter

		if err := d.Adapter.Backup(e.HomeDir, backupDir, entry.items); err != nil {
			return nil, fmt.Errorf("backup %q: %w", d.Adapter.Name(), err)
		}

		// Scan for secrets in the backed-up files.
		secretFiles := scanBackupForSecrets(backupDir, filepath.Join(backupDir, d.Adapter.Name()), patterns, e.Verbose)
		allSecretFiles = append(allSecretFiles, secretFiles...)

		// Exclude secret-containing files from backup (security requirement).
		secretRelPaths := make(map[string]bool)
		for _, secretFile := range secretFiles {
			if rel, err := filepath.Rel(backupDir, secretFile); err == nil {
				secretRelPaths[paths.Slash(rel)] = true
			}
			if err := os.Remove(secretFile); err != nil && e.Verbose {
				fmt.Fprintf(os.Stderr, "warning: could not remove secret file: %v\n", err)
			}
		}

		// Convert adapter items to manifest items.
		manifestItems := make([]manifest.Item, 0, len(entry.items))
		for _, item := range entry.items {
			if item.IsDir {
				continue // manifest tracks files only
			}

			// Progress callback — nil-safe.
			filesDone++
			if e.ProgressFn != nil {
				e.ProgressFn(item.RelPath, filesDone, filesTotal)
			}

			backupPath := paths.Slash(filepath.Join(d.Adapter.Name(), item.RelPath))

			// Skip items whose backup file was removed (contained secrets).
			if secretRelPaths[backupPath] {
				continue
			}

			// Security: validate source path stays under home directory.
			absSource := item.SourcePath
			if strings.HasPrefix(absSource, "~/") {
				absSource = paths.FromCanonical(absSource, e.HomeDir)
			}
			cleanSource := paths.CanonicalPath(absSource)
			cleanHome := paths.CanonicalPath(e.HomeDir) + "/"
			if !strings.HasPrefix(strings.ToLower(cleanSource), strings.ToLower(cleanHome)) &&
				!strings.EqualFold(cleanSource, paths.CanonicalPath(e.HomeDir)) {
				return nil, fmt.Errorf("adapter %q returned source path outside home directory", d.Adapter.Name())
			}

			manifestItems = append(manifestItems, manifest.Item{
				Category:   item.Category,
				SourcePath: item.SourcePath,
				BackupPath: backupPath,
				Hash:       item.Hash,
				Size:       item.Size,
			})
			totalFiles++
			totalSize += item.Size
		}

		configDirCanonical := paths.ToCanonical(d.ConfigDir)
		m.AddAdapter(d.Adapter.Name(), "", configDirCanonical, manifestItems)
	}

	// --- 5. Generate .env.example -----------------------------------------
	secretsExcluded := len(allSecretFiles) > 0
	if secretsExcluded {
		if err := GenerateEnvExample(allSecretFiles, patterns, backupDir); err != nil {
			return nil, fmt.Errorf("generate .env.example: %w", err)
		}
	}
	m.SecretsExcluded = secretsExcluded

	// --- 6. Save manifest -------------------------------------------------
	if err := m.Save(backupDir); err != nil {
		return nil, fmt.Errorf("save manifest: %w", err)
	}

	cleanupOnError = false

	return &Result{
		ID:          backupID,
		BackupDir:   backupDir,
		FileCount:   totalFiles,
		TotalSize:   totalSize,
		Secrets:     len(allSecretFiles),
		AdaptersRun: len(detected),
	}, nil
}

// scanBackupForSecrets walks the adapter's backup directory and collects
// paths of files that contain secrets.
func scanBackupForSecrets(backupRoot, adapterBackupDir string, patterns []*regexp.Regexp, verbose bool) []string {
	var secretFiles []string

	if err := filepath.WalkDir(adapterBackupDir, func(fpath string, d os.DirEntry, err error) error {
		if err != nil {
			if verbose {
				fmt.Fprintf(os.Stderr, "warning: walk %s: %v\n", fpath, err)
			}
			// Skip entries with access errors and continue walking.
			return nil //nolint:nilerr
		}
		if d.IsDir() {
			return nil
		}
		results, scanErr := ScanFile(fpath, patterns)
		if scanErr != nil {
			if verbose {
				fmt.Fprintf(os.Stderr, "warning: scan %s: %v\n", fpath, scanErr)
			}
			// Skip unreadable files and continue walking.
			return nil //nolint:nilerr
		}
		if len(results) > 0 {
			secretFiles = append(secretFiles, fpath)
		}
		return nil
	}); err != nil {
		// Walk error — return whatever we found so far.
		return secretFiles
	}

	return secretFiles
}
