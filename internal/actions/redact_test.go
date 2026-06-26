package actions

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestSensitiveKeyMatcher verifies the isSensitiveKey helper catches
// all required patterns.
func TestSensitiveKeyMatcher(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	tests := []struct {
		name string
		key  string
		want bool
	}{
		{"token exact", "token", true},
		{"token in name", "github_token", true},
		{"api_key", "api_key", true},
		{"secret", "client_secret", true},
		{"password", "password", true},
		{"normal key", "gist_id", false},
		{"preset key", "default_preset", false},
		{"provider name", "github", false},
		{"empty key", "", false},
	}

	for _, tt := range tests { //nolint:paralleltest // subtests share table/struct state
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
			got := isSensitiveKey(tt.key)
			if got != tt.want {
				t.Errorf("isSensitiveKey(%q) = %v, want %v", tt.key, got, tt.want)
			}
		})
	}
}

// TestRedactJSON_NestedProviderTokens verifies nested provider tokens
// are redacted while non-sensitive values are preserved.
func TestRedactJSON_NestedProviderTokens(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	input := map[string]any{
		"providers": map[string]any{
			"github": map[string]any{
				"token":    "ghp_secret1234567890",
				"gist_id":  "abc123",
				"username": "user1",
			},
		},
		"settings": map[string]any{
			"default_preset": "quick",
		},
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	redacted, err := RedactJSON(data)
	if err != nil {
		t.Fatalf("RedactJSON: %v", err)
	}

	output := string(redacted)

	// Token must be redacted.
	if strings.Contains(output, "ghp_secret1234567890") {
		t.Error("output should NOT contain raw token")
	}
	if !strings.Contains(output, "***7890") {
		t.Error("output should contain '***7890'")
	}
	// Non-sensitive values preserved.
	if !strings.Contains(output, "abc123") {
		t.Error("output should contain gist_id")
	}
	if !strings.Contains(output, "user1") {
		t.Error("output should contain username")
	}
	if !strings.Contains(output, "quick") {
		t.Error("output should contain default_preset")
	}
}
