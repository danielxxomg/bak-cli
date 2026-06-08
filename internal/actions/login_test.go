package actions

import (
	"fmt"
	"strings"
	"testing"

	"github.com/danielxxomg/bak-cli/internal/config"
)

// fakeConfigSaver implements a config save without touching the real disk.
// It records the last saved token for verification.
type fakeConfigSaver struct {
	cfg       *config.Config
	saveCount int
	saveErr   error
}

func (f *fakeConfigSaver) Save() error {
	f.saveCount++
	return f.saveErr
}

func (f *fakeConfigSaver) Set(key, value string) error {
	return f.cfg.Set(key, value)
}

func TestLoginAction_ReplaceYes(t *testing.T) {
	_, cfg := setupConfigDir(t, map[string]config.ProviderConfig{
		"github": {Token: "existing-token"},
	})

	saver := &fakeConfigSaver{cfg: cfg}
	stdin := strings.NewReader("y\ngithub-token-123\n")

	action := &LoginAction{
		Stdin:          stdin,
		TokenValidator: func(token string) error { return nil },
		ConfigSaver:    saver,
		Config:         cfg,
	}

	out := &strings.Builder{}
	err := action.Run("github-gist", out)
	if err != nil {
		t.Fatalf("LoginAction.Run: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "Token saved") {
		t.Errorf("output should confirm token saved: %q", output)
	}

	// Verify the token was set on the underlying config.
	tok, err := cfg.Get("github.token")
	if err != nil {
		t.Fatalf("cfg.Get: %v", err)
	}
	if tok != "github-token-123" {
		t.Errorf("token = %q, want github-token-123", tok)
	}
}

func TestLoginAction_ReplaceNo(t *testing.T) {
	_, cfg := setupConfigDir(t, map[string]config.ProviderConfig{
		"github": {Token: "existing-token"},
	})

	saver := &fakeConfigSaver{cfg: cfg}
	stdin := strings.NewReader("n\n")

	action := &LoginAction{
		Stdin:          stdin,
		TokenValidator: func(token string) error { return nil },
		ConfigSaver:    saver,
		Config:         cfg,
	}

	out := &strings.Builder{}
	err := action.Run("github-gist", out)
	if err != nil {
		t.Fatalf("LoginAction.Run: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "cancelled") {
		t.Errorf("output should mention cancelled: %q", output)
	}
}

func TestLoginAction_EmptyToken(t *testing.T) {
	// Clear GITHUB_TOKEN to ensure test isolation in CI
	t.Setenv("GITHUB_TOKEN", "")
	_, cfg := setupConfigDir(t, nil)

	saver := &fakeConfigSaver{cfg: cfg}
	stdin := strings.NewReader("\n")

	action := &LoginAction{
		Stdin:          stdin,
		TokenValidator: func(token string) error { return nil },
		ConfigSaver:    saver,
		Config:         cfg,
	}

	out := &strings.Builder{}
	err := action.Run("github-gist", out)
	if err == nil {
		t.Fatal("expected error for empty token")
	}
	if !strings.Contains(err.Error(), "cannot be empty") {
		t.Errorf("error should mention empty token: %v", err)
	}
}

func TestLoginAction_ValidationFailure(t *testing.T) {
	// Clear GITHUB_TOKEN to ensure test isolation in CI
	t.Setenv("GITHUB_TOKEN", "")
	_, cfg := setupConfigDir(t, nil)

	saver := &fakeConfigSaver{cfg: cfg}
	stdin := strings.NewReader("bad-token\n")

	action := &LoginAction{
		Stdin: stdin,
		TokenValidator: func(token string) error {
			return fmt.Errorf("invalid token")
		},
		ConfigSaver: saver,
		Config:      cfg,
	}

	out := &strings.Builder{}
	err := action.Run("github-gist", out)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "validation failed") {
		t.Errorf("error should mention validation failed: %v", err)
	}
}

func TestLoginAction_NoTokenYet(t *testing.T) {
	// Clear GITHUB_TOKEN to ensure test isolation in CI
	t.Setenv("GITHUB_TOKEN", "")
	_, cfg := setupConfigDir(t, nil)

	saver := &fakeConfigSaver{cfg: cfg}
	stdin := strings.NewReader("fresh-token-456\n")

	action := &LoginAction{
		Stdin:          stdin,
		TokenValidator: func(token string) error { return nil },
		ConfigSaver:    saver,
		Config:         cfg,
	}

	out := &strings.Builder{}
	err := action.Run("github-gist", out)
	if err != nil {
		t.Fatalf("LoginAction.Run: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "Token saved") {
		t.Errorf("output should confirm token saved: %q", output)
	}
}

func TestLoginAction_NonGitHubProvider(t *testing.T) {
	_, cfg := setupConfigDir(t, nil)

	saver := &fakeConfigSaver{cfg: cfg}
	action := &LoginAction{
		Stdin:          strings.NewReader(""),
		TokenValidator: func(token string) error { return nil },
		ConfigSaver:    saver,
		Config:         cfg,
	}

	out := &strings.Builder{}
	err := action.Run("codeberg", out)
	if err == nil {
		t.Fatal("expected error for non-interactive provider")
	}
	if !strings.Contains(err.Error(), "not interactive") {
		t.Errorf("error should mention not interactive: %v", err)
	}
}
