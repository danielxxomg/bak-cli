package cmd

import (
	"bytes"
	"testing"

	"github.com/danielxxomg/bak-cli/internal/config"
)

// setupTestDeps returns injectable cmdDeps with mock config and buffer I/O.
// Tests call runXWithDeps directly with these deps to isolate from real config.
func setupTestDeps(t *testing.T) (cmdDeps, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	deps := cmdDeps{
		ConfigLoader: func() (*config.Config, error) {
			return &config.Config{
				Providers: map[string]config.ProviderConfig{},
				Profiles:  map[string]config.ProfileConfig{},
			}, nil
		},
		Stdout: stdout,
		Stderr: stderr,
		Stdin:  &bytes.Buffer{},
	}
	return deps, stdout, stderr
}
