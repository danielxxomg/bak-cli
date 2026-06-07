// Package configtest provides shared test helpers for isolating config
// lookups (os.UserConfigDir, os.UserHomeDir) during tests.
package configtest

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// SetConfigHome sets OS-specific environment variables so that
// os.UserConfigDir() resolves to a subdirectory of dir on every platform.
//
//	Linux:   sets XDG_CONFIG_HOME
//	Windows: sets APPDATA
//	macOS:   sets HOME and pre-creates Library/Application Support
//
// The function uses t.Setenv so variables are automatically restored
// when the test finishes.
func SetConfigHome(t *testing.T, dir string) {
	t.Helper()
	switch runtime.GOOS {
	case "darwin":
		t.Setenv("HOME", dir)
		// os.UserConfigDir returns $HOME/Library/Application Support;
		// pre-create it so lookups succeed even when the caller does not
		// create the full chain.
		macDir := filepath.Join(dir, "Library", "Application Support")
		if err := os.MkdirAll(macDir, 0755); err != nil {
			t.Fatal(err)
		}
	case "windows":
		t.Setenv("APPDATA", dir)
	default: // linux
		t.Setenv("XDG_CONFIG_HOME", dir)
	}
}
