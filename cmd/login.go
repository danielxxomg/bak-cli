package cmd

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/atotto/clipboard"
	"github.com/spf13/cobra"

	"github.com/danielxxomg/bak-cli/internal/actions"
	"github.com/danielxxomg/bak-cli/internal/cloud"
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
	return runLoginWithDeps(cmd, args, depsFromCmd(cmd))
}

func runLoginWithDeps(cmd *cobra.Command, args []string, deps cmdDeps) error {
	// Interactive wizard mode: select provider via TUI, then prompt for token.
	if loginInteractive {
		return runLoginInteractiveWithDeps(cmd, deps)
	}

	// Validation is handled by LoginAction.Run (supports github-gist and github).
	cfg, err := deps.ConfigLoader()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	action := &actions.LoginAction{
		Stdin:          deps.Stdin,
		TokenValidator: cloud.ValidateToken,
		ConfigSaver:    cfg,
		Config:         cfg,
	}

	// Wire OAuth Device Flow if client ID env var is set.
	if clientID := os.Getenv("BAK_GITHUB_OAUTH_CLIENT_ID"); clientID != "" {
		action.OAuthClient = &cloud.DeviceClient{
			ClientID:    clientID,
			Out:         deps.Stdout,
			OpenBrowser: cloud.OpenBrowser,
			Clipboard:   clipboard.WriteAll,
		}
	}

	return action.Run(loginProvider, deps.Stdout)
}

func runLoginInteractiveWithDeps(cmd *cobra.Command, deps cmdDeps) error {
	if !isTTY() {
		return fmt.Errorf("interactive login requires a terminal (TTY)")
	}

	action := &actions.LoginInteractiveAction{
		ConfigLoader: deps.ConfigLoader,
		Stdout:       deps.Stdout,
		Wizard: func(providers []string) (string, error) {
			m := newWizardModel("login", providers)
			p := tea.NewProgram(m)
			finalModel, err := p.Run()
			if err != nil {
				return "", err
			}
			wm := finalModel.(*wizardModel)
			if !wm.confirmed {
				return "", nil
			}
			return wm.selectedProvider, nil
		},
	}

	selected, err := action.Run()
	if err != nil {
		return err
	}

	if selected == "" {
		_, _ = fmt.Fprintln(deps.Stdout, "Login cancelled.")
		return nil
	}

	// Use the selected provider for the rest of the login flow.
	loginProvider = selected
	return runLoginWithDeps(cmd, nil, deps)
}
