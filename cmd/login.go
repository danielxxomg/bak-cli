package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/danielxxomg/bak-cli/internal/actions"
	"github.com/danielxxomg/bak-cli/internal/cloud"
	"github.com/danielxxomg/bak-cli/internal/config"
	"github.com/spf13/cobra"
)

var loginProvider string
var loginInteractive bool

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
  bak config set providers.gitea.token <your-token>

Use --interactive to launch the step-by-step wizard for provider selection.`,
	Args: cobra.NoArgs,
	RunE: runLogin,
}

func init() {
	loginCmd.Flags().StringVar(&loginProvider, "provider", "github-gist",
		"cloud provider to authenticate with (github-gist)")
	loginCmd.Flags().BoolVar(&loginInteractive, "interactive", false,
		"launch interactive wizard for provider selection")
	rootCmd.AddCommand(loginCmd)
}

func runLogin(cmd *cobra.Command, args []string) error {
	// Interactive wizard mode: select provider via TUI, then prompt for token.
	if loginInteractive {
		return runLoginInteractive(cmd)
	}

	// Only GitHub login is interactive; other providers use bak config set.
	if loginProvider != "" && loginProvider != "github-gist" && loginProvider != "github" {
		return fmt.Errorf(
			"login for %q is not interactive — use 'bak config set providers.%s.token <your-token>'",
			loginProvider, loginProvider,
		)
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	action := &actions.LoginAction{
		Stdin:          os.Stdin,
		TokenValidator: cloud.ValidateToken,
		ConfigSaver:    cfg,
		Config:         cfg,
	}

	return action.Run(loginProvider, cmd.OutOrStdout())
}

// runLoginInteractive launches the interactive wizard to select a provider
// and then falls through to the normal token entry flow.
func runLoginInteractive(cmd *cobra.Command) error {
	out := cmd.OutOrStdout()

	// Build provider list: include common providers even if not yet configured.
	providers := []string{"github-gist", "github-repo", "codeberg", "gitea", "rclone"}

	// Also include any providers already configured.
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	for k := range cfg.Providers {
		found := false
		for _, p := range providers {
			if p == k {
				found = true
				break
			}
		}
		if !found {
			providers = append(providers, k)
		}
	}

	// Launch wizard (only provider selection step is relevant for login).
	if !isTTY() {
		return fmt.Errorf("interactive login requires a terminal (TTY)")
	}
	m := newWizardModel("login", providers)
	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("wizard: %w", err)
	}

	wm := finalModel.(*wizardModel)
	if !wm.confirmed {
		fmt.Fprintln(out, "Login cancelled.")
		return nil
	}

	// Use the selected provider for the rest of the login flow.
	loginProvider = wm.selectedProvider
	return runLogin(cmd, nil)
}
