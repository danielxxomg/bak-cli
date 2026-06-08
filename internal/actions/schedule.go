package actions

import (
	"fmt"
	"io"
	"strings"

	"github.com/danielxxomg/bak-cli/internal/config"
	"github.com/danielxxomg/bak-cli/internal/schedule"
)

// ScheduleAction handles schedule create/list/remove operations.
type ScheduleAction struct {
	// ConfigLoader returns the current config.
	ConfigLoader func() (*config.Config, error)

	// Stdout receives user-facing output.
	Stdout io.Writer

	// Stderr receives warning messages.
	Stderr io.Writer

	// NewScheduler creates a scheduler instance. Defaults to schedule.NewScheduler.
	NewScheduler func() schedule.Scheduler
}

// sched returns the scheduler, falling back to the production default.
func (a *ScheduleAction) sched() schedule.Scheduler {
	if a.NewScheduler != nil {
		return a.NewScheduler()
	}
	return schedule.NewScheduler()
}

// Create creates a scheduled backup for the given profile and interval.
func (a *ScheduleAction) Create(profile, interval string) error {
	// Validate interval.
	if !schedule.IsValidInterval(interval) {
		valid := schedule.ValidIntervals()
		return fmt.Errorf("invalid interval %q (valid: %s)", interval, strings.Join(valid, ", "))
	}

	// Load config and validate profile exists.
	cfg, err := a.ConfigLoader()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	pc, ok := cfg.Profiles[profile]
	if !ok {
		return fmt.Errorf("profile %q not found — use 'bak profile list' to see configured profiles", profile)
	}

	// Create the scheduled task.
	s := a.sched()
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
		_, _ = fmt.Fprintf(a.Stderr, "warning: schedule created but config save failed: %v\n", err)
	}

	_, _ = fmt.Fprintf(a.Stdout, "Schedule created for profile %q (interval: %s)\n", profile, interval)
	return nil
}

// List displays all active bak-cli backup schedules.
func (a *ScheduleAction) List() error {
	s := a.sched()
	entries, err := s.List()
	if err != nil {
		return fmt.Errorf("schedule list: %w", err)
	}

	if len(entries) == 0 {
		_, _ = fmt.Fprintln(a.Stdout, "No bak-cli schedules found.")
		return nil
	}

	_, _ = fmt.Fprintf(a.Stdout, "%-20s %-15s\n", "PROFILE", "INTERVAL")
	_, _ = fmt.Fprintln(a.Stdout, strings.Repeat("-", 40))

	for _, e := range entries {
		interval := e.Interval
		if interval == "" {
			interval = "—"
		}
		_, _ = fmt.Fprintf(a.Stdout, "%-20s %-15s\n", e.Profile, interval)
	}

	return nil
}

// Remove deletes a scheduled backup and clears its profile config.
func (a *ScheduleAction) Remove(profile string) error {
	s := a.sched()
	if err := s.Remove(profile); err != nil {
		return fmt.Errorf("schedule remove: %w", err)
	}

	cfg, err := a.ConfigLoader()
	if err != nil {
		_, _ = fmt.Fprintf(a.Stderr, "warning: schedule removed but config load failed: %v\n", err)
	} else {
		if pc, ok := cfg.Profiles[profile]; ok {
			pc.Schedule = nil
			cfg.Profiles[profile] = pc
			if err := cfg.Save(); err != nil {
				_, _ = fmt.Fprintf(a.Stderr, "warning: schedule removed but config save failed: %v\n", err)
			}
		}
	}

	_, _ = fmt.Fprintf(a.Stdout, "Schedule removed for profile %q.\n", profile)
	return nil
}
