// Package register provides the RegisterAll function that wires up
// all known agent adapters in priority order. It lives in its own
// package to avoid circular imports — the adapter sub-packages import
// the adapters package for the Adapter interface.
package register

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/danielxxomg/bak-cli/internal/adapters"
	"github.com/danielxxomg/bak-cli/internal/adapters/claudecode"
	"github.com/danielxxomg/bak-cli/internal/adapters/codex"
	"github.com/danielxxomg/bak-cli/internal/adapters/cursor"
	"github.com/danielxxomg/bak-cli/internal/adapters/kilocode"
	"github.com/danielxxomg/bak-cli/internal/adapters/kiro"
	"github.com/danielxxomg/bak-cli/internal/adapters/opencode"
	"github.com/danielxxomg/bak-cli/internal/adapters/pidev"
	"github.com/danielxxomg/bak-cli/internal/adapters/windsurf"
)

// All registers every known adapter with the provided registry in
// priority order (Claude Code → Cursor → Codex → Windsurf → Kiro →
// KiloCode → pi.dev → OpenCode). Returns the first registration error
// encountered.
func All(r *adapters.Registry) error {
	adapters := []adapters.Adapter{
		&claudecode.Adapter{},
		&cursor.Adapter{},
		&codex.Adapter{},
		&windsurf.Adapter{},
		&kiro.Adapter{},
		&kilocode.Adapter{},
		&pidev.Adapter{},
		&opencode.Adapter{},
	}

	for _, a := range adapters {
		if err := r.Register(a); err != nil {
			return err
		}
	}

	return nil
}

// LoadYAMLAdapters scans the standard YAML adapters directory
// (~/.config/bak/adapters/) for *.yaml adapter definitions and
// registers each one with the provided registry. When override is
// true, custom adapters replace built-in adapters with the same name.
// A warning is emitted to stderr for each overridden adapter.
func LoadYAMLAdapters(reg *adapters.Registry, override bool) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("home dir: %w", err)
	}
	dir := filepath.Join(home, ".config", "bak", "adapters")

	yamlAdapters, err := adapters.LoadYAMLAdapters(dir)
	if err != nil {
		return fmt.Errorf("load yaml adapters: %w", err)
	}

	for _, a := range yamlAdapters {
		if override {
			_, exists := reg.Get(a.Name())
			if exists {
				fmt.Fprintf(os.Stderr, "warning: overriding built-in adapter %q with custom YAML definition\n", a.Name())
			}
		}
		if err := reg.RegisterOrReplace(a, override); err != nil {
			return fmt.Errorf("register adapter %q: %w", a.Name(), err)
		}
	}

	return nil
}
