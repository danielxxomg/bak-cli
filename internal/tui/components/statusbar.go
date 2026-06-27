// Package components provides reusable, pure render functions for TUI
// components. statusbar.go renders the persistent one-line status bar shown at
// the bottom of every screen (tui-personality REQ-TP-003).
package components

import (
	"charm.land/lipgloss/v2"

	"github.com/danielxxomg/bak-cli/internal/tui/styles"
)

// statusBarSeparator joins the status bar segments.
const statusBarSeparator = " • "

// statusBarHiddenBelow is the minimum terminal width (in columns) below which
// the status bar is hidden to avoid overflow. Matches the logo threshold.
const statusBarHiddenBelow = 40

// RenderStatusBar renders a one-line status bar containing the version,
// active preset, and backup path. It is a stateless pure function styled with
// the package-level StatusBarStyle (AGENTS.md §styles).
//
// The bar is hidden (returns "") when width is below 40 columns. When the
// backup path is too long to fit, it is truncated with an ellipsis ("…") so
// the whole bar stays within the terminal width.
func RenderStatusBar(width int, version, preset, path string) string {
	if width < statusBarHiddenBelow {
		return ""
	}

	// Leading segment: app name + version + preset.
	left := "bak"
	if version != "" {
		left += " v" + version
	}
	if preset != "" {
		left += statusBarSeparator + preset
	}

	full := left
	if path != "" {
		full += statusBarSeparator + path
	}

	// Fits as-is.
	if lipgloss.Width(full) <= width {
		return styles.StatusBarStyle.Render(full)
	}

	// Doesn't fit: truncate the path segment (spec: path truncated with
	// ellipsis). If there's no path, truncate the leading segment.
	if path == "" {
		return styles.StatusBarStyle.Render(truncateEllipsis(left, width))
	}

	avail := width - lipgloss.Width(left) - len(statusBarSeparator)
	if avail <= 0 {
		// No room for the path at all; show the (possibly truncated) lead.
		return styles.StatusBarStyle.Render(truncateEllipsis(left, width))
	}
	return styles.StatusBarStyle.Render(left + statusBarSeparator + truncateEllipsis(path, avail))
}

// truncateEllipsis truncates s to max visible columns, appending an ellipsis
// ("…") when truncation occurs. It is rune-aware so multi-byte paths truncate
// cleanly. Returns s unchanged when it already fits.
func truncateEllipsis(s string, max int) string {
	if max <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	if max == 1 {
		return "…"
	}
	return string(r[:max-1]) + "…"
}
