package actions

import (
	"errors"
	"testing"

	"github.com/danielxxomg/bak-cli/internal/config"
	configtest "github.com/danielxxomg/bak-cli/internal/config/testutil"
)

// TestLoadConfigOr_InjectedLoaderReturns verifies the injected loader's
// config is returned verbatim on success.
func TestLoadConfigOr_InjectedLoaderReturns(t *testing.T) {
	sentinel := &config.Config{}
	got, err := loadConfigOr(func() (*config.Config, error) { return sentinel, nil })
	if err != nil {
		t.Fatalf("loadConfigOr: %v", err)
	}
	if got != sentinel {
		t.Errorf("loadConfigOr returned %p, want sentinel %p", got, sentinel)
	}
}

// TestLoadConfigOr_NilFallsBackToConfigLoad verifies a nil loader delegates
// to config.Load (isolated config home so the real user config is not read).
func TestLoadConfigOr_NilFallsBackToConfigLoad(t *testing.T) {
	configtest.SetConfigHome(t, t.TempDir())
	got, err := loadConfigOr(nil)
	if err != nil {
		t.Fatalf("loadConfigOr(nil): %v", err)
	}
	if got == nil {
		t.Fatal("loadConfigOr(nil) returned nil config, want config.Load default")
	}
}

// TestLoadConfigOr_ErrorPropagated verifies a loader error is propagated.
func TestLoadConfigOr_ErrorPropagated(t *testing.T) {
	sentinel := errors.New("config corrupted")
	_, err := loadConfigOr(func() (*config.Config, error) { return nil, sentinel })
	if !errors.Is(err, sentinel) {
		t.Errorf("loadConfigOr error = %v, want %v", err, sentinel)
	}
}
