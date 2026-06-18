package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/danielxxomg/bak-cli/internal/actions"
	"github.com/danielxxomg/bak-cli/internal/config"
)

// TestConfigShow_Registered verifies the config command is registered.
func TestConfigShow_Registered(t *testing.T) {
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub.Name() == "config" {
			found = true
			break
		}
	}
	if !found {
		t.Error("config command not registered on root")
	}
}

// TestConfigShow_Subcommands verifies show/get/set subcommands exist.
func TestConfigShow_Subcommands(t *testing.T) {
	var cfgCmd *cobra.Command
	for _, sub := range rootCmd.Commands() {
		if sub.Name() == "config" {
			cfgCmd = sub
			break
		}
	}
	if cfgCmd == nil {
		t.Skip("config command not registered yet")
	}

	subNames := make(map[string]bool)
	for _, sub := range cfgCmd.Commands() {
		subNames[sub.Name()] = true
	}

	for _, want := range []string{"show", "get", "set"} {
		if !subNames[want] {
			t.Errorf("config subcommand %q not registered", want)
		}
	}
}

// TestConfigShow_OutputRedactsToken verifies ConfigShow redacts tokens.
func TestConfigShow_OutputRedactsToken(t *testing.T) {
	cfg := &config.Config{
		SchemaVersion: "0.3.0",
		Providers: map[string]config.ProviderConfig{
			"github": {Token: "ghp_mysecrettoken1234", GistID: "test_gist"},
		},
	}
	var buf bytes.Buffer
	err := actions.ConfigShow(cfg, &buf)
	if err != nil {
		t.Fatalf("ConfigShow: %v", err)
	}
	output := buf.String()
	if strings.Contains(output, "ghp_mysecrettoken1234") {
		t.Error("output should NOT contain raw token")
	}
	if !strings.Contains(output, "***1234") {
		t.Error("output should contain redacted token '***1234'")
	}
	if !strings.Contains(output, "test_gist") {
		t.Error("output should contain non-sensitive gist_id")
	}
}

// TestConfigShow_NoConfigNil shows helpful message.
func TestConfigShow_NoConfigNil(t *testing.T) {
	var buf bytes.Buffer
	_ = actions.ConfigShow(nil, &buf)
	if buf.Len() == 0 {
		t.Error("nil config should produce a helpful message")
	}
}

// TestConfigSet_SettingsRoundTrip verifies set persists to file.
func TestConfigSet_SettingsRoundTrip(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	initial := `{"schema_version":"0.3.0","providers":{},"settings":{"default_preset":"quick"}}`
	if err := os.WriteFile(cfgPath, []byte(initial), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.LoadPath(cfgPath)
	if err != nil {
		t.Fatalf("LoadPath: %v", err)
	}

	var buf bytes.Buffer
	err = actions.ConfigSet(cfg, "settings.default_preset", "full", &buf)
	if err != nil {
		t.Fatalf("ConfigSet: %v", err)
	}

	reloaded, err := config.LoadPath(cfgPath)
	if err != nil {
		t.Fatalf("LoadPath: %v", err)
	}
	if reloaded.Settings.DefaultPreset != "full" {
		t.Errorf("settings.default_preset = %q, want %q", reloaded.Settings.DefaultPreset, "full")
	}
}

// TestConfigShow_Help verifies help output.
func TestConfigShow_Help(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	rootCmd.SetArgs([]string{"config", "--help"})
	rootCmd.Execute()

	output := buf.String()
	if !strings.Contains(output, "config") {
		t.Error("config help should mention 'config'")
	}
}
