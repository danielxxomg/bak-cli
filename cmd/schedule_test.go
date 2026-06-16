package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/danielxxomg/bak-cli/internal/config"
	"github.com/danielxxomg/bak-cli/internal/schedule"
)

// MockScheduler implements schedule.Scheduler for cmd-level happy-path tests.
type MockScheduler struct {
	createCalls []struct {
		profile  string
		interval string
	}
	removeCalls []string
	entries     []schedule.ScheduleEntry
}

var _ schedule.Scheduler = (*MockScheduler)(nil)

func (m *MockScheduler) Create(profile, interval string) error {
	m.createCalls = append(m.createCalls, struct {
		profile  string
		interval string
	}{profile, interval})
	return nil
}

func (m *MockScheduler) Remove(profile string) error {
	m.removeCalls = append(m.removeCalls, profile)
	return nil
}

func (m *MockScheduler) List() ([]schedule.ScheduleEntry, error) {
	return m.entries, nil
}

// --- schedule command structure ---

func TestScheduleCmd_Registered(t *testing.T) {
	cmd := findSubcommand(t, "schedule")
	if cmd == nil {
		t.Fatal("schedule command not registered on root")
	}
}

func TestScheduleCmd_Structure(t *testing.T) {
	cmd := findSubcommand(t, "schedule")
	if cmd == nil {
		t.Fatal("schedule command not registered")
	}
	if cmd.Use != "schedule" {
		t.Errorf("Use = %q, want 'schedule'", cmd.Use)
	}
	if cmd.Short == "" {
		t.Error("schedule should have a short description")
	}
	// Parent command should not have RunE (subcommands do).
	if cmd.RunE != nil {
		t.Error("schedule parent should not have RunE")
	}
}

func TestScheduleCmd_HasSubcommands(t *testing.T) {
	cmd := findSubcommand(t, "schedule")
	if cmd == nil {
		t.Fatal("schedule command not found")
	}

	subs := cmd.Commands()
	names := make(map[string]bool)
	for _, s := range subs {
		names[s.Name()] = true
	}

	expected := []string{"create", "list", "remove"}
	for _, want := range expected {
		if !names[want] {
			t.Errorf("schedule should have subcommand %q", want)
		}
	}
}

func TestScheduleCmd_Help(t *testing.T) {
	// Help output doesn't require config — cobra handles it directly.
	var buf strings.Builder
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"schedule", "--help"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("schedule --help: %v", err)
	}

	if !strings.Contains(buf.String(), "schedule") {
		t.Error("help output should contain 'schedule'")
	}
}

// --- schedule create ---

func TestScheduleCreate_HasEveryFlag(t *testing.T) {
	createCmd, _, _ := rootCmd.Find([]string{"schedule", "create"})
	if createCmd == nil {
		t.Fatal("schedule create not found")
	}

	everyFlag := createCmd.Flags().Lookup("every")
	if everyFlag == nil {
		t.Fatal("schedule create should have --every flag")
	}
}

func TestScheduleCreate_RequiresArgs(t *testing.T) {
	createCmd, _, _ := rootCmd.Find([]string{"schedule", "create"})
	if createCmd == nil {
		t.Fatal("schedule create not found")
	}

	if createCmd.Args == nil {
		t.Error("schedule create should require profile name arg")
	}
}

func TestScheduleCreate_Execute(t *testing.T) {
	// Uses WithDeps to isolate from real config.
	var out, errOut strings.Builder
	deps := cmdDeps{
		ConfigLoader: func() (*config.Config, error) {
			return &config.Config{SchemaVersion: "0.3.0"}, nil
		},
		Stdout:       &out,
		Stderr:       &errOut,
		NewScheduler: func() schedule.Scheduler { return &MockScheduler{} },
	}

	scheduleCreateEvery = "daily"
	defer func() { scheduleCreateEvery = "" }()

	cmd := &cobra.Command{}
	err := runScheduleCreateWithDeps(cmd, []string{"nonexistent-profile"}, deps)
	if err == nil {
		t.Fatal("expected error when profile not found")
	}
	if !strings.Contains(err.Error(), "not found") && !strings.Contains(err.Error(), "load config") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestScheduleCreate_InvalidInterval(t *testing.T) {
	var out, errOut strings.Builder
	deps := cmdDeps{
		ConfigLoader: func() (*config.Config, error) {
			return &config.Config{SchemaVersion: "0.3.0"}, nil
		},
		Stdout:       &out,
		Stderr:       &errOut,
		NewScheduler: func() schedule.Scheduler { return &MockScheduler{} },
	}

	scheduleCreateEvery = "hourly"
	defer func() { scheduleCreateEvery = "" }()

	cmd := &cobra.Command{}
	err := runScheduleCreateWithDeps(cmd, []string{"test"}, deps)
	if err == nil {
		t.Fatal("expected error for invalid interval")
	}
	if !strings.Contains(err.Error(), "invalid interval") {
		t.Errorf("error should mention invalid interval: %v", err)
	}
}

func TestScheduleCreate_MissingEveryFlag(t *testing.T) {
	bufOut := new(bytes.Buffer)
	bufErr := new(bytes.Buffer)
	rootCmd.SetOut(bufOut)
	rootCmd.SetErr(bufErr)

	rootCmd.SetArgs([]string{"schedule", "create", "test"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when --every is missing")
	}
}

func TestScheduleCreate_NoArgs(t *testing.T) {
	bufOut := new(bytes.Buffer)
	bufErr := new(bytes.Buffer)
	rootCmd.SetOut(bufOut)
	rootCmd.SetErr(bufErr)

	rootCmd.SetArgs([]string{"schedule", "create"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error with no args")
	}
}

// --- schedule list ---

func TestScheduleList_NoArgs(t *testing.T) {
	cmd, _, _ := rootCmd.Find([]string{"schedule", "list"})
	if cmd == nil {
		t.Fatal("schedule list not found")
	}
}

func TestScheduleList_Execute(t *testing.T) {
	// Uses WithDeps to isolate from real config.
	var out, errOut strings.Builder
	deps := cmdDeps{
		ConfigLoader: func() (*config.Config, error) {
			return &config.Config{SchemaVersion: "0.3.0"}, nil
		},
		Stdout:       &out,
		Stderr:       &errOut,
		NewScheduler: func() schedule.Scheduler { return &MockScheduler{} },
	}

	cmd := &cobra.Command{}
	err := runScheduleListWithDeps(cmd, nil, deps)
	if err != nil {
		t.Logf("schedule list: %v", err)
	}
}

// --- schedule remove ---

func TestScheduleRemove_RequiresArgs(t *testing.T) {
	removeCmd, _, _ := rootCmd.Find([]string{"schedule", "remove"})
	if removeCmd == nil {
		t.Fatal("schedule remove not found")
	}

	if removeCmd.Args == nil {
		t.Error("schedule remove should require profile name arg")
	}
}

func TestScheduleRemove_Execute(t *testing.T) {
	// Uses WithDeps to isolate from real config and real scheduler.
	var out, errOut strings.Builder
	deps := cmdDeps{
		ConfigLoader: func() (*config.Config, error) {
			return &config.Config{SchemaVersion: "0.3.0"}, nil
		},
		Stdout:       &out,
		Stderr:       &errOut,
		NewScheduler: func() schedule.Scheduler { return &MockScheduler{} },
	}

	cmd := &cobra.Command{}
	err := runScheduleRemoveWithDeps(cmd, []string{"nonexistent-profile"}, deps)
	if err != nil {
		t.Logf("schedule remove: %v", err)
	}
}

func TestScheduleRemove_NoArgs(t *testing.T) {
	bufOut := new(bytes.Buffer)
	bufErr := new(bytes.Buffer)
	rootCmd.SetOut(bufOut)
	rootCmd.SetErr(bufErr)

	rootCmd.SetArgs([]string{"schedule", "remove"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error with no args")
	}
}

// --- schedule happy path (cmd-level with mock scheduler) ---

func TestScheduleHappyPath(t *testing.T) {
	tests := []struct {
		name      string
		setup     func() (*MockScheduler, cmdDeps, func())
		run       func(cmd *cobra.Command, args []string, deps cmdDeps) error
		args      []string
		assert    func(t *testing.T, sched *MockScheduler, output string)
	}{
		{
			name: "create",
			setup: func() (*MockScheduler, cmdDeps, func()) {
				sched := &MockScheduler{}
				var out, errOut strings.Builder
				cfg := &config.Config{
					SchemaVersion: "0.3.0",
					Profiles:     map[string]config.ProfileConfig{"work": {Provider: "github-gist", Preset: "quick"}},
				}
				deps := cmdDeps{
					ConfigLoader: func() (*config.Config, error) { return cfg, nil },
					Stdout:       &out,
					Stderr:       &errOut,
					NewScheduler: func() schedule.Scheduler { return sched },
				}
				scheduleCreateEvery = "daily"
				cleanup := func() { scheduleCreateEvery = "" }
				return sched, deps, cleanup
			},
			run:  runScheduleCreateWithDeps,
			args: []string{"work"},
			assert: func(t *testing.T, sched *MockScheduler, output string) {
				if len(sched.createCalls) != 1 {
					t.Fatalf("expected 1 Create call, got %d", len(sched.createCalls))
				}
				if sched.createCalls[0].profile != "work" {
					t.Errorf("Create profile = %q, want work", sched.createCalls[0].profile)
				}
				if sched.createCalls[0].interval != "daily" {
					t.Errorf("Create interval = %q, want daily", sched.createCalls[0].interval)
				}
				if !strings.Contains(output, "Schedule created") {
					t.Errorf("stdout should contain 'Schedule created': %q", output)
				}
			},
		},
		{
			name: "list",
			setup: func() (*MockScheduler, cmdDeps, func()) {
				sched := &MockScheduler{
					entries: []schedule.ScheduleEntry{
						{Profile: "work", Interval: "daily"},
						{Profile: "home", Interval: "weekly"},
					},
				}
				var out, errOut strings.Builder
				deps := cmdDeps{
					ConfigLoader: func() (*config.Config, error) {
						return &config.Config{SchemaVersion: "0.3.0"}, nil
					},
					Stdout:       &out,
					Stderr:       &errOut,
					NewScheduler: func() schedule.Scheduler { return sched },
				}
				return sched, deps, func() {}
			},
			run:  runScheduleListWithDeps,
			args: nil,
			assert: func(t *testing.T, _ *MockScheduler, output string) {
				if !strings.Contains(output, "work") || !strings.Contains(output, "daily") {
					t.Errorf("stdout should contain 'work' and 'daily': %q", output)
				}
				if !strings.Contains(output, "home") || !strings.Contains(output, "weekly") {
					t.Errorf("stdout should contain 'home' and 'weekly': %q", output)
				}
			},
		},
		{
			name: "remove",
			setup: func() (*MockScheduler, cmdDeps, func()) {
				sched := &MockScheduler{}
				var out, errOut strings.Builder
				cfg := &config.Config{
					SchemaVersion: "0.3.0",
					Profiles:     map[string]config.ProfileConfig{"work": {Provider: "github-gist", Preset: "quick"}},
				}
				deps := cmdDeps{
					ConfigLoader: func() (*config.Config, error) { return cfg, nil },
					Stdout:       &out,
					Stderr:       &errOut,
					NewScheduler: func() schedule.Scheduler { return sched },
				}
				return sched, deps, func() {}
			},
			run:  runScheduleRemoveWithDeps,
			args: []string{"work"},
			assert: func(t *testing.T, sched *MockScheduler, output string) {
				if len(sched.removeCalls) != 1 {
					t.Fatalf("expected 1 Remove call, got %d", len(sched.removeCalls))
				}
				if sched.removeCalls[0] != "work" {
					t.Errorf("Remove profile = %q, want work", sched.removeCalls[0])
				}
				if !strings.Contains(output, "Schedule removed") {
					t.Errorf("stdout should contain 'Schedule removed': %q", output)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sched, deps, cleanup := tt.setup()
			defer cleanup()

			cmd := &cobra.Command{}
			err := tt.run(cmd, tt.args, deps)
			if err != nil {
				t.Fatalf("runSchedule%s: %v", tt.name, err)
			}

			var output string
			if w, ok := deps.Stdout.(*strings.Builder); ok {
				output = w.String()
			}
			tt.assert(t, sched, output)
		})
	}
}
