package actions

import (
	"errors"
	"strings"
	"testing"

	"github.com/danielxxomg/bak-cli/internal/config"
	"github.com/danielxxomg/bak-cli/internal/schedule"
)

// mockScheduler implements schedule.Scheduler for testing.
type mockScheduler struct {
	createErr error
	removeErr error
	listErr   error
	entries   []schedule.ScheduleEntry

	createCalls []struct {
		profile  string
		interval string
	}
	removeCalls []string
}

var _ schedule.Scheduler = (*mockScheduler)(nil)

func (m *mockScheduler) Create(profile, interval string) error {
	m.createCalls = append(m.createCalls, struct {
		profile  string
		interval string
	}{profile, interval})
	return m.createErr
}

func (m *mockScheduler) Remove(profile string) error {
	m.removeCalls = append(m.removeCalls, profile)
	return m.removeErr
}

func (m *mockScheduler) List() ([]schedule.ScheduleEntry, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.entries, nil
}

// --- ScheduleAction.Create tests ---

func TestScheduleCreate_Success(t *testing.T) {
	sched := &mockScheduler{}
	var out, errOut strings.Builder

	cfg := &config.Config{
		SchemaVersion: "0.3.0",
		Profiles: map[string]config.ProfileConfig{
			"work": {Provider: "github-gist", Preset: "quick"},
		},
	}

	action := &ScheduleAction{
		ConfigLoader: func() (*config.Config, error) {
			return cfg, nil
		},
		Stdout:  &out,
		Stderr:  &errOut,
		NewScheduler: func() schedule.Scheduler {
			return sched
		},
	}

	err := action.Create("work", "daily")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if len(sched.createCalls) != 1 {
		t.Fatalf("expected 1 Create call, got %d", len(sched.createCalls))
	}
	if sched.createCalls[0].profile != "work" {
		t.Errorf("Create profile = %q, want work", sched.createCalls[0].profile)
	}
	if sched.createCalls[0].interval != "daily" {
		t.Errorf("Create interval = %q, want daily", sched.createCalls[0].interval)
	}

	output := out.String()
	if !strings.Contains(output, "Schedule created") {
		t.Errorf("output should confirm creation: %q", output)
	}
	if !strings.Contains(output, "work") {
		t.Errorf("output should mention profile: %q", output)
	}
}

func TestScheduleCreate_InvalidInterval(t *testing.T) {
	sched := &mockScheduler{}
	var out, errOut strings.Builder

	cfg := &config.Config{
		SchemaVersion: "0.3.0",
		Profiles: map[string]config.ProfileConfig{
			"work": {Provider: "github-gist"},
		},
	}

	action := &ScheduleAction{
		ConfigLoader: func() (*config.Config, error) {
			return cfg, nil
		},
		Stdout: &out,
		Stderr: &errOut,
		NewScheduler: func() schedule.Scheduler {
			return sched
		},
	}

	err := action.Create("work", "minutely")
	if err == nil {
		t.Fatal("expected error for invalid interval")
	}
	if !strings.Contains(err.Error(), "invalid interval") {
		t.Errorf("error should mention invalid interval: %v", err)
	}
	if len(sched.createCalls) != 0 {
		t.Error("Create should not be called with invalid interval")
	}
}

func TestScheduleCreate_ProfileNotFound(t *testing.T) {
	sched := &mockScheduler{}
	var out, errOut strings.Builder

	cfg := &config.Config{
		SchemaVersion: "0.3.0",
		Profiles: map[string]config.ProfileConfig{
			"work": {Provider: "github-gist"},
		},
	}

	action := &ScheduleAction{
		ConfigLoader: func() (*config.Config, error) {
			return cfg, nil
		},
		Stdout: &out,
		Stderr: &errOut,
		NewScheduler: func() schedule.Scheduler {
			return sched
		},
	}

	err := action.Create("nonexistent", "daily")
	if err == nil {
		t.Fatal("expected error for missing profile")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention not found: %v", err)
	}
}

func TestScheduleCreate_ConfigLoadError(t *testing.T) {
	sched := &mockScheduler{}
	var out, errOut strings.Builder

	action := &ScheduleAction{
		ConfigLoader: func() (*config.Config, error) {
			return nil, errors.New("config disk error")
		},
		Stdout: &out,
		Stderr: &errOut,
		NewScheduler: func() schedule.Scheduler {
			return sched
		},
	}

	err := action.Create("work", "daily")
	if err == nil {
		t.Fatal("expected error when config fails")
	}
	if !strings.Contains(err.Error(), "load config") {
		t.Errorf("error should mention load config: %v", err)
	}
}

func TestScheduleCreate_SchedulerError(t *testing.T) {
	sched := &mockScheduler{
		createErr: errors.New("crontab permission denied"),
	}
	var out, errOut strings.Builder

	cfg := &config.Config{
		SchemaVersion: "0.3.0",
		Profiles: map[string]config.ProfileConfig{
			"work": {Provider: "github-gist"},
		},
	}

	action := &ScheduleAction{
		ConfigLoader: func() (*config.Config, error) {
			return cfg, nil
		},
		Stdout: &out,
		Stderr: &errOut,
		NewScheduler: func() schedule.Scheduler {
			return sched
		},
	}

	err := action.Create("work", "daily")
	if err == nil {
		t.Fatal("expected error from scheduler")
	}
	if !strings.Contains(err.Error(), "schedule create") {
		t.Errorf("error should mention schedule create: %v", err)
	}
}

// --- ScheduleAction.List tests ---

func TestScheduleList_Success(t *testing.T) {
	sched := &mockScheduler{
		entries: []schedule.ScheduleEntry{
			{Profile: "work", Interval: "daily"},
			{Profile: "home", Interval: "weekly"},
		},
	}
	var out, errOut strings.Builder

	action := &ScheduleAction{
		ConfigLoader: func() (*config.Config, error) {
			return &config.Config{SchemaVersion: "0.3.0"}, nil
		},
		Stdout: &out,
		Stderr: &errOut,
		NewScheduler: func() schedule.Scheduler {
			return sched
		},
	}

	err := action.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "work") {
		t.Errorf("output should contain work profile: %q", output)
	}
	if !strings.Contains(output, "home") {
		t.Errorf("output should contain home profile: %q", output)
	}
	if !strings.Contains(output, "daily") {
		t.Errorf("output should contain daily interval: %q", output)
	}
}

func TestScheduleList_Empty(t *testing.T) {
	sched := &mockScheduler{
		entries: nil,
	}
	var out, errOut strings.Builder

	action := &ScheduleAction{
		ConfigLoader: func() (*config.Config, error) {
			return &config.Config{SchemaVersion: "0.3.0"}, nil
		},
		Stdout: &out,
		Stderr: &errOut,
		NewScheduler: func() schedule.Scheduler {
			return sched
		},
	}

	err := action.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "No bak-cli schedules") {
		t.Errorf("output should indicate no schedules: %q", output)
	}
}

func TestScheduleList_SchedulerError(t *testing.T) {
	sched := &mockScheduler{
		listErr: errors.New("crontab read error"),
	}
	var out, errOut strings.Builder

	action := &ScheduleAction{
		ConfigLoader: func() (*config.Config, error) {
			return &config.Config{SchemaVersion: "0.3.0"}, nil
		},
		Stdout: &out,
		Stderr: &errOut,
		NewScheduler: func() schedule.Scheduler {
			return sched
		},
	}

	err := action.List()
	if err == nil {
		t.Fatal("expected error from scheduler list")
	}
	if !strings.Contains(err.Error(), "schedule list") {
		t.Errorf("error should mention schedule list: %v", err)
	}
}

// --- ScheduleAction.Remove tests ---

func TestScheduleRemove_Success(t *testing.T) {
	sched := &mockScheduler{}
	var out, errOut strings.Builder

	cfg := &config.Config{
		SchemaVersion: "0.3.0",
		Profiles: map[string]config.ProfileConfig{
			"work": {Provider: "github-gist", Preset: "quick"},
		},
	}

	action := &ScheduleAction{
		ConfigLoader: func() (*config.Config, error) {
			return cfg, nil
		},
		Stdout: &out,
		Stderr: &errOut,
		NewScheduler: func() schedule.Scheduler {
			return sched
		},
	}

	err := action.Remove("work")
	if err != nil {
		t.Fatalf("Remove: %v", err)
	}

	if len(sched.removeCalls) != 1 {
		t.Fatalf("expected 1 Remove call, got %d", len(sched.removeCalls))
	}
	if sched.removeCalls[0] != "work" {
		t.Errorf("Remove profile = %q, want work", sched.removeCalls[0])
	}

	output := out.String()
	if !strings.Contains(output, "Schedule removed") {
		t.Errorf("output should confirm removal: %q", output)
	}
}

func TestScheduleRemove_SchedulerError(t *testing.T) {
	sched := &mockScheduler{
		removeErr: errors.New("crontab write error"),
	}
	var out, errOut strings.Builder

	action := &ScheduleAction{
		ConfigLoader: func() (*config.Config, error) {
			return &config.Config{SchemaVersion: "0.3.0"}, nil
		},
		Stdout: &out,
		Stderr: &errOut,
		NewScheduler: func() schedule.Scheduler {
			return sched
		},
	}

	err := action.Remove("work")
	if err == nil {
		t.Fatal("expected error from scheduler remove")
	}
	if !strings.Contains(err.Error(), "schedule remove") {
		t.Errorf("error should mention schedule remove: %v", err)
	}
}

func TestScheduleRemove_ConfigLoadWarning(t *testing.T) {
	// Remove should still succeed even if config reload fails.
	// The warning goes to stderr, not the return error.
	sched := &mockScheduler{}
	var out, errOut strings.Builder

	action := &ScheduleAction{
		ConfigLoader: func() (*config.Config, error) {
			return nil, errors.New("config gone")
		},
		Stdout: &out,
		Stderr: &errOut,
		NewScheduler: func() schedule.Scheduler {
			return sched
		},
	}

	err := action.Remove("work")
	if err != nil {
		t.Fatalf("Remove should succeed even if config reload fails: %v", err)
	}

	// Warning should be on stderr.
	if !strings.Contains(errOut.String(), "warning") {
		t.Errorf("stderr should contain warning about config reload: %q", errOut.String())
	}
	// Output should still confirm removal.
	if !strings.Contains(out.String(), "Schedule removed") {
		t.Errorf("stdout should confirm removal: %q", out.String())
	}
}

func TestScheduleCreate_DefaultScheduler(t *testing.T) {
	// When NewScheduler is nil, Run() should use the default scheduler.
	// We test with an invalid interval to short-circuit before scheduler use.
	var out, errOut strings.Builder

	action := &ScheduleAction{
		ConfigLoader: func() (*config.Config, error) {
			return &config.Config{SchemaVersion: "0.3.0"}, nil
		},
		Stdout: &out,
		Stderr: &errOut,
		// NewScheduler is nil — exercises default.
	}

	err := action.Create("work", "bananas")
	if err == nil {
		t.Fatal("expected error for invalid interval (default scheduler)")
	}
	if !strings.Contains(err.Error(), "invalid interval") {
		t.Errorf("error should mention invalid interval: %v", err)
	}
}
