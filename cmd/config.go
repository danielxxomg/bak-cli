package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/danielxxomg/bak-cli/internal/actions"
)

// configCmd is the parent command for configuration management.
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "View and modify bak-cli configuration",
	Long: `Manage your bak-cli configuration from the command line.

The 'show' subcommand prints your current configuration with tokens
redacted for security. Use 'get' to read a specific value and 'set'
to update it.

Dotted-key syntax:
  providers.<name>.<field>  (token, gist_id, repo, remote, base_url)
  settings.<field>          (default_preset, auto_sync, max_file_size, ...)

Examples:
  bak config show
  bak config get providers.github.token
  bak config set providers.codeberg.token <your-token>
  bak config set settings.default_preset full`,
}

// --- config show ---

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Display current configuration (tokens redacted)",
	Long:  "Print the full bak-cli configuration as JSON with all tokens, secrets, and passwords redacted.",
	Args:  cobra.NoArgs,
	RunE:  runConfigShow,
}

func init() {
	configCmd.AddCommand(configShowCmd)
}

func runConfigShow(cmd *cobra.Command, args []string) error {
	return runConfigShowWithDeps(cmd, args, depsFromCmd(cmd))
}

func runConfigShowWithDeps(cmd *cobra.Command, args []string, deps cmdDeps) error {
	cfg, err := deps.ConfigLoader()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	return actions.ConfigShow(cfg, deps.Stdout)
}

// --- config get ---

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a specific configuration value (tokens redacted)",
	Long: `Retrieve a single configuration value by dotted key.

Supported keys:
  providers.<name>.token
  providers.<name>.gist_id
  settings.default_preset
  settings.auto_sync
  settings.max_file_size
  settings.verbose_default
  settings.default_provider

Token values are redacted in the output for security.`,
	Args: cobra.ExactArgs(1),
	RunE: runConfigGet,
}

func init() {
	configCmd.AddCommand(configGetCmd)
}

func runConfigGet(cmd *cobra.Command, args []string) error {
	return runConfigGetWithDeps(cmd, args, depsFromCmd(cmd))
}

func runConfigGetWithDeps(cmd *cobra.Command, args []string, deps cmdDeps) error {
	cfg, err := deps.ConfigLoader()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	return actions.ConfigGet(cfg, args[0], deps.Stdout)
}

// --- config set ---

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Long: `Update a single configuration value and save to disk.

Supported keys:
  providers.<name>.token
  providers.<name>.gist_id
  providers.<name>.repo
  providers.<name>.remote
  providers.<name>.base_url
  settings.default_preset   (quick, full, skills)
  settings.auto_sync        (true/false)
  settings.max_file_size    (bytes, e.g. 1048576)
  settings.verbose_default  (true/false)
  settings.default_provider (provider name)

Examples:
  bak config set providers.codeberg.token ghp_YOURTOKENHERE
  bak config set settings.default_preset full
  bak config set settings.auto_sync true`,
	Args: cobra.ExactArgs(2),
	RunE: runConfigSet,
}

func init() {
	configCmd.AddCommand(configSetCmd)
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	return runConfigSetWithDeps(cmd, args, depsFromCmd(cmd))
}

func runConfigSetWithDeps(cmd *cobra.Command, args []string, deps cmdDeps) error {
	cfg, err := deps.ConfigLoader()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	return actions.ConfigSet(cfg, args[0], args[1], deps.Stdout)
}

func init() {
	rootCmd.AddCommand(configCmd)
}
