// Package schedule provides OS-native backup scheduling via crontab (Unix)
// or schtasks (Windows). Platform-specific implementations are selected at
// compile time using build tags.
package schedule

import (
	"os/exec"
	"strings"
)

// execCommand is a package-level variable so tests can replace it with a mock.
// Default points to exec.Command.
var execCommand = exec.Command

// Scheduler manages OS-native scheduled tasks for bak-cli backup profiles.
// Each implementation (CronScheduler, SchtasksScheduler) handles the
// platform-specific commands and output parsing.
type Scheduler interface {
	// Create adds a scheduled task for the given profile at the specified
	// interval. The interval must be one of ValidIntervals().
	Create(profile string, interval string) error

	// Remove deletes the scheduled task for the given profile.
	Remove(profile string) error

	// List returns all bak-cli-tagged scheduled entries currently active.
	List() ([]ScheduleEntry, error)
}

// ScheduleEntry represents a single active scheduled backup.
type ScheduleEntry struct {
	Profile  string // profile name extracted from the scheduler line
	Interval string // detected interval (daily, weekly, etc.)
	Raw      string // original scheduler line (cron line or schtasks output)
}

// ValidIntervals returns the list of supported scheduling intervals.
func ValidIntervals() []string {
	return []string{scheduleDaily, scheduleWeekly, scheduleEvery12h, scheduleEvery6h}
}

// IsValidInterval reports whether the given string is a supported interval.
func IsValidInterval(interval string) bool {
	for _, iv := range ValidIntervals() {
		if iv == interval {
			return true
		}
	}
	return false
}

// CronScheduler implements Scheduler using crontab on Unix-like systems.
// Build constraint: !windows (see scheduler_crontab.go).
type CronScheduler struct{}

// SchtasksScheduler implements Scheduler using schtasks.exe on Windows.
// Build constraint: windows (see scheduler_windows.go).
type SchtasksScheduler struct{}

// --- Cross-platform helpers ---

// formatCronLine builds a crontab line for the given profile and interval.
// Format: "{cron_expr} bak backup --profile {name} && bak push --profile {name} # bak-cli:{name}"
func formatCronLine(profile string, interval string) string {
	var cronExpr string
	switch interval {
	case scheduleDaily:
		cronExpr = cronDailyAt2AM
	case scheduleWeekly:
		cronExpr = "0 3 * * 0"
	case scheduleEvery12h:
		cronExpr = "0 */12 * * *"
	case scheduleEvery6h:
		cronExpr = "0 */6 * * *"
	default:
		cronExpr = cronDailyAt2AM
	}
	cmd := "bak backup --profile " + profile + " && bak push --profile " + profile
	return cronExpr + " " + cmd + " # bak-cli:" + profile
}

// parseCronLine extracts a ScheduleEntry from a crontab line if it is a
// bak-cli tagged entry. Returns (entry, true) on success, (_, false) otherwise.
func parseCronLine(line string) (ScheduleEntry, bool) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return ScheduleEntry{}, false
	}

	// Must contain a bak-cli tag.
	tagIdx := strings.Index(line, "# bak-cli")
	if tagIdx == -1 {
		return ScheduleEntry{}, false
	}

	// Extract cron prefix (first 5 fields) for interval detection.
	fields := strings.Fields(line)
	if len(fields) < 7 {
		// Need at least 5 cron fields + 1 command word + tag parts.
		return ScheduleEntry{}, false
	}

	// The sixth field must NOT be the tag itself (need a real command).
	sixth := fields[5]
	if strings.HasPrefix(sixth, "#") {
		return ScheduleEntry{}, false
	}

	cronPrefix := strings.Join(fields[:5], " ")

	// Extract profile name from the tag.
	tagPart := strings.TrimSpace(line[tagIdx:])
	profile := ""
	if strings.HasPrefix(tagPart, "# bak-cli:") {
		profile = strings.TrimPrefix(tagPart, "# bak-cli:")
	} else {
		profile = strings.TrimPrefix(tagPart, "# bak-cli")
	}
	profile = strings.TrimSpace(profile)

	return ScheduleEntry{
		Profile:  profile,
		Interval: intervalFromCron(cronPrefix),
		Raw:      line,
	}, true
}

// intervalFromCron maps a cron expression prefix to a named interval.
func intervalFromCron(cronPrefix string) string {
	switch cronPrefix {
	case cronDailyAt2AM:
		return scheduleDaily
	case "0 3 * * 0":
		return scheduleWeekly
	case "0 */12 * * *":
		return scheduleEvery12h
	case "0 */6 * * *":
		return scheduleEvery6h
	default:
		return ""
	}
}

// --- Schtasks helpers (cross-platform — used by scheduler_windows.go) ---

// buildSchtasksCreateArgs returns the arguments for "schtasks /create".
func buildSchtasksCreateArgs(profile string, interval string) []string {
	sc, mo, st := intervalToSchtasksParams(interval)
	taskName := "bak-cli-" + profile
	cmd := "bak backup --profile " + profile + " && bak push --profile " + profile

	args := []string{"/create", "/tn", taskName, "/tr", cmd, "/sc", sc, "/st", st}
	if mo != "" {
		args = append(args, "/mo", mo)
	}
	args = append(args, "/f") // force overwrite
	return args
}

// buildSchtasksDeleteArgs returns the arguments for "schtasks /delete".
func buildSchtasksDeleteArgs(profile string) []string {
	taskName := "bak-cli-" + profile
	return []string{"/delete", "/tn", taskName, "/f"}
}

// buildSchtasksQueryArgs returns the arguments for "schtasks /query".
func buildSchtasksQueryArgs() []string {
	return []string{"/query", "/fo", "csv", "/nh"}
}

// intervalToSchtasksParams maps a named interval to schtasks /sc, /mo, /st values.
func intervalToSchtasksParams(interval string) (sc, mo, st string) {
	switch interval {
	case scheduleDaily:
		return scheduleDaily, "", "02:00"
	case scheduleWeekly:
		return scheduleWeekly, "", "03:00"
	case scheduleEvery12h:
		return "hourly", "12", "00:00"
	case scheduleEvery6h:
		return "hourly", "6", "00:00"
	default:
		return scheduleDaily, "", "02:00"
	}
}
