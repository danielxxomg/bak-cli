package actions

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/danielxxomg/bak-cli/internal/adapters"
	"github.com/danielxxomg/bak-cli/internal/backup"
	"github.com/danielxxomg/bak-cli/internal/manifest"
	"github.com/danielxxomg/bak-cli/internal/paths"
	"github.com/danielxxomg/bak-cli/internal/presets"
)

// BackupAction encapsulates the backup workflow with injectable
// dependencies for testability. Replace os.* calls with a.FS.* and
// config.Load() with a.Config.Load().
type BackupAction struct {
	FS       FileSystem
	Config   ConfigLoader
	Registry *adapters.Registry

	// Stdout receives informational output. Nil falls back to os.Stdout.
	Stdout io.Writer
	// Stderr receives warnings and error diagnostics. Nil falls back to os.Stderr.
	Stderr io.Writer

	// Parameters (from CLI flags).
	Preset           string
	AdapterFilter    []string
	Verbose          bool
	BakVersion       string
	SecretPatterns   []*regexp.Regexp
	CustomCategories []string

	// ProgressFn is an optional callback invoked once per file during backup.
	// When nil (default), no progress is reported. Signature matches
	// backup.Engine.ProgressFn.
	ProgressFn func(currentFile string, filesDone int, filesTotal int)

	// ExcludesLoader returns scan options to filter files during backup.
	// nil means no exclusions. Wired by cmd/ to config.Load+LoadExcludes.
	ExcludesLoader func() (adapters.ScanOptions, error)

	// HostnameFn returns the current hostname. Nil falls back to os.Hostname.
	HostnameFn HostnameFunc
}

// Run executes the backup workflow: resolve preset, detect adapters,
// copy files, scan secrets, and write manifest. All OS operations go
// through a.FS.
func (a *BackupAction) Run() error { //nolint:maintidx // SEVERE: tracked for qa-refactor-analysis (needs extraction, not config)
	out := a.Stdout
	if out == nil {
		out = os.Stdout
	}
	errOut := a.Stderr
	if errOut == nil {
		errOut = os.Stderr
	}
	// 1. Determine home directory via injected FS.
	homeDir, err := a.FS.UserHomeDir()
	if err != nil {
		return fmt.Errorf("cannot determine home directory: %w", err)
	}

	// 2. Resolve categories from preset or custom.
	var categories []string
	if len(a.CustomCategories) > 0 {
		categories = a.CustomCategories
	} else {
		cats, err := presets.Resolve(a.Preset)
		if err != nil {
			return fmt.Errorf("resolve preset %q: %w", a.Preset, err)
		}
		categories = cats
	}

	// 3. Identify which adapters to run.
	var detected []adapters.DetectedAdapter

	if len(a.AdapterFilter) > 0 {
		for _, filterName := range a.AdapterFilter {
			adp, ok := a.Registry.Get(filterName)
			if !ok {
				return fmt.Errorf("adapter %q not registered", filterName)
			}
			installed, configDir, err := adp.Detect(homeDir)
			if err != nil {
				return fmt.Errorf("detect %q: %w", filterName, err)
			}
			if !installed {
				return fmt.Errorf("adapter %q: config directory not found", filterName)
			}
			detected = append(detected, adapters.DetectedAdapter{
				Adapter:   adp,
				ConfigDir: configDir,
			})
		}
	} else {
		detected = a.Registry.DetectAll(homeDir)
	}

	if len(detected) == 0 {
		return fmt.Errorf("no installed adapters detected")
	}

	// 3a. Apply exclusion rules (if loader is set).
	if a.ExcludesLoader != nil {
		opts, err := a.ExcludesLoader()
		if err != nil {
			return fmt.Errorf("load excludes: %w", err)
		}
		for _, d := range detected {
			if sc, ok := d.Adapter.(adapters.ScanConfigurable); ok {
				sc.SetScanOptions(opts)
			}
		}
	}

	// 4. Create backup directory.
	bakDir := filepath.Join(homeDir, ".bak")
	backupID := time.Now().UTC().Format("20060102-150405")
	backupDir := filepath.Join(bakDir, "backups", backupID)
	if err := a.FS.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("create backup dir: %w", err)
	}

	// 5. Build and save initial manifest (fail-fast).
	hostname := "unknown"
	if a.HostnameFn != nil {
		if h, err := a.HostnameFn(); err == nil {
			hostname = h
		} else if a.Verbose {
			warnf(errOut, "warning: could not get hostname: %v\n", err)
		}
	} else if h, err := os.Hostname(); err == nil {
		hostname = h
	} else if a.Verbose {
		warnf(errOut, "warning: could not get hostname: %v\n", err)
	}

	m := manifest.New(backupID, runtime.GOOS, hostname, a.BakVersion, a.Preset, categories)
	if err := a.saveManifest(m, backupDir); err != nil {
		return fmt.Errorf("save manifest (fail-fast): %w", err)
	}

	// Cleanup on error.
	var cleanupOnError = true
	defer func() {
		if cleanupOnError {
			if err := a.FS.RemoveAll(backupDir); err != nil && a.Verbose {
				warnf(errOut, "warning: cleanup failed: %v\n", err)
			}
		}
	}()

	patterns := a.SecretPatterns
	if patterns == nil {
		patterns = backup.DefaultPatterns()
	}

	// --- 5. Collect items from all adapters to compute total. ------------
	type adapterItems struct {
		adapter adapters.DetectedAdapter
		items   []adapters.Item
	}
	var allItems []adapterItems
	filesTotal := 0
	for _, d := range detected {
		items, err := d.Adapter.ListItems(homeDir, categories)
		if err != nil {
			return fmt.Errorf("list items for %q: %w", d.Adapter.Name(), err)
		}
		allItems = append(allItems, adapterItems{adapter: d, items: items})
		for _, item := range items {
			if !item.IsDir {
				filesTotal++
			}
		}
	}

	// --- 6. Backup and build manifest with progress. --------------------
	var allSecretFiles []string
	totalFiles := 0
	var totalSize int64
	filesDone := 0

	for _, entry := range allItems {
		d := entry.adapter

		if err := d.Adapter.Backup(homeDir, backupDir, entry.items); err != nil {
			return fmt.Errorf("backup %q: %w", d.Adapter.Name(), err)
		}

		// Scan backed-up files for secrets.
		adapterBackupDir := filepath.Join(backupDir, d.Adapter.Name())
		secretFiles := a.scanBackupForSecrets(adapterBackupDir, patterns)
		allSecretFiles = append(allSecretFiles, secretFiles...)

		// Remove secret-bearing files.
		for _, sf := range secretFiles {
			if err := a.FS.RemoveAll(sf); err != nil && a.Verbose {
				warnf(errOut, "warning: could not remove secret file: %v\n", err)
			}
		}

		// Build manifest items with path traversal validation.
		manifestItems := make([]manifest.Item, 0, len(entry.items))
		for _, item := range entry.items {
			if item.IsDir {
				continue
			}

			// Progress callback — nil-safe.
			filesDone++
			if a.ProgressFn != nil {
				a.ProgressFn(item.RelPath, filesDone, filesTotal)
			}

			absSource := item.SourcePath
			if strings.HasPrefix(absSource, "~/") {
				absSource = paths.FromCanonical(absSource, homeDir)
			}
			cleanSource := paths.CanonicalPath(absSource)
			cleanHome := paths.CanonicalPath(homeDir) + "/"
			if !strings.HasPrefix(strings.ToLower(cleanSource), strings.ToLower(cleanHome)) &&
				!strings.EqualFold(cleanSource, paths.CanonicalPath(homeDir)) {
				return fmt.Errorf("adapter %q returned source path outside home directory", d.Adapter.Name())
			}

			manifestItems = append(manifestItems, manifest.Item{
				Category:   item.Category,
				SourcePath: item.SourcePath,
				BackupPath: paths.Slash(filepath.Join(d.Adapter.Name(), item.RelPath)),
				Hash:       item.Hash,
				Size:       item.Size,
			})
			totalFiles++
			totalSize += item.Size
		}

		configDirCanonical := paths.ToCanonical(d.ConfigDir)
		m.AddAdapter(d.Adapter.Name(), "", configDirCanonical, manifestItems)
	}

	// 6. Generate .env.example if secrets were detected.
	if len(allSecretFiles) > 0 {
		if err := a.generateEnvExample(allSecretFiles, patterns, backupDir); err != nil {
			return fmt.Errorf("generate .env.example: %w", err)
		}
	}
	m.SecretsExcluded = len(allSecretFiles) > 0

	// 7. Save final manifest.
	if err := a.saveManifest(m, backupDir); err != nil {
		return fmt.Errorf("save manifest: %w", err)
	}

	cleanupOnError = false

	// 8. Report.
	infof(out, "Backup created: %s\n", backupID)
	infof(out, "  Preset:     %s\n", a.Preset)
	infof(out, "  Adapters:   %d\n", len(detected))
	infof(out, "  Files:      %d\n", m.FileCount)
	infof(out, "  Size:       %s\n", formatSize(m.TotalSize))
	infof(out, "  Location:   %s\n", backupDir)
	if m.SecretsExcluded {
		infof(out, "  ⚠ Secrets detected in %d file(s) — .env.example created\n", len(allSecretFiles))
	}

	return nil
}

// saveManifest serializes m as JSON and writes it via the injected FS.
func (a *BackupAction) saveManifest(m *manifest.Manifest, dir string) error {
	path := filepath.Join(dir, "manifest.json")
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	if err := a.FS.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}
	return nil
}

// scanBackupForSecrets walks the adapter backup directory and collects
// paths of files that contain secrets.
func (a *BackupAction) scanBackupForSecrets(adapterBackupDir string, patterns []*regexp.Regexp) []string {
	var secretFiles []string

	if err := a.FS.WalkDir(adapterBackupDir, func(fpath string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		results, scanErr := backup.ScanFile(fpath, patterns)
		if scanErr != nil {
			return scanErr
		}
		if len(results) > 0 {
			secretFiles = append(secretFiles, fpath)
		}
		return nil
	}); err != nil && a.Verbose {
		stderr := a.Stderr
		if stderr == nil {
			stderr = os.Stderr
		}
		warnf(stderr, "warning: secret scan walk: %v\n", err)
	}

	return secretFiles
}

// generateEnvExample produces a .env.example file from the detected
// secret files, replacing sensitive values with placeholders.
func (a *BackupAction) generateEnvExample(filePaths []string, patterns []*regexp.Regexp, outputDir string) error {
	return backup.GenerateEnvExample(filePaths, patterns, outputDir)
}

// formatSize returns a human-readable byte count.
func formatSize(bytes int64) string {
	return FormatSizeBytes(bytes)
}

// warnf writes a formatted warning to w. Write errors are silently
// discarded — warnings are non-critical diagnostics.
func warnf(w io.Writer, format string, args ...any) {
	fmt.Fprintf(w, format, args...) //nolint:errcheck
}

// infof writes a formatted info message to w. Write errors are silently
// discarded — info output is non-critical.
func infof(w io.Writer, format string, args ...any) {
	fmt.Fprintf(w, format, args...) //nolint:errcheck
}
