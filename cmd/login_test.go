package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

// --- login command execution tests ---

func TestRunLogin_EmptyToken(t *testing.T) {
	if os.Getenv("GITHUB_TOKEN") != "" {
		t.Skip("GITHUB_TOKEN is set, login may succeed with env token")
	}

	bufOut := new(bytes.Buffer)
	bufErr := new(bytes.Buffer)
	rootCmd.SetOut(bufOut)
	rootCmd.SetErr(bufErr)

	rootCmd.SetArgs([]string{"login"})
	err := rootCmd.Execute()

	if err == nil {
		t.Skip("login succeeded (token possibly already configured)")
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "token") && !strings.Contains(errStr, "config") && !strings.Contains(errStr, "read") {
		t.Errorf("unexpected error from login: %v", err)
	}
}

func TestLoginCmd_RunEIsSet(t *testing.T) {
	cmd := findSubcommand(t, "login")
	if cmd == nil {
		t.Fatal("login command not found")
	}
	if cmd.RunE == nil {
		t.Fatal("login should have RunE set to runLogin")
	}
}
