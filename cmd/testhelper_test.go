package cmd

import (
	"bytes"
	"testing"

	"github.com/danielxxomg/bak-cli/internal/config"
)

// Force the whole cmd test package into non-interactive mode.
//
// Without this, any test that reaches an unguarded tea.NewProgram call site
// depends on the host terminal probe. On Linux and macOS CI runners stdin is
// piped, so isTTY() is false and nothing happens. On Windows runners the probe
// can report a terminal, so the same test renders a real TUI and blocks on
// input until the package hits the 10-minute test timeout — which silently
// skipped most of the CI matrix. Tests that specifically exercise an
// interactive path override isTTY back to true.
func init() {
	isTTY = func() bool { return false }
}

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
