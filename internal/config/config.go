// Package config manages the bak CLI configuration file stored at
// ~/.config/bak/config.json.
//
// The config file holds user preferences, credentials for cloud-sync
// providers (e.g., GitHub Gist, Codeberg, Rclone), and encryption
// profile settings.
//
// Schema versioning:
//   - v0.1.0: flat github_token + gist_id at root (legacy)
//   - v0.2.0: schema_version + nested providers
//   - v0.3.0: schema_version + nested providers + profiles (current)
//
// LoadPath() auto-detects older configs and migrates them in a chain,
// preserving backups at each step (config.json.v010.bak, .v020.bak).
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
	SchemaVersion string                    `json:"schema_version,omitempty"`
	GitHubToken   string                    `json:"github_token,omitempty"`
	GistID        string                    `json:"gist_id,omitempty"`
	Providers     map[string]ProviderConfig `json:"providers,omitempty"`
	Profiles      map[string]ProfileConfig  `json:"profiles,omitempty"`

	// path is the on-disk location of this config file (not serialized).
	path string `json:"-"`
}

// ProviderConfig holds settings for a single cloud provider.
type ProviderConfig struct {
	Token   string `json:"token,omitempty"`
	GistID  string `json:"gist_id,omitempty"`  // github-gist only
	Repo    string `json:"repo,omitempty"`     // github-repo, codeberg, gitea
	Remote  string `json:"remote,omitempty"`   // rclone remote name
	BaseURL string `json:"base_url,omitempty"` // gitea/forgejo custom URL
}

// EncryptionConfig holds the encryption settings for a named profile.
type EncryptionConfig struct {
	Enabled     bool   `json:"enabled,omitempty"`
	Password    string `json:"password,omitempty"`
	Iterations  int    `json:"iterations,omitempty"`
	MemoryKiB   int    `json:"memory_kib,omitempty"`
	Parallelism int    `json:"parallelism,omitempty"`
}

// ScheduleConfig holds OS-native backup scheduling settings for a profile.
// When Enabled is true, the profile has an active scheduled task managed by
// the OS-native scheduler (crontab on Unix, schtasks on Windows).
type ScheduleConfig struct {
	Enabled  bool   `json:"enabled,omitempty"`
	Interval string `json:"interval,omitempty"` // daily, weekly, every-12h, every-6h
}

// ProfileConfig holds the configuration for a named backup profile.
type ProfileConfig struct {
	Adapters   []string          `json:"adapters,omitempty"`
	Categories []string          `json:"categories,omitempty"`
	Preset     string            `json:"preset,omitempty"`
	Provider   string            `json:"provider,omitempty"`
	Encryption *EncryptionConfig `json:"encryption,omitempty"`
	Schedule   *ScheduleConfig   `json:"schedule,omitempty"`
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
// If the config is in v0.1.0 format (flat github_token + gist_id at root,
// no schema_version), it auto-migrates to v0.2.0 and writes a .v010.bak
// backup. If the config is v0.2.0, it migrates to v0.3.0 (adding the
// profiles map) with a .v020.bak backup. Migrations are chained.
func LoadPath(cfgPath string) (*Config, error) {
	cfg := &Config{path: cfgPath}

	data, err := os.ReadFile(cfgPath)
	if os.IsNotExist(err) {
		return cfg, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	// Attempt normal unmarshal first.
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	cfg.path = cfgPath

	// Detect v0.1.0: has github_token or gist_id at root AND no schema_version.
	if isV010(cfg) {
		if err := migrateV010(cfg, data); err != nil {
			return nil, fmt.Errorf("migrate config: %w", err)
		}
	}

	// Detect v0.2.0 (runs after potential v0.1.0 → v0.2.0 migration).
	if isV020(cfg) {
		if err := migrateV020(cfg); err != nil {
			return nil, fmt.Errorf("migrate config: %w", err)
		}
	}

	return cfg, nil
}

// isV010 returns true if the config appears to be v0.1.0 format:
// has github_token or gist_id at root level and no schema_version.
func isV010(cfg *Config) bool {
	if cfg.SchemaVersion != "" {
		return false
	}
	return cfg.GitHubToken != "" || cfg.GistID != ""
}

// migrateV010 transforms a v0.1.0 flat config into v0.2.0 nested format.
// Writes config.json.v010.bak before overwriting.
func migrateV010(cfg *Config, original []byte) error {
	bakPath := cfg.path + ".v010.bak"
	//nolint:gosec // G703: bakPath is derived from cfg.path (trusted config loader path)
	if err := os.WriteFile(bakPath, original, 0600); err != nil {
		return fmt.Errorf("write backup: %w", err)
	}

	// Move flat fields into providers.github.
	if cfg.Providers == nil {
		cfg.Providers = make(map[string]ProviderConfig)
	}
	githubCfg := cfg.Providers["github"]
	if cfg.GitHubToken != "" && githubCfg.Token == "" {
		githubCfg.Token = cfg.GitHubToken
	}
	if cfg.GistID != "" && githubCfg.GistID == "" {
		githubCfg.GistID = cfg.GistID
	}
	cfg.Providers["github"] = githubCfg

	// Clear compat shim fields.
	cfg.GitHubToken = ""
	cfg.GistID = ""

	// Set schema version.
	cfg.SchemaVersion = "0.2.0"

	// Persist the migrated config.
	return cfg.Save()
}

// isV020 returns true if the config is v0.2.0 format (schema_version == "0.2.0").
func isV020(cfg *Config) bool {
	return cfg.SchemaVersion == "0.2.0"
}

// migrateV020 transforms a v0.2.0 config into v0.3.0 by adding the
// profiles map and bumping the schema version. Writes config.json.v020.bak
// before overwriting.
func migrateV020(cfg *Config) error {
	// Read current v0.2.0 config from disk for backup.
	current, err := os.ReadFile(cfg.path)
	if err != nil {
		return fmt.Errorf("read config for backup: %w", err)
	}

	bakPath := cfg.path + ".v020.bak"
	//nolint:gosec // G703: bakPath is derived from cfg.path (trusted config loader path)
	if err := os.WriteFile(bakPath, current, 0600); err != nil {
		return fmt.Errorf("write backup: %w", err)
	}

	// Add profiles map if not present.
	if cfg.Profiles == nil {
		cfg.Profiles = make(map[string]ProfileConfig)
	}

	// Bump schema version.
	cfg.SchemaVersion = "0.3.0"

	return cfg.Save()
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
//
// Legacy flat keys (backward-compatible):
//   - "github.token" → Providers["github"].Token (or compat GitHubToken)
//   - "github.gist_id" → Providers["github"].GistID (or compat GistID)
//
// Nested provider keys:
//   - "providers.github.token" → Providers["github"].Token
//   - "providers.github.gist_id" → Providers["github"].GistID
//   - "providers.codeberg.token" → Providers["codeberg"].Token
//   - etc.
//
// Returns an error for unknown keys.
func (c *Config) Get(key string) (string, error) {
	// Check nested providers first.
	if provider, subkey, ok := parseNestedKey(key); ok {
		pc, exists := c.Providers[provider]
		if !exists {
			return "", fmt.Errorf("unknown config key: %q (provider %q not configured)", key, provider)
		}
		switch subkey {
		case "token":
			return pc.Token, nil
		case "gist_id":
			return pc.GistID, nil
		case "repo":
			return pc.Repo, nil
		case "remote":
			return pc.Remote, nil
		case "base_url":
			return pc.BaseURL, nil
		default:
			return "", fmt.Errorf("unknown config key: %q (unsupported field %q)", key, subkey)
		}
	}

	// Legacy flat keys with compat shim.
	switch key {
	case "github.token":
		if c.Providers != nil {
			if pc, ok := c.Providers["github"]; ok && pc.Token != "" {
				return pc.Token, nil
			}
		}
		return c.GitHubToken, nil
	case "github.gist_id":
		if c.Providers != nil {
			if pc, ok := c.Providers["github"]; ok && pc.GistID != "" {
				return pc.GistID, nil
			}
		}
		return c.GistID, nil
	default:
		return "", fmt.Errorf("unknown config key: %q", key)
	}
}

// Set updates a value by key and persists to disk.
// Supports the same keys as Get().
func (c *Config) Set(key, value string) error {
	// Check nested providers first.
	if provider, subkey, ok := parseNestedKey(key); ok {
		if c.Providers == nil {
			c.Providers = make(map[string]ProviderConfig)
		}
		pc := c.Providers[provider]
		switch subkey {
		case "token":
			pc.Token = value
		case "gist_id":
			pc.GistID = value
		case "repo":
			pc.Repo = value
		case "remote":
			pc.Remote = value
		case "base_url":
			pc.BaseURL = value
		default:
			return fmt.Errorf("unknown config key: %q (unsupported field %q)", key, subkey)
		}
		c.Providers[provider] = pc
		return c.Save()
	}

	// Legacy flat keys.
	switch key {
	case "github.token":
		c.GitHubToken = value
		// Also set in providers for forward compat.
		if c.Providers == nil {
			c.Providers = make(map[string]ProviderConfig)
		}
		pc := c.Providers["github"]
		pc.Token = value
		c.Providers["github"] = pc
	case "github.gist_id":
		c.GistID = value
		if c.Providers == nil {
			c.Providers = make(map[string]ProviderConfig)
		}
		pc := c.Providers["github"]
		pc.GistID = value
		c.Providers["github"] = pc
	default:
		return fmt.Errorf("unknown config key: %q", key)
	}
	return c.Save()
}

// parseNestedKey splits "providers.<name>.<field>" into provider name and field.
// Returns (name, field, true) on success, or ("", "", false) if the key
// does not match the nested pattern.
func parseNestedKey(key string) (provider, field string, ok bool) {
	const prefix = "providers."
	if len(key) <= len(prefix) || key[:len(prefix)] != prefix {
		return "", "", false
	}
	rest := key[len(prefix):]
	for i := 0; i < len(rest); i++ {
		if rest[i] == '.' {
			return rest[:i], rest[i+1:], true
		}
	}
	return "", "", false
}
