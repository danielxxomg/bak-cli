// Package components provides reusable, pure render functions for TUI
// components. empty_state.go renders the styled empty-state block shown when a
// screen has no data (tui-personality REQ-TP-007).
package components

import (
	"strings"

	"github.com/danielxxomg/bak-cli/internal/tui/styles"
)

// RenderEmptyState renders a styled empty-state block composed of an icon, an
// italic message, and a hint describing the next action. It is a stateless
// pure function styled with the package-level EmptyState styles (AGENTS.md
// styles section). All three segments are optional; when all are empty it
// returns the empty string.
//
// Example: RenderEmptyState(icon, "No backups yet", "Run bak backup to create one")
func RenderEmptyState(icon, message, hint string) string {
	if icon == "" && message == "" && hint == "" {
		return ""
	}

	var b strings.Builder
	switch {
	case icon != "" && message != "":
		b.WriteString(styles.EmptyStateIconStyle.Render(icon))
		b.WriteString(" ")
		b.WriteString(styles.EmptyStateMsgStyle.Render(message))
	case icon != "":
		b.WriteString(styles.EmptyStateIconStyle.Render(icon))
	default:
		b.WriteString(styles.EmptyStateMsgStyle.Render(message))
	}

	if hint != "" {
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString(styles.EmptyStateHintStyle.Render(hint))
	}

	return b.String()
}
