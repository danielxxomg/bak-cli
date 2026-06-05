package cmd

import (
	"bytes"
	"strings"
	"testing"
)

// --- pull command execution tests ---

func TestRunPull_ErrorsAppropriately(t *testing.T) {
	// pull requires a valid GitHub token. If one is configured,
	// it will attempt the API call and may fail differently.
	bufOut := new(bytes.Buffer)
	bufErr := new(bytes.Buffer)
	rootCmd.SetOut(bufOut)
	rootCmd.SetErr(bufErr)

	rootCmd.SetArgs([]string{"pull"})
	err := rootCmd.Execute()
	if err == nil {
		t.Skip("pull succeeded — valid GitHub token with stored gist ID configured")
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "token") &&
		!strings.Contains(errStr, "Gist") &&
		!strings.Contains(errStr, "gist") &&
		!strings.Contains(errStr, "backup") {
		t.Errorf("unexpected pull error: %v", err)
	}
}

func TestRunPull_ExplicitGistID(t *testing.T) {
	bufOut := new(bytes.Buffer)
	bufErr := new(bytes.Buffer)
	rootCmd.SetOut(bufOut)
	rootCmd.SetErr(bufErr)

	rootCmd.SetArgs([]string{"pull", "abc123"})
	err := rootCmd.Execute()

	if err == nil {
		t.Skip("pull with explicit gist ID succeeded (valid GitHub token configured)")
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "token") && !strings.Contains(errStr, "gist") && !strings.Contains(errStr, "Gist") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPullCmd_RunEIsSet(t *testing.T) {
	cmd := findSubcommand(t, "pull")
	if cmd == nil {
		t.Fatal("pull command not found")
	}
	if cmd.RunE == nil {
		t.Error("pull RunE should be set")
	}
}

// TestPullCmd_NoTokenExplicit ensures that pull --help works.
func TestPullCmd_HelpWorks(t *testing.T) {
	bufOut := new(bytes.Buffer)
	bufErr := new(bytes.Buffer)
	rootCmd.SetOut(bufOut)
	rootCmd.SetErr(bufErr)

	rootCmd.SetArgs([]string{"pull", "--help"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("pull --help should not error: %v", err)
	}
	output := bufOut.String()
	if !strings.Contains(output, "pull") {
		t.Error("help should mention 'pull'")
	}
}

func TestRunPull_TooManyArgs(t *testing.T) {
	// pull accepts Max 1 arg (cobra.MaximumNArgs(1)).
	// Test the arg validator directly.
	cmd := findSubcommand(t, "pull")
	if cmd == nil {
		t.Fatal("pull command not found")
	}
	err := cmd.Args(cmd, []string{"abc", "xyz"})
	if err == nil {
		t.Fatal("expected pull command to reject 2 args")
	}
}

// TestPushCmd_HelpWorks ensures push --help always works.
func TestPushCmd_HelpWorks(t *testing.T) {
	bufOut := new(bytes.Buffer)
	bufErr := new(bytes.Buffer)
	rootCmd.SetOut(bufOut)
	rootCmd.SetErr(bufErr)

	rootCmd.SetArgs([]string{"push", "--help"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("push --help should not error: %v", err)
	}
	output := bufOut.String()
	if !strings.Contains(output, "push") {
		t.Error("help should mention 'push'")
	}
}
