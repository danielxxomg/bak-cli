package cmd

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/danielxxomg/bak-cli/internal/config"
)

// --- login command execution tests ---

func TestRunLoginWithDeps_ConfigLoaderError(t *testing.T) {
	deps, _, _ := setupTestDeps(t)
	deps.ConfigLoader = func() (*config.Config, error) {
		return nil, errors.New("disk failure")
	}

	cmd := &cobra.Command{}
	err := runLoginWithDeps(cmd, nil, deps)

	if err == nil {
		t.Fatal("expected error from failing ConfigLoader")
	}
	if !strings.Contains(err.Error(), "load config") {
		t.Errorf("error should contain 'load config', got: %v", err)
	}
	if !strings.Contains(err.Error(), "disk failure") {
		t.Errorf("error should wrap 'disk failure', got: %v", err)
	}
}

func TestRunLoginWithDeps_NonTTYGuard(t *testing.T) {
	// Override isTTY to simulate non-interactive terminal.
	origIsTTY := isTTY
	isTTY = func() bool { return false }
	defer func() { isTTY = origIsTTY }()

	// Enable interactive mode.
	origInteractive := loginInteractive
	loginInteractive = true
	defer func() { loginInteractive = origInteractive }()

	deps, _, _ := setupTestDeps(t)
	cmd := &cobra.Command{}
	err := runLoginWithDeps(cmd, nil, deps)

	if err == nil {
		t.Fatal("expected error from non-TTY interactive login")
	}
	if !strings.Contains(err.Error(), "TTY") {
		t.Errorf("error should contain 'TTY', got: %v", err)
	}
}

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

func TestRunLogin_NonGitHubProviders(t *testing.T) {
	tests := []struct {
		provider string
	}{
		{"codeberg"},
		{"gitea"},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			orig := loginProvider
			loginProvider = tt.provider
			defer func() { loginProvider = orig }()

			err := runLogin(nil, nil)
			if err == nil {
				t.Fatal("expected error for non-GitHub provider login")
			}
			if !strings.Contains(err.Error(), "config set") {
				t.Errorf("error should mention 'config set', got: %v", err)
			}
		})
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
