// Package config manages the bak CLI configuration file stored at
// ~/.config/bak/config.json.
//
// The config file holds user preferences and credentials (e.g., GitHub
// token) used by the cloud-sync commands.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/danielxxomg/bak-cli/internal/paths"
)

// Config represents the persistent bak CLI configuration.
type Config struct {
	GitHubToken string `json:"github_token,omitempty"`

	// path is the on-disk location of this config file (not serialized).
	path string `json:"-"`
}

// DefaultPath returns the canonical path to the config file.
func DefaultPath() (string, error) {
	dir, err := paths.ConfigDir("bak")
	if err != nil {
		return "", fmt.Errorf("config dir: %w", err)
	}
	return filepath.Join(dir, "config.json"), nil
}

// Load reads the config from ~/.config/bak/config.json.
// Returns a default (empty) config if the file does not exist.
func Load() (*Config, error) {
	cfgPath, err := DefaultPath()
	if err != nil {
		return nil, err
	}
	return LoadPath(cfgPath)
}

// LoadPath reads config from an explicit path.
func LoadPath(cfgPath string) (*Config, error) {
	cfg := &Config{path: cfgPath}

	data, err := os.ReadFile(cfgPath)
	if os.IsNotExist(err) {
		return cfg, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	cfg.path = cfgPath
	return cfg, nil
}

// Save writes the config to disk, creating parent directories as needed.
func (c *Config) Save() error {
	if c.path == "" {
		var err error
		c.path, err = DefaultPath()
		if err != nil {
			return err
		}
	}

	dir := filepath.Dir(c.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(c.path, data, 0600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

// Get returns a value by key. Supported keys:
//   - "github.token" → c.GitHubToken
//
// Returns an error for unknown keys.
func (c *Config) Get(key string) (string, error) {
	switch key {
	case "github.token":
		return c.GitHubToken, nil
	default:
		return "", fmt.Errorf("unknown config key: %q", key)
	}
}

// Set updates a value by key and persists to disk. Supported keys:
//   - "github.token" → c.GitHubToken
func (c *Config) Set(key, value string) error {
	switch key {
	case "github.token":
		c.GitHubToken = value
	default:
		return fmt.Errorf("unknown config key: %q", key)
	}
	return c.Save()
}
