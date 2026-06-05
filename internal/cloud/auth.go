package cloud

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/danielxxomg/bak-cli/internal/config"
)

// ResolveToken obtains a GitHub token using the following precedence:
//  1. GITHUB_TOKEN environment variable
//  2. bak config (github.token key)
//
// Returns an empty string when no token is found in any source.
func ResolveToken(cfg *config.Config) (string, string) {
	// 1. Environment variable.
	if tok := os.Getenv("GITHUB_TOKEN"); tok != "" {
		return tok, "environment variable GITHUB_TOKEN"
	}

	// 2. Config file.
	if cfg != nil {
		tok, err := cfg.Get("github.token")
		if err == nil && tok != "" {
			return tok, "config file (~/.config/bak/config.json)"
		}
	}

	return "", ""
}

// ValidateToken calls the GitHub API /user endpoint to verify that a
// token is valid. Returns nil when the token authenticates successfully.
func ValidateToken(token string) error {
	if token == "" {
		return fmt.Errorf("validate token: token is empty")
	}

	req, err := http.NewRequest(http.MethodGet, GistAPIBase+"/user", nil)
	if err != nil {
		return fmt.Errorf("validate token: build request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "bak-cli")

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("validate token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("validate token: invalid or expired token (HTTP 401)")
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("validate token: unexpected status %d", resp.StatusCode)
	}

	// Check for required scopes via response headers.
	scopes := resp.Header.Get("X-OAuth-Scopes")
	if scopes == "" {
		// Fine-grained PATs don't have scopes header — this is normal.
		return nil
	}

	if !strings.Contains(scopes, "gist") {
		return fmt.Errorf("validate token: token lacks 'gist' scope (scopes: %s)", scopes)
	}

	return nil
}
