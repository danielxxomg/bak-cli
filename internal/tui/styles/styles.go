// Package styles provides the Rose Pine color palette and shared lipgloss
// styles for the bak-cli TUI. All styles are package-level variables to
// avoid per-frame allocations during rendering.
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

	// CheckedStyle is used for checked/selected checkboxes.
	CheckedStyle = lipgloss.NewStyle().
			Foreground(ColorPine)

	// UncheckedStyle is used for unchecked checkboxes and unselected radios.
	UncheckedStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)

	// RadioSelectedStyle is used for the selected radio button indicator.
	RadioSelectedStyle = lipgloss.NewStyle().
				Foreground(ColorGold)
)

// ToastStyle is the style for toast notification messages. Toasts appear
// at the bottom-right of the screen and auto-hide after a set duration.
var ToastStyle = lipgloss.NewStyle().
	Foreground(ColorText).
	Padding(0, 1)

// ScreenTitleStyle is the shared style for all screen headings
// (settings, health, shortcuts, cloud).
var ScreenTitleStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(ColorLavender).
	Padding(0, 1)

// SearchStyle is the style for the search bar overlay.
var SearchStyle = lipgloss.NewStyle().
	Foreground(ColorText).
	Padding(0, 1)

// MinWidth is the minimum terminal width (in columns) required to render
// the TUI layout without showing the "Terminal too small" warning.
const MinWidth = 40

// MinHeight is the minimum terminal height (in rows) required to render
// the TUI layout without showing the "Terminal too small" warning.
const MinHeight = 12

// CursorIndicator is the prefix character used to indicate the currently
// selected item in menus and lists.
const CursorIndicator = "\u25b8 "
