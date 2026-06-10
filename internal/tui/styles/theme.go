// Package styles provides the Rose Pine theme system and reusable lipgloss
// styles for the bak-cli TUI. All styles are defined as package-level variables
// to avoid per-frame allocations during rendering.
//
// Colors are sourced from https://rosepinetheme.com/palette/ using the
// Rose Pine (main) variant hex codes.
package styles

import "charm.land/lipgloss/v2"

// Rose Pine semantic colors.
//
// Each color maps to a specific role in the UI:
//   - Base, Surface, Overlay: background layers from darkest to lightest
//   - Muted, Subtle: secondary/de-emphasized text
//   - Text: primary foreground text
//   - Love, Gold, Rose, Pine, Lavender: accent/highlight colors
var (
	// ColorBase is the darkest background color.
	ColorBase = lipgloss.Color("#191724")
	// ColorSurface is a slightly lighter background color used for panels.
	ColorSurface = lipgloss.Color("#1f1d2e")
	// ColorOverlay is the lightest background color used for overlays and borders.
	ColorOverlay = lipgloss.Color("#26233a")
	// ColorMuted is used for de-emphasized text (comments, hints, help text).
	ColorMuted = lipgloss.Color("#6e6a86")
	// ColorSubtle is used for secondary text (labels, descriptions).
	ColorSubtle = lipgloss.Color("#908caa")
	// ColorText is the primary foreground color.
	ColorText = lipgloss.Color("#e0def4")
	// ColorLove is a pinkish-red accent used for destructive actions and errors.
	ColorLove = lipgloss.Color("#eb6f92")
	// ColorGold is a yellow accent used for highlights and headings.
	ColorGold = lipgloss.Color("#f6c177")
	// ColorRose is a warm pink accent used for selected items and focus states.
	ColorRose = lipgloss.Color("#ebbcba")
	// ColorPine is a teal-green accent used for success states and positive actions.
	ColorPine = lipgloss.Color("#31748f")
	// ColorLavender is a purple accent used for titles and primary actions.
	ColorLavender = lipgloss.Color("#c4a7e7")
)
