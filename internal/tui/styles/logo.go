package styles

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
)

// asciiLogo holds the ASCII art logo for bak-cli.
// 5 lines tall, ~28 chars wide. Rendered with a 5-band Rose Pine gradient.
const asciiLogo = `  ____    _    _  __
  |  _ \  / \  | |/ /
  | |_) |/ _ \ | ' / 
  |  _ < / ___ \| . \ 
  |_| \_/_/   \_\_|\_\`

// colorProfile controls whether RenderLogo emits ANSI color. It defaults to
// TrueColor so the gradient renders on color terminals. On no-color profiles
// (NoTTY/Ascii) RenderLogo falls back to plain text without ANSI codes
// (REQ-TP-004 §"plain logo on no-color terminal").
//
// Production terminal profile detection is handled by the bubbletea renderer,
// which downsamples the full View to the terminal's profile; this variable
// exists so the logo's explicit no-color branch is unit-testable as a pure
// function (tests in this package override it and restore it).
var colorProfile = colorprofile.TrueColor

// RenderLogo returns the ASCII art "bak" logo with a Rose Pine multi-stop
// vertical gradient (Love → Gold → Rose → Pine → Lavender) applied via
// lipgloss.Blend1D, one gradient color per logo line.
//
// On a no-color profile (NoTTY/Ascii) the logo falls back to uncolored plain
// text. If the terminal width is less than 40 columns, an empty string is
// returned to prevent overflow (existing behavior preserved).
func RenderLogo(width int) string {
	if width < 40 { //nolint:mnd // logo hide threshold, matches status bar
		return ""
	}

	lines := strings.Split(asciiLogo, "\n")
	if len(lines) == 0 {
		return ""
	}

	// No-color profile: monochrome plain text, no ANSI codes.
	if colorProfile <= colorprofile.Ascii {
		return strings.Join(lines, "\n")
	}

	// 5-stop Rose Pine gradient, one color per line. With 5 lines and 5 stops
	// Blend1D returns the stops verbatim; adding lines or stops would smoothly
	// interpolate between them in CIELAB space.
	grad := lipgloss.Blend1D(len(lines), ColorLove, ColorGold, ColorRose, ColorPine, ColorLavender)

	var b strings.Builder
	for i, line := range lines {
		color := grad[i%len(grad)]
		b.WriteString(lipgloss.NewStyle().Foreground(color).Render(line))
		if i < len(lines)-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}
