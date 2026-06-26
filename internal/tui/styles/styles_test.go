package styles

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

// TestColors verifies all 11 Rose Pine semantic colors exist as package-level
// variables and produce valid ANSI true-color sequences when rendered.
// Lipgloss converts hex colors to RGB decimal ANSI sequences: e.g. #191724 → 38;2;25;23;36.
func TestColors(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
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

	for _, tt := range tests { //nolint:paralleltest // subtests share table/struct state
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
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
func TestStylesExist(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
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

	for _, tt := range tests { //nolint:paralleltest // subtests share table/struct state
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
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

// =============================================================================
// TestToastStyle_Bordered — RED (ToastStyle does not yet have Border/Background)
// =============================================================================

// TestToastStyle_HasBorder verifies that ToastStyle includes a visible border.
func TestToastStyle_HasBorder(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	rendered := ToastStyle.Render("test message")

	if len(rendered) == 0 {
		t.Fatal("ToastStyle rendered empty output")
	}

	// A bordered style should contain at least one box-drawing character.
	hasBorder := false
	for _, r := range "─│┌┐└┘" {
		if strings.ContainsRune(rendered, r) {
			hasBorder = true
			break
		}
	}
	if !hasBorder {
		t.Errorf("ToastStyle output %q does not contain border box-drawing characters", rendered)
	}
}

// TestToastStyle_HasBackground verifies that ToastStyle includes a
// background color (ANSI 48;2;R;G;Bm sequence).
func TestToastStyle_HasBackground(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	rendered := ToastStyle.Render("test message")

	if !strings.Contains(rendered, "48;") {
		t.Errorf("ToastStyle output %q does not contain background ANSI sequence '48;'", rendered)
	}
}

// TestCursorIndicator verifies the cursor indicator constant.
func TestCursorIndicator(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	if CursorIndicator == "" {
		t.Error("CursorIndicator is empty")
	}
	if !strings.Contains(CursorIndicator, "\u25b8") {
		t.Errorf("CursorIndicator %q does not contain ▸ (U+25B8)", CursorIndicator)
	}
}

// =============================================================================
// TestIsTooSmall — RED (IsTooSmall does not exist yet)
// =============================================================================

func TestIsTooSmall(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	tests := []struct {
		name     string
		width    int
		height   int
		tooSmall bool
	}{
		// Exact minimum: 30x15 should NOT be too small.
		{"exact minimum (30x15)", 30, 15, false},
		// Below minimum in one dimension.
		{"below width (29x20)", 29, 20, true},
		{"below height (50x14)", 50, 14, true},
		// Well below in both.
		{"well below (20x10)", 20, 10, true},
		{"very small (10x5)", 10, 5, true},
		// Well above minimum.
		{"well above (80x24)", 80, 24, false},
		{"large (120x40)", 120, 40, false},
		// Edge: exactly on width, below height.
		{"width exact, height below (30x14)", 30, 14, true},
		// Edge: exactly on height, below width.
		{"height exact, width below (29x15)", 29, 15, true},
		// Zero values.
		{"zero dimensions (0x0)", 0, 0, true},
		// Just above minimum.
		{"just above (31x16)", 31, 16, false},
	}

	for _, tt := range tests { //nolint:paralleltest // subtests share table/struct state
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
			got := IsTooSmall(tt.width, tt.height)
			if got != tt.tooSmall {
				t.Errorf("IsTooSmall(%d, %d) = %v, want %v",
					tt.width, tt.height, got, tt.tooSmall)
			}
		})
	}
}

// TestRenderTooSmall verifies styles.RenderTooSmall produces the "Terminal too
// small" warning showing the current dimensions and the required minimum.
// Covers spec REQ-TD-003 §"RenderTooSmall produces correct message" (task 4.1,
// RED): RenderTooSmall(15, 5) must contain "Terminal too small (15x5)".
func TestRenderTooSmall(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	tests := []struct {
		name    string
		width   int
		height  int
		wantSub []string // substrings that MUST appear (proves real output, not hardcoded)
	}{
		{
			name:    "15x5 contains current dims and minimum hint",
			width:   15,
			height:  5,
			wantSub: []string{"Terminal too small (15x5)", "Need at least 30x15"},
		},
		{
			name:    "10x3 uses different current dims (triangulation)",
			width:   10,
			height:  3,
			wantSub: []string{"Terminal too small (10x3)", "Need at least 30x15"},
		},
		{
			name:    "80x24 still reports current dimensions",
			width:   80,
			height:  24,
			wantSub: []string{"Terminal too small (80x24)", "Need at least 30x15"},
		},
	}

	for _, tt := range tests { //nolint:paralleltest // subtests share table/struct state
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
			got := RenderTooSmall(tt.width, tt.height)
			if got == "" {
				t.Fatal("RenderTooSmall returned empty string")
			}
			for _, sub := range tt.wantSub {
				if !strings.Contains(got, sub) {
					t.Errorf("RenderTooSmall(%d, %d) = %q, want substring %q", tt.width, tt.height, got, sub)
				}
			}
		})
	}
}

// TestFrame verifies that Frame() wraps content in a DoubleBorder
// and produces the expected box-drawing characters.
func TestFrame(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
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

	for _, tt := range tests { //nolint:paralleltest // subtests share table/struct state
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
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
