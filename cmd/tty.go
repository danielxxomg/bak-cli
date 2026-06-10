package cmd

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/danielxxomg/bak-cli/internal/tui"
)

// runTUI is the injection point for launching the TUI. Tests override
// this variable to verify wiring without launching a real Bubble Tea
// program (which requires a TTY).
var runTUI func(deps tui.Deps) error = defaultRunTUI

// defaultRunTUI creates the root TUI model and runs the Bubble Tea
// program loop. The alternate screen buffer is activated via the model's
// View method (tea.View.AltScreen).
func defaultRunTUI(deps tui.Deps) error {
	m := tui.NewModel(deps)
	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("tui: %w", err)
	}
	return nil
}
