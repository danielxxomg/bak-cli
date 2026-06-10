// Package components provides reusable, pure render functions for TUI
// components such as menus, checkboxes, radio buttons, and help bars.
// All functions are stateless — they accept input and return a styled string
// using package-level styles from the styles package.
package components

import (
	"strings"

	"github.com/danielxxomg/bak-cli/internal/tui/styles"
)

// RenderMenu renders a vertical menu of items with a cursor indicator (▸)
// at the currently selected position. The cursor item is styled with
// SelectedStyle; all other items are rendered without additional styling.
//
// If cursor is out of bounds, it is clamped to the valid range.
func RenderMenu(items []string, cursor int) string {
	if len(items) == 0 {
		return ""
	}

	// Clamp cursor to valid range.
	if cursor < 0 {
		cursor = 0
	}
	if cursor >= len(items) {
		cursor = len(items) - 1
	}

	var b strings.Builder
	for i, item := range items {
		if i == cursor {
			b.WriteString(styles.CursorIndicator)
			b.WriteString(styles.SelectedStyle.Render(item))
		} else {
			// Two spaces to align with "▸ ".
			b.WriteString("  ")
			b.WriteString(item)
		}
		if i < len(items)-1 {
			b.WriteByte('\n')
		}
	}

	return b.String()
}
