package screens

import (
	"strings"

	"github.com/danielxxomg/bak-cli/internal/tui/styles"
)

// shortcutsEntry represents a single keybinding row in the shortcut overlay.
type shortcutsEntry struct {
	key  string
	desc string
}

// RenderShortcuts renders a categorized keyboard shortcut reference overlay.
// Keybindings are grouped into Navigation, Actions, Screens, and Meta sections.
// Each group heading uses HeadingStyle; individual key names use SelectedStyle.
// On terminals wider than 50 columns, the content is wrapped in a Frame.
func RenderShortcuts(width int) string { //nolint:funlen // static key-bindings table
	groups := []struct {
		heading string
		entries []shortcutsEntry
	}{
		{
			heading: "Navigation",
			entries: []shortcutsEntry{
				{key: "j / \u2193", desc: "move down"},
				{key: "k / \u2191", desc: "move up"},
				{key: keyEnter, desc: keySelect},
			},
		},
		{
			heading: "Actions",
			entries: []shortcutsEntry{
				{key: keySpace, desc: keyToggle},
				{key: "/", desc: "search"},
			},
		},
		{
			heading: "Screens",
			entries: []shortcutsEntry{
				{key: "1", desc: "Menu"},
				{key: "2", desc: "Dashboard"},
				{key: "3", desc: "Settings"},
				{key: "4", desc: "Progress"},
				{key: "5", desc: "Health"},
				{key: "6", desc: "Cloud"},
				{key: "7", desc: "Wizard"},
			},
		},
		{
			heading: "Meta",
			entries: []shortcutsEntry{
				{key: "?", desc: "shortcuts"},
				{key: "q", desc: keyQuit},
				{key: "esc", desc: keyBack},
			},
		},
	}

	var b strings.Builder

	for gi, group := range groups {
		// Group heading.
		b.WriteString(styles.HeadingStyle.Render(group.heading))
		b.WriteString("\n")

		for _, entry := range group.entries {
			b.WriteString("  ")
			b.WriteString(styles.SelectedStyle.Render(entry.key))
			b.WriteString(" ")
			b.WriteString(entry.desc)
			b.WriteString("\n")
		}

		if gi < len(groups)-1 {
			b.WriteString("\n")
		}
	}

	content := b.String()
	if width >= 50 {
		content = styles.Frame(content, width-4)
	}
	return content
}
