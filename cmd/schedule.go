package cmd

import (
	"github.com/spf13/cobra"

	"github.com/danielxxomg/bak-cli/internal/actions"
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
	if err := scheduleCreateCmd.MarkFlagRequired("every"); err != nil {
		panic("mark 'every' required: " + err.Error())
	}

	scheduleCmd.AddCommand(scheduleCreateCmd)
	rootCmd.AddCommand(scheduleCmd)
}

func runScheduleCreate(cmd *cobra.Command, args []string) error {
	return runScheduleCreateWithDeps(cmd, args, depsFromCmd(cmd))
}

func runScheduleCreateWithDeps(cmd *cobra.Command, args []string, deps cmdDeps) error {
	action := &actions.ScheduleAction{
		ConfigLoader: deps.ConfigLoader,
		Stdout:       deps.Stdout,
		Stderr:       deps.Stderr,
		NewScheduler: deps.NewScheduler,
	}
	return action.Create(args[0], scheduleCreateEvery)
}

// --- list ---

var scheduleListCmd = &cobra.Command{
	Use:   listAction,
	Short: "List all bak-cli scheduled backups",
	Long:  "Display a table of all active bak-cli backup schedules.",
	Args:  cobra.NoArgs,
	RunE:  runScheduleList,
}

func init() {
	scheduleCmd.AddCommand(scheduleListCmd)
}

func runScheduleList(cmd *cobra.Command, args []string) error {
	return runScheduleListWithDeps(cmd, args, depsFromCmd(cmd))
}

func runScheduleListWithDeps(cmd *cobra.Command, args []string, deps cmdDeps) error {
	action := &actions.ScheduleAction{
		Stdout:       deps.Stdout,
		NewScheduler: deps.NewScheduler,
	}
	return action.List()
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
	return runScheduleRemoveWithDeps(cmd, args, depsFromCmd(cmd))
}

func runScheduleRemoveWithDeps(cmd *cobra.Command, args []string, deps cmdDeps) error {
	action := &actions.ScheduleAction{
		ConfigLoader: deps.ConfigLoader,
		Stdout:       deps.Stdout,
		Stderr:       deps.Stderr,
		NewScheduler: deps.NewScheduler,
	}
	return action.Remove(args[0])
}
