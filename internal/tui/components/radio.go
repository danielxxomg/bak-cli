package components

import "github.com/danielxxomg/bak-cli/internal/tui/styles"

// RenderRadio renders a radio button with the given label, selected state,
// and focus state. Selected items show (•) with ColorGold; unselected items
// show ( ) with ColorMuted. Focused items additionally apply SelectedStyle
// to the entire line.
func RenderRadio(label string, selected, focused bool) string {
	var indicator string
	if selected {
		indicator = "(•)"
	} else {
		indicator = "( )"
	}

	line := indicator + " " + label

	switch {
	case focused:
		return styles.SelectedStyle.Render(line)
	case selected:
		return styles.RadioSelectedStyle.Render(indicator) + " " + label
	default:
		return styles.UncheckedStyle.Render(indicator) + " " + label
	}
}
