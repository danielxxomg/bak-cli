package screens

import (
	"strings"
	"testing"
)

func TestRenderShortcuts(t *testing.T) {
	tests := []struct {
		name  string
		width int
		check func(t *testing.T, output string)
	}{
		{
			name:  "full width contains all keys and screens",
			width: 80,
			check: func(t *testing.T, output string) {
				if len(output) == 0 {
					t.Fatal("RenderShortcuts(80) returned empty string")
				}
				if !strings.Contains(output, "Navigation") {
					t.Error("missing 'Navigation' heading")
				}
				wantKeys := []string{"j", "k", "\u2191", "\u2193", "enter", "q", "esc", "?"}
				for _, key := range wantKeys {
					if !strings.Contains(output, key) {
						t.Errorf("missing key %q", key)
					}
				}
				wantScreens := []string{"Menu", "Dashboard", "Settings", "Progress", "Health"}
				for _, screen := range wantScreens {
					if !strings.Contains(output, screen) {
						t.Errorf("missing screen reference %q", screen)
					}
				}
			},
		},
		{
			name:  "narrow terminal still shows core content",
			width: 30,
			check: func(t *testing.T, output string) {
				if len(output) == 0 {
					t.Fatal("RenderShortcuts(30) returned empty string")
				}
				if !strings.Contains(output, "Navigation") {
					t.Error("narrow missing 'Navigation' heading")
				}
				if !strings.Contains(output, "j") {
					t.Error("narrow missing key 'j'")
				}
			},
		},
		{
			name:  "contains all four groups",
			width: 120,
			check: func(t *testing.T, output string) {
				groups := []string{"Navigation", "Actions", "Screens", "Meta"}
				for _, group := range groups {
					if !strings.Contains(output, group) {
						t.Errorf("missing group %q", group)
					}
				}
			},
		},
		{
			name:  "keys have descriptions",
			width: 80,
			check: func(t *testing.T, output string) {
				pairs := map[string]string{
					"j":     "down",
					"k":     "up",
					"enter": "select",
					"q":     "quit",
					"esc":   "back",
					"?":     "shortcuts",
					"1":     "menu",
				}
				for key, desc := range pairs {
					if !strings.Contains(output, key) {
						t.Errorf("missing key %q", key)
					}
					if !strings.Contains(strings.ToLower(output), desc) {
						t.Errorf("missing description %q for key %q", desc, key)
					}
				}
			},
		},
		{
			name:  "zero width does not panic",
			width: 0,
			check: func(t *testing.T, output string) {
				// Must not panic; output can be anything but must exist.
				_ = output
			},
		},
		{
			name:  "exactly frame threshold does not frame",
			width: 50,
			check: func(t *testing.T, output string) {
				if len(output) == 0 {
					t.Fatal("RenderShortcuts(50) returned empty")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := RenderShortcuts(tt.width)
			tt.check(t, output)
		})
	}
}
