// Package components provides reusable, pure render functions for TUI
// components. empty_state_test.go covers the styled empty-state renderer
// (tui-personality REQ-TP-007) written BEFORE the production code.
package components

import (
	"strings"
	"testing"
)

// TestRenderEmptyState verifies the styled empty state renders the icon, the
// message, and the hint (REQ-TP-007). Table-driven so each call site's
// icon/message/hint triple is exercised.
func TestRenderEmptyState(t *testing.T) { //nolint:paralleltest // matches established codebase convention across all tui tests
	tests := []struct {
		name    string
		icon    string
		message string
		hint    string
	}{
		{
			name:    "no backups",
			icon:    "∅",
			message: "No backups yet",
			hint:    "Run 'bak backup' to create one",
		},
		{
			name:    "no cloud provider",
			icon:    "☁",
			message: "No cloud provider configured",
			hint:    "Run 'bak cloud login' to connect",
		},
		{
			name:    "no restore targets",
			icon:    "∅",
			message: "No backups to restore",
			hint:    "Run 'bak backup' to create one first",
		},
	}

	for _, tt := range tests { //nolint:paralleltest // subtests share table/struct state
		t.Run(tt.name, func(t *testing.T) {
			out := RenderEmptyState(tt.icon, tt.message, tt.hint)

			if out == "" {
				t.Fatal("RenderEmptyState returned empty string")
			}
			if !strings.Contains(out, tt.icon) {
				t.Errorf("empty state missing icon %q: %q", tt.icon, out)
			}
			if !strings.Contains(out, tt.message) {
				t.Errorf("empty state missing message %q: %q", tt.message, out)
			}
			if !strings.Contains(out, tt.hint) {
				t.Errorf("empty state missing hint %q: %q", tt.hint, out)
			}
		})
	}
}

// TestRenderEmptyState_NonEmptySegments triangulates that the renderer
// composes all three segments together (not just one) and is stable across
// repeated calls.
func TestRenderEmptyState_NonEmptySegments(t *testing.T) { //nolint:paralleltest // matches established codebase convention across all tui tests
	out := RenderEmptyState("∅", "Nothing here", "do something")

	// All three pieces appear together in a single render.
	if !strings.Contains(out, "∅") || !strings.Contains(out, "Nothing here") || !strings.Contains(out, "do something") {
		t.Errorf("empty state did not compose all segments: %q", out)
	}

	// Idempotent: a second call yields the same output.
	if out2 := RenderEmptyState("∅", "Nothing here", "do something"); out2 != out {
		t.Errorf("RenderEmptyState not idempotent: %q vs %q", out, out2)
	}
}
