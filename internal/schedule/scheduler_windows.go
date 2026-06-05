//go:build windows

package schedule

import (
	"bytes"
	"fmt"
	"strings"
)

// NewScheduler returns the platform-appropriate Scheduler implementation.
func NewScheduler() Scheduler {
	return &SchtasksScheduler{}
}

// Create adds a schtasks scheduled task for the given profile.
func (s *SchtasksScheduler) Create(profile string, interval string) error {
	args := buildSchtasksCreateArgs(profile, interval)
	cmd := execCommand("schtasks", args...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("schtasks create: %w (stderr: %s)", err, stderr.String())
	}
	return nil
}

// Remove deletes the schtasks task for the given profile.
func (s *SchtasksScheduler) Remove(profile string) error {
	args := buildSchtasksDeleteArgs(profile)
	cmd := execCommand("schtasks", args...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("schtasks delete: %w (stderr: %s)", err, stderr.String())
	}
	return nil
}

// List returns schtasks entries created by bak-cli.
func (s *SchtasksScheduler) List() ([]ScheduleEntry, error) {
	args := buildSchtasksQueryArgs()
	cmd := execCommand("schtasks", args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("schtasks query: %w (stderr: %s)", err, stderr.String())
	}

	return parseSchtasksCSV(stdout.String()), nil
}

// parseSchtasksCSV extracts bak-cli entries from "schtasks /query /fo csv /nh" output.
// CSV format: "TaskName","Next Run Time","Status"
func parseSchtasksCSV(output string) []ScheduleEntry {
	var entries []ScheduleEntry
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Expect format: "bak-cli-work","...","..."
		if !strings.Contains(line, "bak-cli-") {
			continue
		}
		// Extract profile name from task name.
		parts := strings.Split(line, ",")
		if len(parts) < 1 {
			continue
		}
		taskName := strings.Trim(parts[0], `"`)
		profile := strings.TrimPrefix(taskName, "bak-cli-")

		entries = append(entries, ScheduleEntry{
			Profile:  profile,
			Interval: "", // interval not extractable from schtasks CSV alone
			Raw:      line,
		})
	}
	return entries
}
