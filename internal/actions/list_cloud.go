package actions

import (
	"fmt"
	"io"
	"text/tabwriter"
	"time"

	"github.com/danielxxomg/bak-cli/internal/cloud"
	"github.com/danielxxomg/bak-cli/internal/config"
)

// ListCloudAction lists backups from a cloud provider.
// All I/O is injected for testability.
type ListCloudAction struct {
	// Config is the current configuration.
	Config *config.Config

	// Stdout is the writer for formatted output (tabwriter table).
	Stdout io.Writer

	// Stderr is the writer for verbose/diagnostic messages.
	Stderr io.Writer

	// Verbose enables diagnostic output.
	Verbose bool

	// RegistryFactory creates the provider registry.
	// If nil, Run() uses the default registry with all providers.
	RegistryFactory func() *cloud.ProviderRegistry
}

// Run lists backups from the named cloud provider and writes a formatted
// table to Stdout.
func (a *ListCloudAction) Run(providerName string) error {
	cfg := a.Config

	var reg *cloud.ProviderRegistry
	if a.RegistryFactory != nil {
		reg = a.RegistryFactory()
	} else {
		reg = cloud.NewProviderRegistry()

		// Register all available providers (they'll fail at runtime if not configured).
		reg.Register(cloud.NewGitHubGistProvider(cfg, ""))
		reg.Register(cloud.NewGitHubRepoProvider(cfg, "", cfg.Providers["github"].Repo))
		reg.Register(cloud.NewCodebergProvider(cfg, "", cfg.Providers["codeberg"].Repo))
		reg.Register(cloud.NewGiteaProvider(cfg, "", cfg.Providers["gitea"].BaseURL, cfg.Providers["gitea"].Repo))
		reg.Register(cloud.NewRcloneProvider(cfg, cfg.Providers["rclone"].Remote))
		reg.SetDefault("github-gist")
	}

	provider, err := reg.Get(providerName)
	if err != nil {
		return fmt.Errorf("provider: %w", err)
	}

	if a.Verbose {
		fmt.Fprintf(a.Stderr, "Using provider: %s\n", provider.Name())
	}

	backups, err := provider.List()
	if err != nil {
		return fmt.Errorf("list from %s: %w", providerName, err)
	}

	if len(backups) == 0 {
		fmt.Fprintf(a.Stdout, "No backups found on %s.\n", providerName)
		return nil
	}

	w := tabwriter.NewWriter(a.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tDATE\tHOST\tSIZE\tURL")
	fmt.Fprintln(w, "--\t----\t----\t----\t---")

	for _, b := range backups {
		date := b.CreatedAt.Format(time.RFC3339)
		sizeStr := FormatSizeBytes(b.Size)
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			b.ID, date, b.Hostname, sizeStr, b.URL)
	}

	if err := w.Flush(); err != nil {
		return fmt.Errorf("flush output: %w", err)
	}

	return nil
}
