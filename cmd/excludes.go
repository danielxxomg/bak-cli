package cmd

import (
	"github.com/danielxxomg/bak-cli/internal/adapters"
	"github.com/danielxxomg/bak-cli/internal/config"
	"github.com/danielxxomg/bak-cli/internal/paths"
)

// loadExcludes resolves the scan filtering options (exclude patterns and
// max file size) from the persisted config and the ~/.config/bak/ignore
// file. It is shared by the backup command wiring in backup.go and root.go
// as the ExcludesLoader for BackupAction/backup.Engine.
func loadExcludes() (adapters.ScanOptions, error) {
	cfg, err := config.Load()
	if err != nil {
		return adapters.ScanOptions{}, err
	}
	cfgDir, err := paths.ConfigDir("bak")
	if err != nil {
		return adapters.ScanOptions{}, err
	}
	patterns, maxSize, err := config.LoadExcludes(cfgDir, cfg.Settings)
	if err != nil {
		return adapters.ScanOptions{}, err
	}
	return adapters.ScanOptions{
		Excludes:    patterns,
		MaxFileSize: maxSize,
	}, nil
}
