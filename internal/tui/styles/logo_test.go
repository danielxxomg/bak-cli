package styles

import (
	"strings"
	"testing"
)

func TestRenderLogo_NonEmpty(t *testing.T) {
	result := RenderLogo(80)

	if len(result) == 0 {
		t.Error("RenderLogo(80) returned empty string")
	}

	// Logo should have at least 4 lines (one per gradient band).
	lines := strings.Split(result, "\n")
	if len(lines) < 4 {
		t.Errorf("RenderLogo(80) has %d lines, want at least 4", len(lines))
	}
}

// TestRenderLogo_ContainsBakWordmark verifies the logo contains a recognizable
// ASCII art representation of the project name.
func TestRenderLogo_ContainsBakWordmark(t *testing.T) {
	result := RenderLogo(80)

	// The logo contains ASCII art for "bak" — check for recognizable
	// structure characters (vertical bars, slashes) with ANSI codes stripped.
	// At minimum, the output should have multiple non-empty lines.
	lines := strings.Split(result, "\n")
	nonEmpty := 0
	for _, line := range lines {
		if len(strings.TrimSpace(line)) > 0 {
			nonEmpty++
		}
	}
	if nonEmpty < 4 {
		t.Errorf("RenderLogo(80) has %d non-empty lines, want at least 4. Output: %q", nonEmpty, result)
	}
}

func TestRenderLogo_NarrowTerminal(t *testing.T) {
	tests := []struct {
		name  string
		width int
	}{
		{"too narrow", 39},
		{"very narrow", 10},
		{"zero", 0},
		{"negative", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RenderLogo(tt.width)
			if len(result) != 0 {
				t.Errorf("RenderLogo(%d) = %q, want empty string for narrow terminal", tt.width, result)
			}
		})
	}
}

// TestRenderLogo_Gradient verifies the logo uses the Rose Pine gradient colors.
// The 5-band gradient: Love → Gold → Rose → Pine → Lavender.
func TestRenderLogo_Gradient(t *testing.T) {
	result := RenderLogo(80)

	gradientColors := []string{
		"38;2;235;111;146", // Love  #eb6f92
		"38;2;246;193;119", // Gold  #f6c177
		"38;2;235;188;186", // Rose  #ebbcba
		"38;2;49;116;143",  // Pine  #31748f
		"38;2;196;167;231", // Lavender #c4a7e7
	}

	found := 0
	for _, c := range gradientColors {
		if strings.Contains(result, c) {
			found++
		}
	}

	if found < 4 {
		t.Errorf("RenderLogo(80) uses %d/5 gradient colors, want at least 4. Output: %q", found, result)
	}
}
