// Package tui provides the root Bubble Tea model and screen routing for
// the bak-cli interactive TUI.
package tui

// RouteSelection dispatches the user's menu selection to the appropriate
// action in Deps. It is a pure function with no TUI dependency — callers
// extract the selection from the model after tea.Program.Run() returns.
//
// An empty selection (Item == "") returns nil immediately. This covers
// the case where menuItems is empty and Selection() returns a zero-value
// MenuSelection.
func RouteSelection(sel MenuSelection, deps Deps) error {
	if sel.Item == "" {
		return nil
	}

	if sel.Cursor == 0 { // "Create backup"
		if deps.RunBackup != nil {
			return deps.RunBackup(nil, nil)
		}
	}

	return nil
}
