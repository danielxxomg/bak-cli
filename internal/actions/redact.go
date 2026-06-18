// Package actions provides business logic for bak-cli CLI commands.
package actions

import (
	"encoding/json"
	"strings"
)

// SensitiveKeys are the lowercase key substrings that trigger redaction.
var SensitiveKeys = []string{"token", "api_key", "secret", "password"}

// RedactString returns a redacted version of val. Tokens longer than 4
// characters are shown as "***" + last 4 chars; shorter tokens are shown
// as "***" + full value (e.g. "***ab" for "ab").
func RedactString(key, val string) string {
	if val == "" {
		return ""
	}
	if !isSensitiveKey(key) {
		return val
	}
	if len(val) <= 4 {
		return "***" + val
	}
	return "***" + val[len(val)-4:]
}

// RedactJSON walks a JSON object recursively and redacts any value whose
// key name (lowercased) contains one of the SensitiveKeys substrings.
// Returns the redacted JSON as indented bytes.
func RedactJSON(data []byte) ([]byte, error) {
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	redactMap(m)
	return json.MarshalIndent(m, "", "  ")
}

// redactMap recursively walks a map and redacts sensitive values.
func redactMap(m map[string]any) {
	for k, v := range m {
		lower := strings.ToLower(k)
		if isSensitiveKey(lower) {
			if s, ok := v.(string); ok {
				m[k] = RedactString(lower, s)
				continue
			}
		}
		// Recurse into nested maps.
		if nested, ok := v.(map[string]any); ok {
			redactMap(nested)
		}
	}
}

// isSensitiveKey returns true when the lowercased key name contains
// any of the SensitiveKeys substrings.
func isSensitiveKey(key string) bool {
	for _, sk := range SensitiveKeys {
		if strings.Contains(key, sk) {
			return true
		}
	}
	return false
}
