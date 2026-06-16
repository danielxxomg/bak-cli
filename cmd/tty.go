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
// program loop. After the TUI exits, it dispatches the user's menu
// selection to the appropriate action via tui.RouteSelection.
func defaultRunTUI(deps tui.Deps) error {
	m := tui.NewModel(deps)
	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("tui: %w", err)
	}
	model, ok := finalModel.(tui.Model)
	if !ok {
		return fmt.Errorf("tui: unexpected model type after exit")
	}
	if err := tui.RouteSelection(model.Selection(), deps); err != nil {
		return fmt.Errorf("tui: %w", err)
	}
	return nil
}
