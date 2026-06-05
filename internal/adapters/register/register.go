// Package register provides the RegisterAll function that wires up
// all known agent adapters in priority order. It lives in its own
// package to avoid circular imports — the adapter sub-packages import
// the adapters package for the Adapter interface.
package register

import (
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
