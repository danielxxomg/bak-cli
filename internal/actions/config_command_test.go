package actions

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/danielxxomg/bak-cli/internal/config"
)

// testConfig returns a Config with both sensitive and non-sensitive values.
func testConfig() *config.Config {
	return &config.Config{
		SchemaVersion: "0.3.0",
		Providers: map[string]config.ProviderConfig{
			"github":   {Token: "ghp_abcdef1234567890", GistID: "gist_123"},
			"codeberg": {Token: "cb_token_longerthan4"},
		},
		Settings: config.Settings{
			DefaultPreset: "quick",
			MaxFileSize:   1048576,
		},
	}
}

// TestConfigShow_RedactsTokens verifies ConfigShow redacts tokens in output.
func TestConfigShow_RedactsTokens(t *testing.T) {
	cfg := testConfig()
	var buf bytes.Buffer

	err := ConfigShow(cfg, &buf)
	if err != nil {
		t.Fatalf("ConfigShow: %v", err)
	}

	output := buf.String()

	// Redacted token format: "***" + last 4 chars.
	if !strings.Contains(output, "***7890") {
		t.Error("output should contain redacted token '***7890'")
	}
	// Redacted short token.
	if !strings.Contains(output, "***han4") {
		t.Error("output should contain redacted short token '***han4'")
	}
	// Raw token should NOT appear.
	if strings.Contains(output, "ghp_abcdef1234567890") {
		t.Error("output should NOT contain raw token 'ghp_abcdef1234567890'")
	}
	// Non-sensitive values should appear.
	if !strings.Contains(output, "quick") {
		t.Error("output should contain non-sensitive value 'quick'")
	}
	if !strings.Contains(output, "gist_123") {
		t.Error("output should contain non-sensitive value 'gist_123'")
	}
}

// TestConfigShow_NoConfig handles nil config gracefully.
func TestConfigShow_NoConfig(t *testing.T) {
	var buf bytes.Buffer
	err := ConfigShow(nil, &buf)
	if err != nil {
		t.Fatalf("ConfigShow(nil): %v", err)
	}
	if buf.Len() == 0 {
		t.Error("ConfigShow(nil) should produce a helpful message")
	}
}

// TestConfigGet_RedactsToken verifies config get on sensitive keys redacts.
func TestConfigGet_RedactsToken(t *testing.T) {
	cfg := testConfig()
	var buf bytes.Buffer

	err := ConfigGet(cfg, "providers.github.token", &buf)
	if err != nil {
		t.Fatalf("ConfigGet: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "***") {
		t.Error("token output should be redacted")
	}
	if strings.Contains(output, "ghp_abcdef1234567890") {
		t.Error("token output should NOT contain raw token")
	}
}

// TestConfigGet_NonSensitive returns plain value.
func TestConfigGet_NonSensitive(t *testing.T) {
	cfg := testConfig()
	var buf bytes.Buffer

	err := ConfigGet(cfg, "providers.github.gist_id", &buf)
	if err != nil {
		t.Fatalf("ConfigGet: %v", err)
	}

	output := strings.TrimSpace(buf.String())
	if output != "gist_123" {
		t.Errorf("ConfigGet gist_id = %q, want %q", output, "gist_123")
	}
}

// TestConfigSet_Persists verifies set+get round-trip through config file.
func TestConfigSet_Persists(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	data, _ := json.Marshal(testConfig())
	os.WriteFile(cfgPath, data, 0644)
	cfg, err := config.LoadPath(cfgPath)
	if err != nil {
		t.Fatalf("LoadPath: %v", err)
	}

	var buf bytes.Buffer
	err = ConfigSet(cfg, "settings.default_preset", "full", &buf)
	if err != nil {
		t.Fatalf("ConfigSet: %v", err)
	}

	// Reload config to verify persistence.
	cfg2, err := config.LoadPath(cfgPath)
	if err != nil {
		t.Fatalf("LoadPath: %v", err)
	}

	val, err := cfg2.Get("settings.default_preset")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if val != "full" {
		t.Errorf("settings.default_preset = %q, want %q", val, "full")
	}
}

// TestConfigSet_InvalidKey errors helpfully.
func TestConfigSet_InvalidKey(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	data, _ := json.Marshal(testConfig())
	os.WriteFile(cfgPath, data, 0644)
	cfg, err := config.LoadPath(cfgPath)
	if err != nil {
		t.Fatalf("LoadPath: %v", err)
	}

	var buf bytes.Buffer
	err = ConfigSet(cfg, "settings.nonexistent", "value", &buf)
	if err == nil {
		t.Error("ConfigSet with invalid key should error")
	}
	if !strings.Contains(err.Error(), "unknown config key") {
		t.Errorf("error should mention 'unknown config key', got: %v", err)
	}
}

// TestRedactString covers the redaction helper.
func TestRedactString(t *testing.T) {
	tests := []struct {
		name string
		key  string
		val  string
		want string
	}{
		{"long token", "token", "ghp_abcdef1234567890", "***7890"},
		{"short token", "token", "ab", "***ab"},
		{"4-char token", "api_key", "abcd", "***abcd"},
		{"empty token", "token", "", ""},
		{"non-sensitive key", "gist_id", "gist_123", "gist_123"},
		{"secret key", "client_secret", "mysecret", "***cret"},
		{"password key", "password", "pass12345678", "***5678"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RedactString(tt.key, tt.val)
			if got != tt.want {
				t.Errorf("RedactString(%q, %q) = %q, want %q", tt.key, tt.val, got, tt.want)
			}
		})
	}
}

// TestRedactJSON covers recursive JSON redaction.
func TestRedactJSON(t *testing.T) {
	cfg := testConfig()
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	redacted, err := RedactJSON(data)
	if err != nil {
		t.Fatalf("RedactJSON: %v", err)
	}

	output := string(redacted)

	// Tokens should be redacted.
	if strings.Contains(output, "ghp_abcdef1234567890") {
		t.Error("redacted JSON should NOT contain raw token")
	}
	if !strings.Contains(output, "***7890") {
		t.Error("redacted JSON should contain '***7890'")
	}
	// Non-sensitive should remain.
	if !strings.Contains(output, "gist_123") {
		t.Error("redacted JSON should contain non-sensitive gist_id")
	}
}
