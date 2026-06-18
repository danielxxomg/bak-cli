// Package actions provides business logic for bak-cli CLI commands.
package actions

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/danielxxomg/bak-cli/internal/config"
)

// ConfigShow marshals the config to indented JSON, redacts sensitive
// values, and writes the result to out. When cfg is nil, it prints a
// helpful message indicating no config file exists.
func ConfigShow(cfg *config.Config, out io.Writer) error {
	if cfg == nil {
		_, err := fmt.Fprintln(out, "No configuration found. Run 'bak login' to get started.")
		return err
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	redacted, err := RedactJSON(data)
	if err != nil {
		return fmt.Errorf("redact config: %w", err)
	}
	_, err = fmt.Fprintln(out, string(redacted))
	return err
}

// ConfigGet retrieves a config value by dotted key and writes it to out.
// Sensitive values (token, api_key, secret, password) are redacted.
func ConfigGet(cfg *config.Config, key string, out io.Writer) error {
	if cfg == nil {
		return fmt.Errorf("no configuration file found — cannot get %q", key)
	}
	val, err := cfg.Get(key)
	if err != nil {
		return err
	}
	// Redact sensitive values on output (per security rules).
	redacted := RedactString(key, val)
	_, err = fmt.Fprintln(out, redacted)
	return err
}

// ConfigSet sets a config value by dotted key and persists to disk.
func ConfigSet(cfg *config.Config, key, value string, out io.Writer) error {
	if cfg == nil {
		return fmt.Errorf("no configuration file found — cannot set %q", key)
	}
	if err := cfg.Set(key, value); err != nil {
		return fmt.Errorf("set %s: %w", key, err)
	}
	_, err := fmt.Fprintf(out, "Saved %s = %s\n", key, RedactString(key, value))
	return err
}
