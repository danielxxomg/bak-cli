package actions

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"

	"github.com/danielxxomg/bak-cli/internal/adapters"
	"github.com/danielxxomg/bak-cli/internal/backup"
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
// copy files, scan secrets, and write manifest. It delegates the entire
// workflow to the canonical backup.Run in internal/backup so the CLI and
// TUI paths share one implementation, then prints a human-readable report.
func (a *BackupAction) Run() error {
	out := a.Stdout
	if out == nil {
		out = os.Stdout
	}
	errOut := a.Stderr
	if errOut == nil {
		errOut = os.Stderr
	}

	homeDir, err := a.FS.UserHomeDir()
	if err != nil {
		return fmt.Errorf("cannot determine home directory: %w", err)
	}

	result, err := backup.Run(backup.Context{
		FS:               a.FS,
		HomeDir:          homeDir,
		BakDir:           filepath.Join(homeDir, ".bak"),
		Registry:         a.Registry,
		Preset:           a.Preset,
		AdapterFilter:    a.AdapterFilter,
		BakVersion:       a.BakVersion,
		Verbose:          a.Verbose,
		SecretPatterns:   a.SecretPatterns,
		CustomCategories: a.CustomCategories,
		ProgressFn:       a.ProgressFn,
		ExcludesLoader:   a.ExcludesLoader,
		HostnameFn:       a.HostnameFn,
		Stderr:           errOut,
	})
	if err != nil {
		return err
	}

	a.report(out, result)
	return nil
}

// report prints the user-facing backup summary from a completed Result.
func (a *BackupAction) report(out io.Writer, r *backup.Result) {
	infof(out, "Backup created: %s\n", r.ID)
	infof(out, "  Preset:     %s\n", r.Preset)
	infof(out, "  Adapters:   %d\n", r.AdaptersRun)
	infof(out, "  Files:      %d\n", r.FileCount)
	infof(out, "  Size:       %s\n", formatSize(r.TotalSize))
	infof(out, "  Location:   %s\n", r.BackupDir)
	if r.SecretsExcluded {
		infof(out, "  ⚠ Secrets detected in %d file(s) — .env.example created\n", r.Secrets)
	}
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
