package actions

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/danielxxomg/bak-cli/internal/cloud"
	"github.com/danielxxomg/bak-cli/internal/config"
)

// ConfigSaver abstracts config persistence for login flows.
// Set stores a value and Save persists the config to disk.
type ConfigSaver interface {
	Set(key, value string) error
	Save() error
}

// oauthTokenRequester abstracts the OAuth Device Flow so tests can
// inject stubs. *cloud.DeviceClient satisfies this interface.
type oauthTokenRequester interface {
	RequestToken() (string, error)
}

// LoginAction encapsulates the interactive login workflow for a cloud
// provider. All I/O is injected for testability.
type LoginAction struct {
	// Stdin is the reader for user input (token + confirmation).
	// Nil falls back to os.Stdin in production wiring.
	Stdin io.Reader

	// TokenValidator validates the token against the cloud API.
	// The real implementation is cloud.ValidateToken.
	TokenValidator func(token string) error

	// ConfigSaver persists the token to config.
	ConfigSaver ConfigSaver

	// Config is the current configuration, used to check for existing tokens.
	Config *config.Config

	// OAuthClient enables OAuth Device Flow login when non-nil.
	// When nil, the existing manual PAT flow is used.
	OAuthClient oauthTokenRequester
}

// Run executes the login flow for the given provider. Only "github-gist"
// and "github" support interactive login; other providers are rejected
// with a hint to use bak config set.
func (a *LoginAction) Run(provider string, out io.Writer) error {
	// Only GitHub login is interactive; other providers use bak config set.
	if provider != "" && provider != "github-gist" && provider != providerGithub {
		return fmt.Errorf(
			"login for %q is not interactive — use 'bak config set providers.%s.token <your-token>'",
			provider, provider,
		)
	}

	stdin := a.Stdin
	reader := bufio.NewReader(stdin)

	// 1. Check if token already exists.
	tok, _ := cloud.ResolveToken(a.Config) // source description is informational only, not an error
	if tok != "" {
		_, _ = fmt.Fprintln(out, "Token already configured.")
		_, _ = fmt.Fprint(out, "Do you want to replace it? [y/N]: ")
		answer, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("read input: %w", err)
		}
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "y" && answer != "yes" {
			_, _ = fmt.Fprintln(out, "Login cancelled.")
			return nil
		}
	}

	// 2. OAuth Device Flow (if configured).
	if a.OAuthClient != nil {
		return a.runOAuthLogin(out)
	}

	// 3. Manual PAT prompt (fallback).
	_, _ = fmt.Fprint(out, "Enter GitHub personal access token: ")
	input, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("read token: %w", err)
	}
	token := strings.TrimSpace(input)

	if token == "" {
		return fmt.Errorf("login: token cannot be empty")
	}

	// 4. Validate token.
	return a.validateAndSave(token, out)
}

// runOAuthLogin performs the OAuth Device Flow and saves the resulting token.
func (a *LoginAction) runOAuthLogin(out io.Writer) error {
	_, _ = fmt.Fprintln(out, "Starting OAuth Device Flow...")

	token, err := a.OAuthClient.RequestToken()
	if err != nil {
		return fmt.Errorf("oauth login: %w", err)
	}

	return a.validateAndSave(token, out)
}

// validateAndSave validates a token and saves it to config.
func (a *LoginAction) validateAndSave(token string, out io.Writer) error {
	_, _ = fmt.Fprint(out, "Validating token... ")
	if a.TokenValidator != nil {
		if err := a.TokenValidator(token); err != nil {
			_, _ = fmt.Fprintln(out, "❌")
			return fmt.Errorf("token validation failed: %w", err)
		}
	}
	_, _ = fmt.Fprintln(out, "✅")

	// Save to config (Set persists automatically).
	if err := a.ConfigSaver.Set("github.token", token); err != nil {
		return fmt.Errorf("save token: %w", err)
	}

	_, _ = fmt.Fprintln(out, "Token saved successfully.")
	return nil
}
