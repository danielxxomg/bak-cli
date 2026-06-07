package actions

import (
	"fmt"

	"github.com/danielxxomg/bak-cli/internal/cloud"
	"github.com/danielxxomg/bak-cli/internal/config"
)

// RealProviderFactory creates cloud providers using the real configuration
// and cloud provider registry. It satisfies the ProviderFactory interface
// so cmd/ wiring can inject it into PushAction/PullAction.
type RealProviderFactory struct {
	// Cfg is the loaded configuration. If nil, Load is called during
	// the first CreateProvider invocation.
	Cfg *config.Config
}

// Compile-time check.
var _ ProviderFactory = (*RealProviderFactory)(nil)

// CreateProvider returns a configured cloud.Provider for the given name.
// It lazily loads config when Cfg is nil.
func (f *RealProviderFactory) CreateProvider(name string) (cloud.Provider, error) {
	cfg := f.Cfg
	if cfg == nil {
		var err error
		cfg, err = config.Load()
		if err != nil {
			return nil, fmt.Errorf("load config: %w", err)
		}
		f.Cfg = cfg
	}

	reg := cloud.NewProviderRegistry()

	// Register all available providers (they'll fail at runtime if not configured).
	reg.Register(cloud.NewGitHubGistProvider(cfg, ""))
	reg.Register(cloud.NewGitHubRepoProvider(cfg, "", cfg.Providers["github"].Repo))
	reg.Register(cloud.NewCodebergProvider(cfg, "", cfg.Providers["codeberg"].Repo))
	reg.Register(cloud.NewGiteaProvider(cfg, "", cfg.Providers["gitea"].BaseURL, cfg.Providers["gitea"].Repo))
	reg.Register(cloud.NewRcloneProvider(cfg, cfg.Providers["rclone"].Remote))
	reg.SetDefault("github-gist")

	return reg.Get(name)
}
