package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/danielxxomg/bak-cli/internal/config"
	"github.com/danielxxomg/bak-cli/internal/schedule"
	"github.com/spf13/cobra"
)

// scheduleCmd is the parent command for backup scheduling.
var scheduleCmd = &cobra.Command{
	Use:   "schedule",
	Short: "Manage OS-native backup schedules",
	Long: `Schedule automatic bak-cli backups using OS-native task schedulers.

On Linux/macOS, schedules are managed via crontab.
On Windows, schedules are managed via schtasks.

Examples:
  bak schedule create work --every daily
  bak schedule list
  bak schedule remove work`,
}

// --- create ---

var scheduleCreateEvery string

var scheduleCreateCmd = &cobra.Command{
	Use:   "create <profile>",
	Short: "Create a scheduled backup for a profile",
	Long: `Create an OS-native scheduled task that runs 'bak backup' and 'bak push'
for the specified profile at the given interval.

Supported intervals: daily, weekly, every-12h, every-6h

Examples:
  bak schedule create work --every daily
  bak schedule create home --every weekly`,
	Args: cobra.ExactArgs(1),
	RunE: runScheduleCreate,
}

func init() {
	scheduleCreateCmd.Flags().StringVar(&scheduleCreateEvery, "every", "",
		"scheduling interval: daily, weekly, every-12h, every-6h (required)")
	scheduleCreateCmd.MarkFlagRequired("every")

	scheduleCmd.AddCommand(scheduleCreateCmd)
	rootCmd.AddCommand(scheduleCmd)
}

func runScheduleCreate(cmd *cobra.Command, args []string) error {
	profile := args[0]
	interval := scheduleCreateEvery
	out := cmd.OutOrStdout()

	// Validate interval.
	if !schedule.IsValidInterval(interval) {
		valid := schedule.ValidIntervals()
		return fmt.Errorf("invalid interval %q (valid: %s)", interval, strings.Join(valid, ", "))
	}

	// Load config and validate profile exists.
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	pc, ok := cfg.Profiles[profile]
	if !ok {
		return fmt.Errorf("profile %q not found — use 'bak profile list' to see configured profiles", profile)
	}

	// Create the scheduled task.
	s := schedule.NewScheduler()
	if err := s.Create(profile, interval); err != nil {
		return fmt.Errorf("schedule create: %w", err)
	}

	// Update profile config.
	pc.Schedule = &config.ScheduleConfig{
		Enabled:  true,
		Interval: interval,
	}
	cfg.Profiles[profile] = pc

	if err := cfg.Save(); err != nil {
		// Warn but don't fail — the schedule is already created.
		fmt.Fprintf(os.Stderr, "warning: schedule created but config save failed: %v\n", err)
	}

	fmt.Fprintf(out, "Schedule created for profile %q (interval: %s)\n", profile, interval)
	return nil
}

// --- list ---

var scheduleListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all bak-cli scheduled backups",
	Long:  "Display a table of all active bak-cli backup schedules.",
	Args:  cobra.NoArgs,
	RunE:  runScheduleList,
}

func init() {
	scheduleCmd.AddCommand(scheduleListCmd)
}

func runScheduleList(cmd *cobra.Command, args []string) error {
	out := cmd.OutOrStdout()

	s := schedule.NewScheduler()
	entries, err := s.List()
	if err != nil {
		return fmt.Errorf("schedule list: %w", err)
	}

	if len(entries) == 0 {
		fmt.Fprintln(out, "No bak-cli schedules found.")
		return nil
	}

	// Header.
	fmt.Fprintf(out, "%-20s %-15s\n", "PROFILE", "INTERVAL")
	fmt.Fprintln(out, strings.Repeat("-", 40))

	for _, e := range entries {
		interval := e.Interval
		if interval == "" {
			interval = "—"
		}
		fmt.Fprintf(out, "%-20s %-15s\n", e.Profile, interval)
	}

	return nil
}

// --- remove ---

var scheduleRemoveCmd = &cobra.Command{
	Use:   "remove <profile>",
	Short: "Remove a scheduled backup for a profile",
	Long: `Delete the OS-native scheduled task for the given profile and clear
its schedule configuration.

Examples:
  bak schedule remove work`,
	Args: cobra.ExactArgs(1),
	RunE: runScheduleRemove,
}

func init() {
	scheduleCmd.AddCommand(scheduleRemoveCmd)
}

func runScheduleRemove(cmd *cobra.Command, args []string) error {
	profile := args[0]
	out := cmd.OutOrStdout()

	// Remove the scheduled task.
	s := schedule.NewScheduler()
	if err := s.Remove(profile); err != nil {
		return fmt.Errorf("schedule remove: %w", err)
	}

	// Clear profile schedule config.
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: schedule removed but config load failed: %v\n", err)
	} else {
		if pc, ok := cfg.Profiles[profile]; ok {
			pc.Schedule = nil
			cfg.Profiles[profile] = pc
			if err := cfg.Save(); err != nil {
				fmt.Fprintf(os.Stderr, "warning: schedule removed but config save failed: %v\n", err)
			}
		}
	}

	fmt.Fprintf(out, "Schedule removed for profile %q.\n", profile)
	return nil
}
