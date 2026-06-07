package actions

import (
	"fmt"
	"io"

	"github.com/danielxxomg/bak-cli/internal/config"
)

// WizardRunner launches an interactive TUI wizard and returns the user's
// selected provider. An implementation is provided by the cmd package.
// If the user cancels, both return values are zero.
type WizardRunner func(providers []string) (selectedProvider string, err error)

// LoginInteractiveAction orchestrates the interactive login wizard flow.
// All I/O is injected for testability.
type LoginInteractiveAction struct {
	// ConfigLoader returns the current config, used to discover already-
	// configured providers for the selection list.
	ConfigLoader func() (*config.Config, error)

	// Wizard runs the TUI wizard for provider selection.
	Wizard WizardRunner

	// Stdout is the writer for user-facing messages.
	Stdout io.Writer
}

// Run builds the provider list, launches the wizard, and returns the
// selected provider (or empty string if cancelled).
func (a *LoginInteractiveAction) Run() (string, error) {
	// Build provider list: include common providers even if not yet configured.
	providers := []string{"github-gist", "github-repo", "codeberg", "gitea", "rclone"}

	// Also include any providers already configured.
	cfg, err := a.ConfigLoader()
	if err != nil {
		return "", fmt.Errorf("load config: %w", err)
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

	// Launch wizard.
	selected, err := a.Wizard(providers)
	if err != nil {
		return "", fmt.Errorf("wizard: %w", err)
	}

	return selected, nil
}
