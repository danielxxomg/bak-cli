package actions

// configLoad bridges the RealConfigLoader to the config package without an
// import cycle. It is assigned in os_impl_config.go (a separate file that
// imports config.Config).
var configLoad = defaultConfigLoad

// defaultConfigLoad returns an empty Config, used as fallback until the
// real bridge is wired in Phase 3.
func defaultConfigLoad() (*Config, error) {
	return &Config{}, nil
}
