package cloud

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/danielxxomg/bak-cli/internal/config"
)

func TestValidateToken_Success(t *testing.T) {
	_, cleanup := setupMockGistAPI(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/user" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.Header.Get("Authorization") != "Bearer valid-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("X-OAuth-Scopes", "gist, user")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"login": "testuser"})
	})
	defer cleanup()

	err := ValidateToken("valid-token")
	if err != nil {
		t.Fatalf("ValidateToken: unexpected error: %v", err)
	}
}

func TestValidateToken_Unauthorized(t *testing.T) {
	_, cleanup := setupMockGistAPI(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	defer cleanup()

	err := ValidateToken("invalid-token")
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
	if !strings.Contains(err.Error(), "invalid or expired") {
		t.Errorf("error = %v, want 'invalid or expired'", err)
	}
}

func TestValidateToken_Empty(t *testing.T) {
	err := ValidateToken("")
	if err == nil {
		t.Fatal("expected error for empty token")
	}
}

func TestValidateToken_NoGistScope(t *testing.T) {
	_, cleanup := setupMockGistAPI(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/user" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		// Classic PAT without gist scope.
		w.Header().Set("X-OAuth-Scopes", "user, repo")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"login": "testuser"})
	})
	defer cleanup()

	err := ValidateToken("valid-token-no-gist")
	if err == nil {
		t.Fatal("expected error for missing gist scope")
	}
	if !strings.Contains(err.Error(), "gist") {
		t.Errorf("error = %v, want mention of 'gist'", err)
	}
}

func TestValidateToken_FineGrainedPAT(t *testing.T) {
	// Fine-grained PATs don't have X-OAuth-Scopes header.
	_, cleanup := setupMockGistAPI(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"login": "testuser"})
	})
	defer cleanup()

	err := ValidateToken("fine-grained-token")
	if err != nil {
		t.Fatalf("fine-grained PAT should pass: %v", err)
	}
}

func TestResolveProviderToken(t *testing.T) {
	origEnv := os.Getenv("GITHUB_TOKEN")
	defer os.Setenv("GITHUB_TOKEN", origEnv)

	t.Run("github-gist from environment", func(t *testing.T) {
		os.Setenv("GITHUB_TOKEN", "ghp_env123")
		defer os.Unsetenv("GITHUB_TOKEN")

		tok, source := ResolveProviderToken("github-gist", nil)
		if tok != "ghp_env123" {
			t.Errorf("token = %q, want ghp_env123", tok)
		}
		if !strings.Contains(source, "GITHUB_TOKEN") {
			t.Errorf("source = %q, want mention of GITHUB_TOKEN", source)
		}
	})

	t.Run("github-gist from config", func(t *testing.T) {
		os.Unsetenv("GITHUB_TOKEN")

		cfg := &config.Config{}
		_ = cfg.Set("github.token", "cfg_token456")

		tok, source := ResolveProviderToken("github-gist", cfg)
		if tok != "cfg_token456" {
			t.Errorf("token = %q, want cfg_token456", tok)
		}
		if !strings.Contains(source, "config") {
			t.Errorf("source = %q, want mention of config", source)
		}
	})

	t.Run("env takes precedence over config", func(t *testing.T) {
		os.Setenv("GITHUB_TOKEN", "env_wins")
		defer os.Unsetenv("GITHUB_TOKEN")

		cfg := &config.Config{}
		_ = cfg.Set("github.token", "config_loses")

		tok, source := ResolveProviderToken("github-gist", cfg)
		if tok != "env_wins" {
			t.Errorf("token = %q, want env_wins (env must win)", tok)
		}
		if !strings.Contains(source, "environment") {
			t.Errorf("source = %q, want environment", source)
		}
	})

	t.Run("no token anywhere", func(t *testing.T) {
		os.Unsetenv("GITHUB_TOKEN")

		tok, source := ResolveProviderToken("github-gist", nil)
		if tok != "" {
			t.Errorf("expected empty token, got %q", tok)
		}
		if source != "" {
			t.Errorf("expected empty source, got %q", source)
		}
	})

	t.Run("codeberg from environment", func(t *testing.T) {
		os.Setenv("CODEBERG_TOKEN", "cb_token789")
		os.Unsetenv("GITHUB_TOKEN")
		defer os.Unsetenv("CODEBERG_TOKEN")

		tok, source := ResolveProviderToken("codeberg", nil)
		if tok != "cb_token789" {
			t.Errorf("token = %q, want cb_token789", tok)
		}
		if !strings.Contains(source, "CODEBERG_TOKEN") {
			t.Errorf("source = %q, want CODEBERG_TOKEN", source)
		}
	})

	t.Run("gitea from config", func(t *testing.T) {
		os.Unsetenv("GITEA_TOKEN")

		cfg := &config.Config{}
		_ = cfg.Set("providers.gitea.token", "gitea_cfg")

		tok, source := ResolveProviderToken("gitea", cfg)
		if tok != "gitea_cfg" {
			t.Errorf("token = %q, want gitea_cfg", tok)
		}
		if !strings.Contains(source, "config") {
			t.Errorf("source = %q, want config", source)
		}
	})

	t.Run("unknown provider returns empty", func(t *testing.T) {
		tok, source := ResolveProviderToken("unknown-provider", nil)
		if tok != "" {
			t.Errorf("expected empty token for unknown provider, got %q", tok)
		}
		if source != "" {
			t.Errorf("expected empty source, got %q", source)
		}
	})

	t.Run("github-repo uses GITHUB_TOKEN", func(t *testing.T) {
		os.Setenv("GITHUB_TOKEN", "repo_token")
		defer os.Unsetenv("GITHUB_TOKEN")

		tok, _ := ResolveProviderToken("github-repo", nil)
		if tok != "repo_token" {
			t.Errorf("token = %q, want repo_token", tok)
		}
	})
}

func TestResolveToken(t *testing.T) {
	// Save and restore env.
	origEnv := os.Getenv("GITHUB_TOKEN")
	defer os.Setenv("GITHUB_TOKEN", origEnv)

	t.Run("from environment", func(t *testing.T) {
		os.Setenv("GITHUB_TOKEN", "env-token-123")
		defer os.Unsetenv("GITHUB_TOKEN")

		tok, source := ResolveToken(nil)
		if tok != "env-token-123" {
			t.Errorf("token = %q, want env-token-123", tok)
		}
		if !strings.Contains(source, "environment") {
			t.Errorf("source = %q, want mention of environment", source)
		}
	})

	t.Run("from config", func(t *testing.T) {
		os.Unsetenv("GITHUB_TOKEN")

		cfg := &config.Config{}
		_ = cfg.Set("github.token", "config-token-456")

		tok, source := ResolveToken(cfg)
		if tok != "config-token-456" {
			t.Errorf("token = %q, want config-token-456", tok)
		}
		if !strings.Contains(source, "config") {
			t.Errorf("source = %q, want mention of config", source)
		}
	})

	t.Run("environment takes precedence over config", func(t *testing.T) {
		os.Setenv("GITHUB_TOKEN", "env-first")
		defer os.Unsetenv("GITHUB_TOKEN")

		cfg := &config.Config{}
		_ = cfg.Set("github.token", "config-second")

		tok, source := ResolveToken(cfg)
		if tok != "env-first" {
			t.Errorf("token = %q, want env-first (env should win)", tok)
		}
		if !strings.Contains(source, "environment") {
			t.Errorf("source = %q, want environment", source)
		}
	})

	t.Run("no token anywhere", func(t *testing.T) {
		os.Unsetenv("GITHUB_TOKEN")

		tok, source := ResolveToken(nil)
		if tok != "" {
			t.Errorf("expected empty token, got %q", tok)
		}
		if source != "" {
			t.Errorf("expected empty source, got %q", source)
		}
	})
}
