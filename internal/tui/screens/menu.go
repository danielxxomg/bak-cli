// Package screens provides screen-specific render functions for the
// bak-cli TUI. Screens compose shared components and styles into full
// terminal views. Complex screens use bubbletea sub-models (PR4+).
package screens

import (
	"strings"

	"github.com/danielxxomg/bak-cli/internal/tui/components"
	"github.com/danielxxomg/bak-cli/internal/tui/styles"
)

// RenderMainMenu composes the main menu screen from the logo, version
// subtitle, menu items with cursor, and context-specific help bar.
//
// The logo is omitted when width < 40 to prevent overflow on narrow
// terminals. Menu navigation keys are rendered via components.RenderMenu
// and the help bar via components.RenderHelp.
func RenderMainMenu(version string, banner string, menuItems []string, cursor int, width int) string {
	var b strings.Builder

	// Logo (only on wide terminals).
	logo := styles.RenderLogo(width)
	if logo != "" {
		b.WriteString(logo)
		b.WriteString("\n\n")
	}

	// Version subtitle.
	if version != "" {
		b.WriteString(styles.TitleStyle.Render("bak v" + version))
		b.WriteString("\n\n")
	}

	// Banner (optional, used for first-run or special messages).
	if banner != "" {
		b.WriteString(styles.HeadingStyle.Render(banner))
		b.WriteString("\n\n")
	}

	// Menu items with cursor.
	b.WriteString(components.RenderMenu(menuItems, cursor))
	b.WriteString("\n\n")

	// Help bar with contextual keys.
	helpKeys := []components.HelpKey{
		{Key: "\u2191/\u2193", Desc: "navigate"},
		{Key: "enter", Desc: "select"},
		{Key: "q", Desc: "quit"},
	}
	b.WriteString(components.RenderHelp(helpKeys))

	return b.String()
}
