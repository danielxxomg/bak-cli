package screens

import (
	"strings"
	"testing"
)

// =============================================================================
// TestRenderWelcome — RED (welcome.go does not exist yet)
// =============================================================================

func TestRenderWelcome(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	output := RenderWelcome(80)

	if len(output) == 0 {
		t.Fatal("RenderWelcome() returned empty string")
	}

	// Must contain a recognizable welcome message.
	if !strings.Contains(output, "Welcome") && !strings.Contains(output, "welcome") {
		t.Errorf("RenderWelcome output does not contain welcome message: %q", output)
	}

	// Must contain a setup prompt or continue instruction.
	hasSetup := strings.Contains(output, "setup") ||
		strings.Contains(output, "Set up") ||
		strings.Contains(output, "configure") ||
		strings.Contains(output, "Configure") ||
		strings.Contains(output, "continue") ||
		strings.Contains(output, "Continue") ||
		strings.Contains(output, "enter")
	if !hasSetup {
		t.Errorf("RenderWelcome output does not contain setup prompt: %q", output)
	}
}

// TestRenderWelcome_NarrowTerminal verifies the welcome screen adapts to
// narrow terminals: no Frame border, but content still present.
func TestRenderWelcome_NarrowTerminal(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	output := RenderWelcome(30)

	if len(output) == 0 {
		t.Fatal("RenderWelcome(30) returned empty string")
	}

	// Should still have welcome message even on narrow terminal.
	if !strings.Contains(output, "Welcome") && !strings.Contains(output, "welcome") {
		t.Errorf("RenderWelcome(30) output missing welcome: %q", output)
	}
}

// TestRenderWelcome_WideTerminalHasFrame verifies the Frame border
// appears on terminals wide enough (>= 50).
func TestRenderWelcome_WideTerminalHasFrame(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	output := RenderWelcome(80)

	if !strings.Contains(output, "\u2554") { // ╔ (top-left double border)
		t.Errorf("RenderWelcome(80) missing frame border, output: %q", output)
	}
	if !strings.Contains(output, "Welcome") {
		t.Error("RenderWelcome(80) missing welcome message inside frame")
	}
}

// =============================================================================
// TestShouldShowWelcome — RED
// =============================================================================

func TestShouldShowWelcome(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	tests := []struct {
		name         string
		configExists bool
		want         bool
	}{
		{
			name:         "no config — show welcome",
			configExists: false,
			want:         true,
		},
		{
			name:         "config exists — skip welcome",
			configExists: true,
			want:         false,
		},
	}

	for _, tt := range tests { //nolint:paralleltest // subtests share table/struct state
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
			got := ShouldShowWelcome(func() bool { return tt.configExists })
			if got != tt.want {
				t.Errorf("ShouldShowWelcome(configExists=%v) = %v, want %v",
					tt.configExists, got, tt.want)
			}
		})
	}
}

// TestShouldShowWelcome_NilFunc verifies ShouldShowWelcome handles a nil
// function gracefully (returns false — safe default).
func TestShouldShowWelcome_NilFunc(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	got := ShouldShowWelcome(nil)
	if got {
		t.Error("ShouldShowWelcome(nil) = true, want false (safe default)")
	}
}

// TestWelcomeView_HasLogoAndTagline verifies the welcome screen includes
// the ASCII logo and tagline.
func TestWelcomeView_HasLogoAndTagline(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	output := RenderWelcome(80)

	// Tagline must be present.
	if !strings.Contains(output, "Pack your AI coding setup. Move anywhere.") {
		t.Error("welcome screen missing tagline")
	}

	// Logo must be present (part of ASCII art).
	if !strings.Contains(output, "|  _ \\") {
		t.Error("welcome screen missing ASCII logo")
	}

	// "Press Enter to get started" prompt.
	if !strings.Contains(output, "Enter") {
		t.Error("welcome screen missing Enter prompt")
	}
}
