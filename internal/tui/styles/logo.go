package styles

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// asciiLogo holds the ASCII art logo for bak-cli.
// 5 lines tall, ~28 chars wide. Rendered with a 5-band Rose Pine gradient.
const asciiLogo = `  ____    _    _  __
 |  _ \  / \  | |/ /
 | |_) |/ _ \ | ' / 
 |  _ < / ___ \| . \ 
 |_| \_/_/   \_\_|\_\`

// RenderLogo returns the ASCII art "bak" logo with a 5-band Rose Pine
// gradient applied. The gradient runs from Love (top) through Gold, Rose,
// Pine, to Lavender (bottom).
//
// If the terminal width is less than 40 columns, an empty string is returned
// to prevent overflow.
func RenderLogo(width int) string {
	if width < 40 {
		return ""
	}

	lines := strings.Split(asciiLogo, "\n")
	if len(lines) == 0 {
		return ""
	}

	// 5-band gradient: Love → Gold → Rose → Pine → Lavender
	bandColors := []lipgloss.Style{
		lipgloss.NewStyle().Foreground(ColorLove),
		lipgloss.NewStyle().Foreground(ColorGold),
		lipgloss.NewStyle().Foreground(ColorRose),
		lipgloss.NewStyle().Foreground(ColorPine),
		lipgloss.NewStyle().Foreground(ColorLavender),
	}

	var b strings.Builder
	for i, line := range lines {
		if i >= len(bandColors) {
			break
		}
		styled := bandColors[i].Render(line)
		b.WriteString(styled)
		if i < len(lines)-1 {
			b.WriteByte('\n')
		}
	}

	return b.String()
}
