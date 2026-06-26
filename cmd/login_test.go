package cmd

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/danielxxomg/bak-cli/internal/cloud"
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

// TestRunLogin_EmptyToken proves the device-flow login path is deterministic and
// does NOT touch the real network. The Device Flow is redirected to a local
// httptest server that never authorizes, so it returns "timed out" within the
// server-advertised expires_in (1s). Calling runLoginWithDeps directly (rather
// than rootCmd.Execute) exercises the exact seam production uses. See REQ-CI-009.
func TestRunLogin_EmptyToken(t *testing.T) {
	if os.Getenv("GITHUB_TOKEN") != "" {
		t.Skip("GITHUB_TOKEN is set, env token would short-circuit the device flow")
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/login/device/code"):
			fmt.Fprint(w, `{"device_code":"dc","user_code":"UC","verification_uri":"http://x","interval":1,"expires_in":1}`)
		default:
			fmt.Fprint(w, `{"error":"authorization_pending"}`)
		}
	}))
	t.Cleanup(srv.Close)

	origBase := cloud.DeviceLoginBase
	cloud.DeviceLoginBase = srv.URL
	t.Cleanup(func() { cloud.DeviceLoginBase = origBase })

	deps, _, _ := setupTestDeps(t)

	start := time.Now()
	err := runLoginWithDeps(&cobra.Command{}, nil, deps)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected error from un-authorized device login, got nil")
	}
	// Must resolve in seconds, not the server-advertised 10 minutes — proves no
	// real network call and the test cannot hang CI.
	if elapsed > 2*time.Second {
		t.Errorf("login exceeded 2s (elapsed=%v); test is not isolated from the network", elapsed)
	}
	if !strings.Contains(err.Error(), "timed out") && !strings.Contains(err.Error(), "token") {
		t.Errorf("expected a token/timeout error, got: %v", err)
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
