package backup

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
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
	HomeDir        string           // user home directory
	BakDir         string           // ~/.bak storage root
	Registry       *adapters.Registry // adapter registry
	Preset         string           // preset name (quick, full, skills)
	AdapterFilter  string           // optional: run only this adapter
	BakVersion     string           // bak binary version
	Verbose        bool             // enable verbose output
	SecretPatterns []*regexp.Regexp // patterns for secret detection; nil = defaults
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
	categories, err := presets.Resolve(e.Preset)
	if err != nil {
		return nil, fmt.Errorf("resolve preset %q: %w", e.Preset, err)
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
	hostname, _ := os.Hostname()
	m := manifest.New(backupID, runtime.GOOS, hostname, e.BakVersion, e.Preset, categories)

	// Fallback on cleanup; caller should inspect Result to know whether to keep.
	var cleanupOnError = true
	defer func() {
		if cleanupOnError {
			os.RemoveAll(backupDir)
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

		// Convert adapter items to manifest items.
		manifestItems := make([]manifest.Item, 0, len(items))
		for _, item := range items {
			if item.IsDir {
				continue // manifest tracks files only
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

	_ = filepath.WalkDir(adapterBackupDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		results, scanErr := ScanFile(path, patterns)
		if scanErr != nil {
			return nil // skip unreadable files
		}
		if len(results) > 0 {
			secretFiles = append(secretFiles, path)
		}
		return nil
	})

	return secretFiles
}
