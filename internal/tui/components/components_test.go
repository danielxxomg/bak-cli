package components

import (
	"strings"
	"testing"

	"github.com/danielxxomg/bak-cli/internal/tui/styles"
)

// =============================================================================
// RenderMenu — table-driven tests
// =============================================================================

func TestRenderMenu(t *testing.T) {
	tests := []struct {
		name       string
		items      []string
		cursor     int
		wantLabels []string // labels that must appear in output
		wantCursor bool     // whether cursor indicator should appear
		wantEmpty  bool     // whether output should be empty
	}{
		{
			name:       "single item with cursor",
			items:      []string{"Backup"},
			cursor:     0,
			wantLabels: []string{"Backup"},
			wantCursor: true,
		},
		{
			name:       "cursor at first of three",
			items:      []string{"Backup", "Restore", "Settings"},
			cursor:     0,
			wantLabels: []string{"Backup", "Restore", "Settings"},
			wantCursor: true,
		},
		{
			name:       "cursor at middle item",
			items:      []string{"Backup", "Restore", "Settings", "Quit"},
			cursor:     2,
			wantLabels: []string{"Backup", "Settings", "Quit"},
			wantCursor: true,
		},
		{
			name:       "empty items",
			items:      []string{},
			cursor:     0,
			wantLabels: nil,
			wantEmpty:  true,
		},
		{
			name:       "nil items",
			items:      nil,
			cursor:     0,
			wantLabels: nil,
			wantEmpty:  true,
		},
		{
			name:       "negative cursor clamped",
			items:      []string{"Backup", "Restore"},
			cursor:     -1,
			wantLabels: []string{"Backup", "Restore"},
			wantCursor: true,
		},
		{
			name:       "cursor past end clamped",
			items:      []string{"Backup", "Restore"},
			cursor:     5,
			wantLabels: []string{"Backup", "Restore"},
			wantCursor: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RenderMenu(tt.items, tt.cursor)

			if tt.wantEmpty {
				if len(result) != 0 {
					t.Errorf("RenderMenu() = %q, want empty string", result)
				}
				return
			}

			if len(result) == 0 {
				t.Error("RenderMenu returned empty string")
			}

			for _, label := range tt.wantLabels {
				if !strings.Contains(result, label) {
					t.Errorf("RenderMenu output %q does not contain label %q", result, label)
				}
			}

			if tt.wantCursor && !strings.Contains(result, styles.CursorIndicator) {
				t.Errorf("RenderMenu output %q does not contain cursor indicator %q", result, styles.CursorIndicator)
			}
		})
	}
}

// =============================================================================
// RenderCheckbox — table-driven tests
// =============================================================================

func TestRenderCheckbox(t *testing.T) {
	tests := []struct {
		name    string
		label   string
		checked bool
		focused bool
		want    string // substring that must appear
		notWant string // substring that must NOT appear
	}{
		{
			name:    "checked unfocused",
			label:   "Enable backups",
			checked: true,
			focused: false,
			want:    "[x]",
			notWant: "[ ]",
		},
		{
			name:    "unchecked unfocused",
			label:   "Enable backups",
			checked: false,
			focused: false,
			want:    "[ ]",
			notWant: "[x]",
		},
		{
			name:    "checked focused",
			label:   "Config files",
			checked: true,
			focused: true,
			want:    "[x]",
		},
		{
			name:    "unchecked focused",
			label:   "Skills",
			checked: false,
			focused: true,
			want:    "[ ]",
		},
		{
			name:    "empty label checked",
			label:   "",
			checked: true,
			focused: false,
			want:    "[x]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RenderCheckbox(tt.label, tt.checked, tt.focused)

			if !strings.Contains(result, tt.want) {
				t.Errorf("RenderCheckbox(%q, checked=%v, focused=%v) = %q, want to contain %q",
					tt.label, tt.checked, tt.focused, result, tt.want)
			}

			if tt.notWant != "" && strings.Contains(result, tt.notWant) {
				t.Errorf("RenderCheckbox(%q, checked=%v, focused=%v) = %q, should NOT contain %q",
					tt.label, tt.checked, tt.focused, result, tt.notWant)
			}

			if tt.label != "" && !strings.Contains(result, tt.label) {
				t.Errorf("RenderCheckbox output %q does not contain label %q", result, tt.label)
			}
		})
	}
}

// TestRenderCheckbox_FocusedDiffers verifies focused and unfocused
// renderings produce different output (style change is applied).
func TestRenderCheckbox_FocusedDiffers(t *testing.T) {
	a := RenderCheckbox("Enable backups", true, true)
	b := RenderCheckbox("Enable backups", true, false)
	if a == b {
		t.Error("focused and unfocused outputs are identical, expected difference")
	}
}

// =============================================================================
// RenderRadio — table-driven tests
// =============================================================================

func TestRenderRadio(t *testing.T) {
	tests := []struct {
		name     string
		label    string
		selected bool
		focused  bool
		want     string // substring that must appear
		notWant  string // substring that must NOT appear
	}{
		{
			name:     "selected unfocused",
			label:    "Light theme",
			selected: true,
			focused:  false,
			want:     "(•)",
			notWant:  "( )",
		},
		{
			name:     "unselected unfocused",
			label:    "Dark theme",
			selected: false,
			focused:  false,
			want:     "( )",
			notWant:  "(•)",
		},
		{
			name:     "selected focused",
			label:    "Rose Pine",
			selected: true,
			focused:  true,
			want:     "(•)",
		},
		{
			name:     "unselected focused",
			label:    "System",
			selected: false,
			focused:  true,
			want:     "( )",
		},
		{
			name:     "empty label selected",
			label:    "",
			selected: true,
			focused:  false,
			want:     "(•)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RenderRadio(tt.label, tt.selected, tt.focused)

			if !strings.Contains(result, tt.want) {
				t.Errorf("RenderRadio(%q, selected=%v, focused=%v) = %q, want to contain %q",
					tt.label, tt.selected, tt.focused, result, tt.want)
			}

			if tt.notWant != "" && strings.Contains(result, tt.notWant) {
				t.Errorf("RenderRadio(%q, selected=%v, focused=%v) = %q, should NOT contain %q",
					tt.label, tt.selected, tt.focused, result, tt.notWant)
			}

			if tt.label != "" && !strings.Contains(result, tt.label) {
				t.Errorf("RenderRadio output %q does not contain label %q", result, tt.label)
			}
		})
	}
}

// TestRenderRadio_FocusedDiffers verifies focused and unfocused
// renderings produce different output.
func TestRenderRadio_FocusedDiffers(t *testing.T) {
	a := RenderRadio("Light theme", true, true)
	b := RenderRadio("Light theme", true, false)
	if a == b {
		t.Error("focused and unfocused outputs are identical, expected difference")
	}
}

// =============================================================================
// RenderHelp — table-driven tests
// =============================================================================

func TestRenderHelp(t *testing.T) {
	tests := []struct {
		name      string
		keys      []HelpKey
		wantEmpty bool
		wantParts []string // key substrings that must appear
	}{
		{
			name:      "single key",
			keys:      []HelpKey{{Key: "q", Desc: "quit"}},
			wantParts: []string{"q", "quit"},
		},
		{
			name: "multiple keys",
			keys: []HelpKey{
				{Key: "↑/↓", Desc: "navigate"},
				{Key: "enter", Desc: "select"},
				{Key: "q", Desc: "quit"},
			},
			wantParts: []string{"↑/↓", "navigate", "enter", "select", "q", "quit"},
		},
		{
			name:      "empty keys",
			keys:      []HelpKey{},
			wantEmpty: true,
		},
		{
			name:      "nil keys",
			keys:      nil,
			wantEmpty: true,
		},
		{
			name:      "key with unicode",
			keys:      []HelpKey{{Key: "esc", Desc: "back"}},
			wantParts: []string{"esc", "back"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RenderHelp(tt.keys)

			if tt.wantEmpty {
				if len(result) != 0 {
					t.Errorf("RenderHelp() = %q, want empty string", result)
				}
				return
			}

			if len(result) == 0 {
				t.Error("RenderHelp returned empty string")
			}

			for _, part := range tt.wantParts {
				if !strings.Contains(result, part) {
					t.Errorf("RenderHelp output %q does not contain %q", result, part)
				}
			}
		})
	}
}
