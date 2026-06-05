package backup

import (
	"fmt"
	"os"
	"path"
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
	HomeDir         string           // user home directory
	BakDir          string           // ~/.bak storage root
	Registry        *adapters.Registry // adapter registry
	Preset          string           // preset name (quick, full, skills)
	AdapterFilter   string           // optional: run only this adapter
	BakVersion      string           // bak binary version
	Verbose         bool             // enable verbose output
	SecretPatterns  []*regexp.Regexp // patterns for secret detection; nil = defaults
	CustomCategories []string        // custom categories from TUI picker; overrides preset
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

	if e.AdapterFilter != "" {
		a, ok := e.Registry.Get(e.AdapterFilter)
		if !ok {
			return nil, fmt.Errorf("adapter %q not registered", e.AdapterFilter)
		}
		installed, configDir, err := a.Detect(e.HomeDir)
		if err != nil {
			return nil, fmt.Errorf("detect %q: %w", e.AdapterFilter, err)
		}
		if !installed {
			return nil, fmt.Errorf("adapter %q: %s not found", e.AdapterFilter, configDir)
		}
		detected = append(detected, adapters.DetectedAdapter{
			Adapter:   a,
			ConfigDir: configDir,
		})
	} else {
		detected = e.Registry.DetectAll(e.HomeDir)
	}

	if len(detected) == 0 {
		return nil, fmt.Errorf("no installed adapters detected (home: %s)", e.HomeDir)
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

	patterns := e.SecretPatterns
	if patterns == nil {
		patterns = DefaultPatterns()
	}

	var allSecretFiles []string
	totalFiles := 0
	var totalSize int64

	for _, d := range detected {
		items, err := d.Adapter.ListItems(e.HomeDir, categories)
		if err != nil {
			return nil, fmt.Errorf("list items for %q: %w", d.Adapter.Name(), err)
		}

		if err := d.Adapter.Backup(e.HomeDir, backupDir, items); err != nil {
			return nil, fmt.Errorf("backup %q: %w", d.Adapter.Name(), err)
		}

		// Scan for secrets in the backed-up files.
		secretFiles := scanBackupForSecrets(backupDir, filepath.Join(backupDir, d.Adapter.Name()), patterns)
		allSecretFiles = append(allSecretFiles, secretFiles...)

		// Exclude secret-containing files from backup (security requirement).
		for _, secretFile := range secretFiles {
			if err := os.Remove(secretFile); err != nil && e.Verbose {
				fmt.Fprintf(os.Stderr, "warning: could not remove secret file %s: %v\n", secretFile, err)
			}
		}

		// Convert adapter items to manifest items.
		manifestItems := make([]manifest.Item, 0, len(items))
		for _, item := range items {
			if item.IsDir {
				continue // manifest tracks files only
			}

			// Security: validate source path stays under home directory.
			// SourcePath may be canonical (~/...) or absolute — normalize both.
			absSource := item.SourcePath
			if strings.HasPrefix(absSource, "~/") {
				absSource = paths.FromCanonical(absSource, e.HomeDir)
			}
			cleanSource := path.Clean(filepath.ToSlash(absSource))
			cleanHome := path.Clean(filepath.ToSlash(e.HomeDir)) + "/"
			// Case-insensitive comparison for Windows (case-insensitive FS).
			if !strings.HasPrefix(strings.ToLower(cleanSource), strings.ToLower(cleanHome)) &&
				!strings.EqualFold(cleanSource, path.Clean(filepath.ToSlash(e.HomeDir))) {
				return nil, fmt.Errorf("adapter %q returned source path outside home: %s", d.Adapter.Name(), item.SourcePath)
			}

			manifestItems = append(manifestItems, manifest.Item{
				Category:   item.Category,
				SourcePath: item.SourcePath,
				BackupPath: filepath.ToSlash(filepath.Join(d.Adapter.Name(), item.RelPath)),
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
func scanBackupForSecrets(backupRoot, adapterBackupDir string, patterns []*regexp.Regexp) []string {
	var secretFiles []string

	if err := filepath.WalkDir(adapterBackupDir, func(fpath string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		results, scanErr := ScanFile(fpath, patterns)
		if scanErr != nil {
			return nil // skip unreadable files
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
