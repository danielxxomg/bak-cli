package cmd

import (
	"fmt"
	"os"

	"github.com/danielxxomg/bak-cli/internal/adapters"
	"github.com/danielxxomg/bak-cli/internal/adapters/register"
	"github.com/danielxxomg/bak-cli/internal/backup"
	"github.com/spf13/cobra"
)

var backupPreset string
var backupAdapter string

// backupCmd represents the backup command.
var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Create a backup of your AI coding setup",
	Long: `Scans for installed AI coding tools (currently OpenCode), resolves the
requested preset, copies configuration files to ~/.bak/backups/<id>/,
detects and redacts secrets, and writes a manifest.

Examples:
  bak backup                    # quick backup (config files only)
  bak backup --preset full      # everything: skills, commands, plugins, agents, config
  bak backup --preset skills    # skills only
  bak backup --adapter opencode # force a specific adapter`,
	RunE: runBackup,
}

func init() {
	backupCmd.Flags().StringVarP(&backupPreset, "preset", "p", "quick",
		"backup preset: quick, full, or skills")
	backupCmd.Flags().StringVarP(&backupAdapter, "adapter", "a", "",
		"run only the named adapter (default: all detected)")

	rootCmd.AddCommand(backupCmd)
}

func runBackup(cmd *cobra.Command, args []string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("cannot determine home directory: %w", err)
	}

	bakDir, err := backup.BakDir()
	if err != nil {
		return fmt.Errorf("bak dir: %w", err)
	}

	// --- Wire adapters ----------------------------------------------------
	reg := adapters.NewRegistry()
	if err := register.All(reg); err != nil {
		return fmt.Errorf("register adapters: %w", err)
	}

	// --- Build and run engine ---------------------------------------------
	engine := &backup.Engine{
		HomeDir:       homeDir,
		BakDir:        bakDir,
		Registry:      reg,
		Preset:        backupPreset,
		AdapterFilter: backupAdapter,
		BakVersion:    Version,
		Verbose:       verbose,
	}

	result, err := engine.Run()
	if err != nil {
		return err
	}

	// --- Report -----------------------------------------------------------
	fmt.Printf("Backup created: %s\n", result.ID)
	fmt.Printf("  Preset:     %s\n", backupPreset)
	fmt.Printf("  Adapters:   %d\n", result.AdaptersRun)
	fmt.Printf("  Files:      %d\n", result.FileCount)
	fmt.Printf("  Size:       %s\n", formatSize(result.TotalSize))
	fmt.Printf("  Location:   %s\n", result.BackupDir)

	if result.Secrets > 0 {
		fmt.Printf("  ⚠ Secrets detected in %d file(s) — .env.example created\n", result.Secrets)
	}

	return nil
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
