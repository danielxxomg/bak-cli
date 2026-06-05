//go:build !windows

package schedule

import (
	"bytes"
	"fmt"
	"strings"
)

// NewScheduler returns the platform-appropriate Scheduler implementation.
func NewScheduler() Scheduler {
	return &CronScheduler{}
}

// Create adds a crontab entry for the given profile at the specified interval.
// It reads the current crontab, appends a tagged line, and pipes the result
// back to crontab.
func (s *CronScheduler) Create(profile string, interval string) error {
	// Read current crontab (ignore errors — crontab may not exist yet).
	current, err := readCrontab()
	if err != nil {
		// If user has no crontab, start fresh.
		current = ""
	}

	// Build the new entry line.
	newLine := formatCronLine(profile, interval)

	// Remove any existing entry for this profile to avoid duplicates.
	var lines []string
	if current != "" {
		for _, line := range strings.Split(strings.TrimSpace(current), "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			// Skip existing entry for the same profile.
			if strings.Contains(line, "# bak-cli:"+profile) || strings.Contains(line, "# bak-cli "+profile) {
				continue
			}
			lines = append(lines, line)
		}
	}

	// Append the new entry.
	lines = append(lines, newLine)

	// Write back.
	return writeCrontab(strings.Join(lines, "\n") + "\n")
}

// Remove deletes the crontab entry for the given profile.
func (s *CronScheduler) Remove(profile string) error {
	current, err := readCrontab()
	if err != nil {
		// No crontab, nothing to remove.
		return nil
	}

	var lines []string
	found := false
	for _, line := range strings.Split(strings.TrimSpace(current), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.Contains(line, "# bak-cli:"+profile) || strings.Contains(line, "# bak-cli "+profile) {
			found = true
			continue // skip this line
		}
		lines = append(lines, line)
	}

	if !found {
		return fmt.Errorf("no schedule found for profile %q", profile)
	}

	return writeCrontab(strings.Join(lines, "\n") + "\n")
}

// List returns crontab entries tagged with # bak-cli.
func (s *CronScheduler) List() ([]ScheduleEntry, error) {
	current, err := readCrontab()
	if err != nil {
		// No crontab = no entries.
		return nil, nil
	}

	var entries []ScheduleEntry
	for _, line := range strings.Split(current, "\n") {
		line = strings.TrimSpace(line)
		if entry, ok := parseCronLine(line); ok {
			entries = append(entries, entry)
		}
	}
	return entries, nil
}

// readCrontab executes "crontab -l" and returns its stdout as a string.
// Returns an error if the user has no crontab or the command fails.
func readCrontab() (string, error) {
	cmd := execCommand("crontab", "-l")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("crontab -l: %w", err)
	}
	return stdout.String(), nil
}

// writeCrontab pipes the given content to "crontab -".
func writeCrontab(content string) error {
	cmd := execCommand("crontab", "-")
	cmd.Stdin = strings.NewReader(content)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("crontab -: %w (stderr: %s)", err, stderr.String())
	}
	return nil
}
