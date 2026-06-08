package actions

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"

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

	// Parameters (from CLI flags).
	Preset           string
	AdapterFilter    []string
	Verbose          bool
	BakVersion       string
	SecretPatterns   []*regexp.Regexp
	CustomCategories []string

	// HostnameFn returns the current hostname. Nil falls back to os.Hostname.
	HostnameFn HostnameFunc
}

// Run executes the backup workflow: resolve preset, detect adapters,
// copy files, scan secrets, and write manifest. All OS operations go
// through a.FS.
func (a *BackupAction) Run(cmd *cobra.Command, args []string) error {
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
			fmt.Fprintf(os.Stderr, "warning: could not get hostname: %v\n", err)
		}
	} else if h, err := os.Hostname(); err == nil {
		hostname = h
	} else if a.Verbose {
		fmt.Fprintf(os.Stderr, "warning: could not get hostname: %v\n", err)
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
				fmt.Fprintf(os.Stderr, "warning: cleanup failed: %v\n", err)
			}
		}
	}()

	patterns := a.SecretPatterns
	if patterns == nil {
		patterns = backup.DefaultPatterns()
	}

	var allSecretFiles []string

	for _, d := range detected {
		items, err := d.Adapter.ListItems(homeDir, categories)
		if err != nil {
			return fmt.Errorf("list items for %q: %w", d.Adapter.Name(), err)
		}

		if err := d.Adapter.Backup(homeDir, backupDir, items); err != nil {
			return fmt.Errorf("backup %q: %w", d.Adapter.Name(), err)
		}

		// Scan backed-up files for secrets.
		adapterBackupDir := filepath.Join(backupDir, d.Adapter.Name())
		secretFiles := a.scanBackupForSecrets(adapterBackupDir, patterns)
		allSecretFiles = append(allSecretFiles, secretFiles...)

		// Remove secret-bearing files.
		for _, sf := range secretFiles {
			if err := a.FS.RemoveAll(sf); err != nil && a.Verbose {
				fmt.Fprintf(os.Stderr, "warning: could not remove secret file: %v\n", err)
			}
		}

		// Build manifest items with path traversal validation.
		manifestItems := make([]manifest.Item, 0, len(items))
		for _, item := range items {
			if item.IsDir {
				continue
			}

			absSource := item.SourcePath
			if strings.HasPrefix(absSource, "~/") {
				absSource = paths.FromCanonical(absSource, homeDir)
			}
			cleanSource := path.Clean(strings.ReplaceAll(absSource, "\\", "/"))
			cleanHome := path.Clean(strings.ReplaceAll(homeDir, "\\", "/")) + "/"
			if !strings.HasPrefix(strings.ToLower(cleanSource), strings.ToLower(cleanHome)) &&
				!strings.EqualFold(cleanSource, path.Clean(strings.ReplaceAll(homeDir, "\\", "/"))) {
				return fmt.Errorf("adapter %q returned source path outside home directory", d.Adapter.Name())
			}

			manifestItems = append(manifestItems, manifest.Item{
				Category:   item.Category,
				SourcePath: item.SourcePath,
				BackupPath: strings.ReplaceAll(filepath.Join(d.Adapter.Name(), item.RelPath), "\\", "/"),
				Hash:       item.Hash,
				Size:       item.Size,
			})
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
	fmt.Printf("Backup created: %s\n", backupID)
	fmt.Printf("  Preset:     %s\n", a.Preset)
	fmt.Printf("  Adapters:   %d\n", len(detected))
	fmt.Printf("  Files:      %d\n", m.FileCount)
	fmt.Printf("  Size:       %s\n", formatSize(m.TotalSize))
	fmt.Printf("  Location:   %s\n", backupDir)
	if m.SecretsExcluded {
		fmt.Printf("  ⚠ Secrets detected in %d file(s) — .env.example created\n", len(allSecretFiles))
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
		fmt.Fprintf(os.Stderr, "warning: secret scan walk: %v\n", err)
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
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
