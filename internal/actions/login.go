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
}

// Run executes the login flow for the given provider. Only "github-gist"
// and "github" support interactive login; other providers are rejected
// with a hint to use bak config set.
func (a *LoginAction) Run(provider string, out io.Writer) error {
	// Only GitHub login is interactive; other providers use bak config set.
	if provider != "" && provider != "github-gist" && provider != "github" {
		return fmt.Errorf(
			"login for %q is not interactive — use 'bak config set providers.%s.token <your-token>'",
			provider, provider,
		)
	}

	stdin := a.Stdin
	reader := bufio.NewReader(stdin)

	// 1. Check if token already exists.
	tok, _ := cloud.ResolveToken(a.Config)
	if tok != "" {
		fmt.Fprintln(out, "Token already configured.")
		fmt.Fprint(out, "Do you want to replace it? [y/N]: ")
		answer, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("read input: %w", err)
		}
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "y" && answer != "yes" {
			fmt.Fprintln(out, "Login cancelled.")
			return nil
		}
	}

	// 2. Prompt for token.
	fmt.Fprint(out, "Enter GitHub personal access token: ")
	input, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("read token: %w", err)
	}
	token := strings.TrimSpace(input)

	if token == "" {
		return fmt.Errorf("login: token cannot be empty")
	}

	// 3. Validate token.
	fmt.Fprint(out, "Validating token... ")
	if a.TokenValidator != nil {
		if err := a.TokenValidator(token); err != nil {
			fmt.Fprintln(out, "❌")
			return fmt.Errorf("token validation failed: %w", err)
		}
	}
	fmt.Fprintln(out, "✅")

	// 4. Save to config (Set persists automatically).
	if err := a.ConfigSaver.Set("github.token", token); err != nil {
		return fmt.Errorf("save token: %w", err)
	}

	fmt.Fprintln(out, "Token saved successfully.")
	return nil
}
