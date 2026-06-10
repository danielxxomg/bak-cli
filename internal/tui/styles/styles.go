package styles

import "charm.land/lipgloss/v2"

// Package-level style variables. All styles use the Rose Pine palette and are
// defined at package scope to avoid per-frame allocations during TUI rendering.
//
// Rules (enforced by AGENTS.md):
//   - MUST use package-level var for all lipgloss styles
//   - MUST NOT use inline lipgloss.NewStyle() in View() methods
var (
	// TitleStyle is used for screen titles and the main heading.
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorLavender)

	// HeadingStyle is used for section headings within screens.
	HeadingStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorGold)

	// SelectedStyle is used for the currently focused/selected item.
	SelectedStyle = lipgloss.NewStyle().
			Foreground(ColorRose)

	// FrameStyle is the base style for bordered content.
	// Use Frame() to apply this style with a DoubleBorder.
	FrameStyle = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(ColorMuted)

	// PanelStyle is used for content panels with a subtle single border.
	PanelStyle = lipgloss.NewStyle().
			Padding(1).
			Border(lipgloss.NormalBorder()).
			BorderForeground(ColorOverlay)

	// HelpStyle is used for help bars and footer text.
	HelpStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)
)

// CursorIndicator is the prefix character used to indicate the currently
// selected item in menus and lists.
const CursorIndicator = "\u25b8 "
