package actions

import (
	"errors"
	"strings"
	"testing"

	"github.com/danielxxomg/bak-cli/internal/config"
)

func TestLoginInteractiveAction_Success(t *testing.T) {
	var out strings.Builder

	action := &LoginInteractiveAction{
		ConfigLoader: func() (*config.Config, error) {
			return &config.Config{
				SchemaVersion: "0.3.0",
				Providers: map[string]config.ProviderConfig{
					"github-gist": {Token: "ghp_test"},
				},
			}, nil
		},
		Wizard: func(providers []string) (string, error) {
			return "github-gist", nil
		},
		Stdout: &out,
	}

	selected, err := action.Run()
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if selected != "github-gist" {
		t.Errorf("selected = %q, want github-gist", selected)
	}
}

func TestLoginInteractiveAction_Cancel(t *testing.T) {
	var out strings.Builder

	action := &LoginInteractiveAction{
		ConfigLoader: func() (*config.Config, error) {
			return &config.Config{SchemaVersion: "0.3.0"}, nil
		},
		Wizard: func(providers []string) (string, error) {
			return "", nil // user cancelled
		},
		Stdout: &out,
	}

	selected, err := action.Run()
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if selected != "" {
		t.Errorf("selected should be empty on cancel, got %q", selected)
	}
}

func TestLoginInteractiveAction_ConfigLoadError(t *testing.T) {
	var out strings.Builder

	action := &LoginInteractiveAction{
		ConfigLoader: func() (*config.Config, error) {
			return nil, errors.New("config read error")
		},
		Wizard: func(providers []string) (string, error) {
			return "github-gist", nil
		},
		Stdout: &out,
	}

	_, err := action.Run()
	if err == nil {
		t.Fatal("expected error when config fails")
	}
	if !strings.Contains(err.Error(), "load config") {
		t.Errorf("error should mention load config: %v", err)
	}
}

func TestLoginInteractiveAction_WizardError(t *testing.T) {
	var out strings.Builder

	action := &LoginInteractiveAction{
		ConfigLoader: func() (*config.Config, error) {
			return &config.Config{SchemaVersion: "0.3.0"}, nil
		},
		Wizard: func(providers []string) (string, error) {
			return "", errors.New("tui crashed")
		},
		Stdout: &out,
	}

	_, err := action.Run()
	if err == nil {
		t.Fatal("expected error when wizard fails")
	}
	if !strings.Contains(err.Error(), "wizard") {
		t.Errorf("error should mention wizard: %v", err)
	}
}

func TestLoginInteractiveAction_AllProvidersIncluded(t *testing.T) {
	// Verify that the default five providers are always included.
	var out strings.Builder
	var receivedProviders []string

	action := &LoginInteractiveAction{
		ConfigLoader: func() (*config.Config, error) {
			return &config.Config{SchemaVersion: "0.3.0"}, nil
		},
		Wizard: func(providers []string) (string, error) {
			receivedProviders = providers
			return "rclone", nil
		},
		Stdout: &out,
	}

	_, err := action.Run()
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	want := []string{"github-gist", "github-repo", "codeberg", "gitea", "rclone"}
	if len(receivedProviders) < len(want) {
		t.Fatalf("expected at least %d providers, got %d: %v", len(want), len(receivedProviders), receivedProviders)
	}
	for _, w := range want {
		found := false
		for _, r := range receivedProviders {
			if r == w {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("provider %q not found in list: %v", w, receivedProviders)
		}
	}
}

func TestLoginInteractiveAction_CustomProviderAdded(t *testing.T) {
	// A provider configured in config but not in the default list
	// should be appended.
	var out strings.Builder
	var receivedProviders []string

	action := &LoginInteractiveAction{
		ConfigLoader: func() (*config.Config, error) {
			return &config.Config{
				SchemaVersion: "0.3.0",
				Providers: map[string]config.ProviderConfig{
					"custom-forge": {BaseURL: "https://git.example.com"},
				},
			}, nil
		},
		Wizard: func(providers []string) (string, error) {
			receivedProviders = providers
			return "custom-forge", nil
		},
		Stdout: &out,
	}

	_, err := action.Run()
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	found := false
	for _, p := range receivedProviders {
		if p == "custom-forge" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("custom-forge should be in provider list: %v", receivedProviders)
	}
}
