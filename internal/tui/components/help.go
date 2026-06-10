package components

import (
	"strings"

	"github.com/danielxxomg/bak-cli/internal/tui/styles"
)

// HelpKey represents a keybinding and its description for the help bar.
type HelpKey struct {
	// Key is the keyboard shortcut (e.g., "↑/↓", "enter", "q").
	Key string
	// Desc is a short description of the action (e.g., "navigate", "quit").
	Desc string
}

// RenderHelp renders a footer-style help bar with key-description pairs
// separated by " • ". The output uses HelpStyle from the styles package.
func RenderHelp(keys []HelpKey) string {
	if len(keys) == 0 {
		return ""
	}

	var parts []string
	for _, k := range keys {
		parts = append(parts, k.Key+" "+k.Desc)
	}

	line := strings.Join(parts, " • ")
	return styles.HelpStyle.Render(line)
}
