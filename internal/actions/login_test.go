package actions

import (
	"fmt"
	"strings"
	"testing"

	"github.com/danielxxomg/bak-cli/internal/config"
)

// oauthStub implements the oauthTokenRequester interface for testing.
type oauthStub struct {
	token string
	err   error
}

func (s *oauthStub) RequestToken() (string, error) {
	return s.token, s.err
}

// Compile-time interface compliance checks.
var _ ConfigSaver = (*MockConfigSaver)(nil)
var _ oauthTokenRequester = (*oauthStub)(nil)

// MockConfigSaver implements ConfigSaver without touching the real disk.
// It records the last saved token for verification.
type MockConfigSaver struct {
	cfg       *config.Config
	saveCount int
	saveErr   error
}

func (f *MockConfigSaver) Save() error {
	f.saveCount++
	return f.saveErr
}

func (f *MockConfigSaver) Set(key, value string) error {
	return f.cfg.Set(key, value)
}

// --- Table-Driven PAT Flow Tests ---

func TestLoginAction_PATFlow(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	tests := []struct {
		name       string
		setupCfg   map[string]config.ProviderConfig
		input      string
		validator  func(string) error
		wantToken  string
		wantErr    string
		wantOutput string
	}{
		{
			name:       "no_existing_token",
			input:      "fresh-token-456\n",
			wantToken:  "fresh-token-456",
			wantOutput: "Token saved",
		},
		{
			name:    "empty_token",
			input:   "\n",
			wantErr: "cannot be empty",
		},
		{
			name: "replace_yes",
			setupCfg: map[string]config.ProviderConfig{
				"github": {Token: "existing-token"},
			},
			input:      "y\ngithub-token-123\n",
			wantToken:  "github-token-123",
			wantOutput: "Token saved",
		},
		{
			name: "replace_no",
			setupCfg: map[string]config.ProviderConfig{
				"github": {Token: "existing-token"},
			},
			input:      "n\n",
			wantOutput: "cancelled",
		},
		{
			name:  "validation_failure",
			input: "bad-token\n",
			validator: func(string) error {
				return fmt.Errorf("invalid token")
			},
			wantErr: "validation failed",
		},
	}

	for _, tt := range tests { //nolint:paralleltest // subtests share table/struct state
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
			t.Setenv("GITHUB_TOKEN", "")
			_, cfg := setupConfigDir(t, tt.setupCfg)

			validator := tt.validator
			if validator == nil {
				validator = func(string) error { return nil }
			}

			saver := &MockConfigSaver{cfg: cfg}
			action := &LoginAction{
				Stdin:          strings.NewReader(tt.input),
				TokenValidator: validator,
				ConfigSaver:    saver,
				Config:         cfg,
			}

			out := &strings.Builder{}
			err := action.Run("github-gist", out)

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("error = %q, want to contain %q", err.Error(), tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			output := out.String()
			if tt.wantOutput != "" && !strings.Contains(output, tt.wantOutput) {
				t.Errorf("output = %q, want to contain %q", output, tt.wantOutput)
			}

			if tt.wantToken != "" {
				tok, err := cfg.Get("github.token")
				if err != nil {
					t.Fatalf("cfg.Get: %v", err)
				}
				if tok != tt.wantToken {
					t.Errorf("token = %q, want %q", tok, tt.wantToken)
				}
			}
		})
	}
}

func TestLoginAction_NonGitHubProvider(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	_, cfg := setupConfigDir(t, nil)
	saver := &MockConfigSaver{cfg: cfg}
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

// --- OAuth Dispatch Error Cases (table-driven) ---

func TestLoginAction_OAuthDispatch_Errors(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	tests := []struct {
		name      string
		stub      oauthStub
		validator func(string) error
		wantErr   string
	}{
		{
			name:    "request_token_failure",
			stub:    oauthStub{err: fmt.Errorf("network error")},
			wantErr: "oauth",
		},
		{
			name:      "validation_failure",
			stub:      oauthStub{token: "gho_bad"},
			validator: func(string) error { return fmt.Errorf("invalid oauth token") },
			wantErr:   "validation failed",
		},
	}

	for _, tt := range tests { //nolint:paralleltest // subtests share table/struct state
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
			t.Setenv("GITHUB_TOKEN", "")
			_, cfg := setupConfigDir(t, nil)

			validator := tt.validator
			if validator == nil {
				validator = func(string) error { return nil }
			}

			saver := &MockConfigSaver{cfg: cfg}
			action := &LoginAction{
				Stdin:          strings.NewReader(""),
				TokenValidator: validator,
				ConfigSaver:    saver,
				Config:         cfg,
				OAuthClient:    &tt.stub,
			}

			out := &strings.Builder{}
			err := action.Run("github-gist", out)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want to contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

// --- Table-Driven OAuth Success Tests ---

func TestLoginAction_OAuthDispatch_SuccessCases(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	tests := []struct {
		name       string
		setupCfg   map[string]config.ProviderConfig
		input      string
		stub       oauthStub
		wantToken  string
		wantOutput string
	}{
		{
			name:      "oauth_success",
			stub:      oauthStub{token: "gho_oauth_token"},
			wantToken: "gho_oauth_token",
		},
		{
			name:       "pat_fallback_when_no_oauth",
			input:      "manual-pat-token\n",
			stub:       oauthStub{}, // zero-value — not used since OAuthClient is nil
			wantToken:  "manual-pat-token",
			wantOutput: "Enter GitHub personal access token",
		},
		{
			name: "oauth_replace_confirm",
			setupCfg: map[string]config.ProviderConfig{
				"github": {Token: "existing-token"},
			},
			input:      "y\n",
			stub:       oauthStub{token: "gho_new_token"},
			wantToken:  "gho_new_token",
			wantOutput: "Token already configured",
		},
	}

	for _, tt := range tests { //nolint:paralleltest // subtests share table/struct state
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
			t.Setenv("GITHUB_TOKEN", "")
			_, cfg := setupConfigDir(t, tt.setupCfg)

			input := tt.input
			if input == "" {
				input = "\n"
			}

			saver := &MockConfigSaver{cfg: cfg}
			var oauthClient oauthTokenRequester
			if tt.name != "pat_fallback_when_no_oauth" {
				oauthClient = &tt.stub
			}

			action := &LoginAction{
				Stdin:          strings.NewReader(input),
				TokenValidator: func(token string) error { return nil },
				ConfigSaver:    saver,
				Config:         cfg,
				OAuthClient:    oauthClient,
			}

			out := &strings.Builder{}
			err := action.Run("github-gist", out)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			output := out.String()
			if tt.wantOutput != "" && !strings.Contains(output, tt.wantOutput) {
				t.Errorf("output = %q, want to contain %q", output, tt.wantOutput)
			}

			tok, err := cfg.Get("github.token")
			if err != nil {
				t.Fatalf("cfg.Get: %v", err)
			}
			if tok != tt.wantToken {
				t.Errorf("token = %q, want %q", tok, tt.wantToken)
			}
		})
	}
}
