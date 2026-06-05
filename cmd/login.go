package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/danielxxomg/bak-cli/internal/cloud"
	"github.com/danielxxomg/bak-cli/internal/config"
	"github.com/spf13/cobra"
)

var loginProvider string

// loginCmd represents the login command.
var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with a cloud provider for sync",
	Long: `Configure authentication for cloud backup providers.

For GitHub (default): Configures a personal access token (PAT) to enable
cloud backup via private GitHub Gists.

The token is stored in ~/.config/bak/config.json and used by the
push and pull commands.

Token requirements for GitHub:
  - Classic PAT: needs the 'gist' scope
  - Fine-grained PAT: needs read/write access to Gists

Create a token at: https://github.com/settings/tokens

For other providers (Codeberg, Gitea, etc.), use 'bak config set':
  bak config set providers.codeberg.token <your-token>
  bak config set providers.gitea.token <your-token>`,
	Args: cobra.NoArgs,
	RunE: runLogin,
}

func init() {
	loginCmd.Flags().StringVar(&loginProvider, "provider", "github-gist",
		"cloud provider to authenticate with (github-gist)")
	rootCmd.AddCommand(loginCmd)
}

func runLogin(cmd *cobra.Command, args []string) error {
	// Only GitHub login is interactive; other providers use bak config set.
	if loginProvider != "" && loginProvider != "github-gist" && loginProvider != "github" {
		return fmt.Errorf(
			"login for %q is not interactive — use 'bak config set providers.%s.token <your-token>'",
			loginProvider, loginProvider,
		)
	}
	// 1. Check if token already exists.
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	tok, _ := cloud.ResolveToken(cfg)
	if tok != "" {
		fmt.Println("Token already configured.")
		fmt.Print("Do you want to replace it? [y/N]: ")
		reader := bufio.NewReader(os.Stdin)
		answer, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("read input: %w", err)
		}
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "y" && answer != "yes" {
			fmt.Println("Login cancelled.")
			return nil
		}
	}

	// 2. Prompt for token.
	fmt.Print("Enter GitHub personal access token: ")
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("read token: %w", err)
	}
	token := strings.TrimSpace(input)

	if token == "" {
		return fmt.Errorf("token cannot be empty")
	}

	// 3. Validate token.
	fmt.Print("Validating token... ")
	if err := cloud.ValidateToken(token); err != nil {
		fmt.Println("❌")
		return fmt.Errorf("token validation failed: %w", err)
	}
	fmt.Println("✅")

	// 4. Save to config.
	if err := cfg.Set("github.token", token); err != nil {
		return fmt.Errorf("save token: %w", err)
	}

	fmt.Println("Token saved successfully.")
	return nil
}
