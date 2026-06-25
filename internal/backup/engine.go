package backup

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"

	"github.com/danielxxomg/bak-cli/internal/adapters"
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

// ProgressFn is a callback invoked once per file during backup.
type ProgressFn func(file string, done, total int)

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
	ProgressFn       ProgressFn         // optional progress callback

	// ExcludesLoader returns scan options to filter files during backup.
	// nil means no exclusions (current behavior). Wired by cmd/ to
	// config.Load + config.LoadExcludes.
	ExcludesLoader func() (adapters.ScanOptions, error)

	// FS optionally injects a filesystem. nil falls back to osFS so existing
	// &Engine{...} instantiations keep working unchanged.
	FS FS

	// Stderr optionally injects a writer for verbose warnings. nil falls
	// back to os.Stderr.
	Stderr io.Writer
}

// Result summarizes a completed backup operation.
type Result struct {
	ID              string // backup ID (timestamp)
	BackupDir       string // full path to backup directory
	FileCount       int    // total files backed up
	TotalSize       int64  // total bytes
	Secrets         int    // number of secret-bearing files detected
	SecretsExcluded bool   // true when at least one secret was detected and excluded
	AdaptersRun     int    // number of adapters that contributed
	Preset          string // preset used for this backup (for reporting)
}

// Run executes the full backup flow and returns a summary. It delegates to
// the canonical backup.Run in workflow.go so the TUI path shares the exact
// same implementation as the CLI path (BackupAction.Run). The injected FS
// and Stderr fields default to osFS and os.Stderr when nil.
func (e *Engine) Run() (*Result, error) {
	return Run(Context{
		FS:               e.FS,
		HomeDir:          e.HomeDir,
		BakDir:           e.BakDir,
		Registry:         e.Registry,
		Preset:           e.Preset,
		AdapterFilter:    e.AdapterFilter,
		BakVersion:       e.BakVersion,
		Verbose:          e.Verbose,
		SecretPatterns:   e.SecretPatterns,
		CustomCategories: e.CustomCategories,
		ProgressFn:       e.ProgressFn,
		ExcludesLoader:   e.ExcludesLoader,
		Stderr:           e.Stderr,
	})
}
