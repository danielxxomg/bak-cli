package components

import "github.com/danielxxomg/bak-cli/internal/tui/styles"

// RenderCheckbox renders a checkbox with the given label, checked state,
// and focus state. Checked items show [x] with ColorPine (green); unchecked
// items show [ ] with ColorMuted. Focused items additionally apply
// SelectedStyle to the entire line.
func RenderCheckbox(label string, checked, focused bool) string {
	var checkbox string
	if checked {
		checkbox = "[x]"
	} else {
		checkbox = "[ ]"
	}

	line := checkbox + " " + label

	switch {
	case focused:
		return styles.SelectedStyle.Render(line)
	case checked:
		return styles.CheckedStyle.Render(checkbox) + " " + label
	default:
		return styles.UncheckedStyle.Render(checkbox) + " " + label
	}
}
