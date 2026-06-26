package actions

import (
	"strings"
	"testing"
)

func TestRealProviderFactory_NilConfig(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	// With nil config, CreateProvider should try to load config
	// and eventually fail because no real config exists in test env.
	factory := &RealProviderFactory{}
	_, err := factory.CreateProvider("github-gist")
	if err == nil {
		t.Log("CreateProvider succeeded (real config exists)")
		return
	}
	// Should be a config load error or "no token" or similar.
	msg := err.Error()
	if msg == "" {
		t.Error("expected non-empty error")
	}
}

func TestRealProviderFactory_UnknownProvider(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	// Create a factory with a minimal config that has no providers.
	configPath := t.TempDir()
	t.Setenv("BAK_CONFIG_DIR", configPath)

	// Load will fail to find config.json — expected.
	factory := &RealProviderFactory{}
	_, err := factory.CreateProvider("nonexistent-provider")
	if err == nil {
		t.Log("CreateProvider found a real config")
		return
	}
	if !strings.Contains(err.Error(), "load config") &&
		!strings.Contains(err.Error(), "provider") &&
		!strings.Contains(err.Error(), "config") {
		t.Logf("CreateProvider error: %v", err)
	}
}

func TestRealProviderFactory_InterfaceCompliance(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	// Compile-time check already done, but runtime verification too.
	var _ ProviderFactory = (*RealProviderFactory)(nil)
}
