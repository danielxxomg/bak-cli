package cmd

import (
	"bytes"
	"strings"
	"testing"
)

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
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	// Reset rootCmd args to avoid stale args from other tests.
	rootCmd.SetArgs([]string{"schedule", "--help"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("schedule --help: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "schedule") {
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
	bufOut := new(bytes.Buffer)
	bufErr := new(bytes.Buffer)
	rootCmd.SetOut(bufOut)
	rootCmd.SetErr(bufErr)

	rootCmd.SetArgs([]string{"schedule", "create", "nonexistent-profile", "--every", "daily"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when profile not found")
	}
	errStr := err.Error()
	if !strings.Contains(errStr, "not found") && !strings.Contains(errStr, "load config") && !strings.Contains(errStr, "schtasks") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestScheduleCreate_InvalidInterval(t *testing.T) {
	bufOut := new(bytes.Buffer)
	bufErr := new(bytes.Buffer)
	rootCmd.SetOut(bufOut)
	rootCmd.SetErr(bufErr)

	rootCmd.SetArgs([]string{"schedule", "create", "test", "--every", "hourly"})
	err := rootCmd.Execute()
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
	bufOut := new(bytes.Buffer)
	bufErr := new(bytes.Buffer)
	rootCmd.SetOut(bufOut)
	rootCmd.SetErr(bufErr)

	rootCmd.SetArgs([]string{"schedule", "list"})
	err := rootCmd.Execute()
	if err != nil {
		t.Logf("schedule list: %v (expected on systems without schtasks)", err)
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
	bufOut := new(bytes.Buffer)
	bufErr := new(bytes.Buffer)
	rootCmd.SetOut(bufOut)
	rootCmd.SetErr(bufErr)

	rootCmd.SetArgs([]string{"schedule", "remove", "nonexistent-profile"})
	err := rootCmd.Execute()
	if err != nil {
		t.Logf("schedule remove: %v (expected without admin/schtasks)", err)
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
