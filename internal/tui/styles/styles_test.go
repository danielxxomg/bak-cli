package styles

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

// TestColors verifies all 11 Rose Pine semantic colors exist as package-level
// variables and produce valid ANSI true-color sequences when rendered.
// Lipgloss converts hex colors to RGB decimal ANSI sequences: e.g. #191724 → 38;2;25;23;36.
func TestColors(t *testing.T) {
	tests := []struct {
		name string
		// applyColor returns a style with the color as foreground.
		applyColor func() lipgloss.Style
		// rgbSeq is the expected ANSI true-color sequence for this color (38;2;R;G;B).
		rgbSeq string
	}{
		{"Base", func() lipgloss.Style { return lipgloss.NewStyle().Foreground(ColorBase) }, "38;2;25;23;36"},
		{"Surface", func() lipgloss.Style { return lipgloss.NewStyle().Foreground(ColorSurface) }, "38;2;31;29;46"},
		{"Overlay", func() lipgloss.Style { return lipgloss.NewStyle().Foreground(ColorOverlay) }, "38;2;38;35;58"},
		{"Muted", func() lipgloss.Style { return lipgloss.NewStyle().Foreground(ColorMuted) }, "38;2;110;106;134"},
		{"Subtle", func() lipgloss.Style { return lipgloss.NewStyle().Foreground(ColorSubtle) }, "38;2;144;140;170"},
		{"Text", func() lipgloss.Style { return lipgloss.NewStyle().Foreground(ColorText) }, "38;2;224;222;244"},
		{"Love", func() lipgloss.Style { return lipgloss.NewStyle().Foreground(ColorLove) }, "38;2;235;111;146"},
		{"Gold", func() lipgloss.Style { return lipgloss.NewStyle().Foreground(ColorGold) }, "38;2;246;193;119"},
		{"Rose", func() lipgloss.Style { return lipgloss.NewStyle().Foreground(ColorRose) }, "38;2;235;188;186"},
		{"Pine", func() lipgloss.Style { return lipgloss.NewStyle().Foreground(ColorPine) }, "38;2;49;116;143"},
		{"Lavender", func() lipgloss.Style { return lipgloss.NewStyle().Foreground(ColorLavender) }, "38;2;196;167;231"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			style := tt.applyColor()
			rendered := style.Render("X")

			if len(rendered) == 0 {
				t.Error("rendered output is empty")
			}

			// The ANSI true-color escape should contain the RGB decimal sequence.
			if !strings.Contains(rendered, tt.rgbSeq) {
				t.Errorf("rendered output %q does not contain ANSI sequence %q", rendered, tt.rgbSeq)
			}
		})
	}
}

// TestStylesExist verifies that all package-level styles exist
// and produce non-empty rendered output containing the input text.
func TestStylesExist(t *testing.T) {
	tests := []struct {
		name     string
		style    lipgloss.Style
		wantBold bool
	}{
		{name: "TitleStyle", style: TitleStyle, wantBold: true},
		{name: "HeadingStyle", style: HeadingStyle, wantBold: true},
		{name: "SelectedStyle", style: SelectedStyle, wantBold: false},
		{name: "FrameStyle", style: FrameStyle, wantBold: false},
		{name: "PanelStyle", style: PanelStyle, wantBold: false},
		{name: "HelpStyle", style: HelpStyle, wantBold: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rendered := tt.style.Render("test")
			if len(rendered) == 0 {
				t.Error("rendered output is empty")
			}

			if !strings.Contains(rendered, "test") {
				t.Errorf("rendered output %q does not contain 'test'", rendered)
			}

			if tt.wantBold {
				// Lipgloss combines bold (1) with color in a single SGR:
				// \x1b[1;38;2;R;G;Bm — check for the "1;" prefix.
				if !strings.Contains(rendered, "\x1b[1;") {
					t.Errorf("rendered output %q does not contain bold escape sequence", rendered)
				}
			}
		})
	}
}

// TestCursorIndicator verifies the cursor indicator constant.
func TestCursorIndicator(t *testing.T) {
	if CursorIndicator == "" {
		t.Error("CursorIndicator is empty")
	}
	if !strings.Contains(CursorIndicator, "\u25b8") {
		t.Errorf("CursorIndicator %q does not contain ▸ (U+25B8)", CursorIndicator)
	}
}

// TestFrame verifies that Frame() wraps content in a DoubleBorder
// and produces the expected box-drawing characters.
func TestFrame(t *testing.T) {
	tests := []struct {
		name    string
		content string
		width   int
		want    []string
	}{
		{
			name:    "simple content",
			content: "hello",
			width:   40,
			want:    []string{"╔", "╚", "╗", "╝", "║", "═", "hello"},
		},
		{
			name:    "empty content",
			content: "",
			width:   40,
			want:    []string{"╔", "╚", "╗", "╝"},
		},
		{
			name:    "narrow width",
			content: "test",
			width:   10,
			want:    []string{"╔", "╗", "╚", "╝", "test"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Frame(tt.content, tt.width)

			if len(result) == 0 {
				t.Error("Frame() returned empty string")
			}

			for _, want := range tt.want {
				if !strings.Contains(result, want) {
					t.Errorf("Frame() output %q does not contain %q", result, want)
				}
			}
		})
	}
}
