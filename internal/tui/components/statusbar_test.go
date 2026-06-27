// Package components provides reusable, pure render functions for TUI
// components. This file contains TDD tests for the persistent status bar
// (tui-personality REQ-TP-003), written BEFORE the production code.
package components

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

func TestRenderStatusBar(t *testing.T) { //nolint:paralleltest // shared styles/colorprofile global state
	tests := []struct {
		name       string
		width      int
		version    string
		preset     string
		path       string
		wantEmpty  bool
		contains   []string
		notContain []string
		wantMaxW   int // visible width must be <= wantMaxW (0 = skip width check)
	}{
		{
			name:     "wide terminal shows all segments",
			width:    80,
			version:  "1.0.0",
			preset:   "default",
			path:     "/home/user/.bak/backups",
			contains: []string{"bak v1.0.0", "default", "/home/user/.bak/backups"},
			wantMaxW: 80,
		},
		{
			name:      "narrow below 40 hidden",
			width:     39,
			version:   "1.0.0",
			preset:    "default",
			path:      "/x",
			wantEmpty: true,
		},
		{
			name:     "exactly 40 columns shown",
			width:    40,
			version:  "1.0.0",
			contains: []string{"bak v1.0.0"},
			wantMaxW: 40,
		},
		{
			name:       "long path truncated with ellipsis",
			width:      60,
			version:    "1.0.0",
			preset:     "default",
			path:       strings.Repeat("a", 50),
			contains:   []string{"bak v1.0.0", "default", "…"},
			notContain: []string{strings.Repeat("a", 50)}, // full path must not fit
			wantMaxW:   60,
		},
		{
			name:     "empty version still shows bak",
			width:    80,
			version:  "",
			preset:   "",
			path:     "",
			contains: []string{"bak"},
		},
		{
			name:       "no preset no path omits separator",
			width:      80,
			version:    "1.0.0",
			preset:     "",
			path:       "",
			contains:   []string{"bak v1.0.0"},
			notContain: []string{"•"},
		},
		{
			name:     "path only segment",
			width:    80,
			version:  "",
			preset:   "",
			path:     "/var/bak",
			contains: []string{"/var/bak"},
		},
	}

	for _, tt := range tests { //nolint:paralleltest // subtests share table/struct state
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
			got := RenderStatusBar(tt.width, tt.version, tt.preset, tt.path)

			if tt.wantEmpty {
				if got != "" {
					t.Errorf("RenderStatusBar(%d,…) = %q, want empty (hidden <40)", tt.width, got)
				}
				return
			}
			if got == "" {
				t.Fatalf("RenderStatusBar(%d,…) returned empty, want non-empty", tt.width)
			}
			for _, want := range tt.contains {
				if !strings.Contains(got, want) {
					t.Errorf("output %q must contain %q", got, want)
				}
			}
			for _, notWant := range tt.notContain {
				if strings.Contains(got, notWant) {
					t.Errorf("output %q must NOT contain %q", got, notWant)
				}
			}
			if tt.wantMaxW > 0 {
				if w := lipgloss.Width(got); w > tt.wantMaxW {
					t.Errorf("visible width = %d, want <= %d (output %q)", w, tt.wantMaxW, got)
				}
			}
		})
	}
}
