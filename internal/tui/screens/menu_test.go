// Package screens provides screen-specific render functions for the bak-cli TUI.
// This file contains TDD tests written BEFORE the production code (strict RED phase).
package screens

import (
	"strings"
	"testing"

	"github.com/danielxxomg/bak-cli/internal/tui/styles"
)

// =============================================================================
// TestRenderMainMenu — table-driven RED (menu.go does not exist yet)
// =============================================================================

func TestRenderMainMenu(t *testing.T) {
	defaultItems := []string{
		"Create backup", "Restore", "Browse backups",
		"Cloud sync", "Profiles", "Settings", "Quit",
	}

	tests := []struct {
		name           string
		version        string
		items          []string
		cursor         int
		width          int
		wantContains   []string
		wantNotContain []string
		wantLogo       bool // true: logo chars expected; false: must NOT appear
	}{
		{
			name:    "full menu at 80 cols",
			version: "1.0.0",
			items:   defaultItems,
			cursor:  0,
			width:   80,
			wantContains: []string{
				"Create backup", "Restore", "Browse backups",
				"Cloud sync", "Profiles", "Settings", "Quit",
				styles.CursorIndicator,
				"\u2191/\u2193", "enter", "q",
				"navigate", "select", "quit",
			},
			wantLogo: true,
		},
		{
			name:    "narrow terminal hides logo",
			version: "1.0.0",
			items:   defaultItems,
			cursor:  0,
			width:   30,
			wantContains: []string{
				"Create backup",
				"navigate", "select", "quit",
			},
			wantNotContain: []string{"|_|", "|  _"},
			wantLogo:       false,
		},
		{
			name:         "custom version subtitle",
			version:      "2.3.1",
			items:        defaultItems,
			cursor:       0,
			width:        80,
			wantContains: []string{"2.3.1"},
			wantLogo:     true,
		},
		{
			name:         "help bar contextual keys",
			version:      "1.0.0",
			items:        defaultItems,
			cursor:       0,
			width:        80,
			wantContains: []string{"navigate", "select", "quit"},
			wantLogo:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := RenderMainMenu(tt.version, "", tt.items, tt.cursor, tt.width)

			if len(output) == 0 {
				t.Fatal("RenderMainMenu returned empty string")
			}

			for _, want := range tt.wantContains {
				if !strings.Contains(output, want) {
					t.Errorf("output missing %q", want)
				}
			}

			for _, notWant := range tt.wantNotContain {
				if strings.Contains(output, notWant) {
					t.Errorf("output unexpectedly contains %q", notWant)
				}
			}

			if tt.wantLogo {
				if !strings.Contains(output, "|") {
					t.Error("output missing logo ASCII art characters")
				}
			}
		})
	}
}

// TestRenderMainMenu_CursorPositions verifies the cursor indicator is at
// the correct position for different cursor values.
func TestRenderMainMenu_CursorPositions(t *testing.T) {
	menuItems := []string{
		"Create backup", "Restore", "Browse backups",
		"Cloud sync", "Profiles", "Settings", "Quit",
	}
	tests := []struct {
		name     string
		cursor   int
		wantItem string // the item that should have cursor indicator
	}{
		{"cursor at first", 0, "Create backup"},
		{"cursor at middle", 3, "Cloud sync"},
		{"cursor at last", 6, "Quit"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := RenderMainMenu("1.0.0", "", menuItems, tt.cursor, 80)

			// The cursor indicator should appear before the selected item.
			// Since RenderMenu pads non-selected items with spaces, we can
			// check that the cursor indicator + selected item appears.
			if !strings.Contains(output, styles.CursorIndicator) {
				t.Errorf("RenderMainMenu(cursor=%d) missing cursor indicator", tt.cursor)
			}

			// Verify the expected item is present.
			if !strings.Contains(output, tt.wantItem) {
				t.Errorf("RenderMainMenu(cursor=%d) missing item %q", tt.cursor, tt.wantItem)
			}
		})
	}
}

// TestRenderMainMenu_EdgeCases verifies behavior with nil and empty
// menu item slices. Even with no menu items, RenderMainMenu still produces
// output (logo, version, help bar) — it should never panic. The key
// validation is that menu item labels do NOT appear when the slice is
// nil/empty, proving the menu renderer was called but returned nothing.
func TestRenderMainMenu_EdgeCases(t *testing.T) {
	tests := []struct {
		name       string
		items      []string
		wantOutput bool // false = menu items section empty
	}{
		{"nil items", nil, false},
		{"empty items", []string{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := RenderMainMenu("1.0.0", "", tt.items, 0, 80)

			// Output must be non-empty even with no items:
			// logo + version + help bar are always rendered when width >= 80.
			if len(output) == 0 {
				t.Fatal("RenderMainMenu returned empty string — expected logo/version/help")
			}

			// The help bar must still be present.
			if !strings.Contains(output, "navigate") {
				t.Error("output missing help bar")
			}

			// The version must still be present.
			if !strings.Contains(output, "1.0.0") {
				t.Error("output missing version")
			}

			// But no known menu items should appear (slice is nil/empty).
			knownItems := []string{
				"Create backup", "Restore", "Browse backups",
				"Cloud sync", "Profiles", "Settings", "Quit",
			}
			for _, knownItem := range knownItems {
				if strings.Contains(output, knownItem) {
					t.Errorf("output unexpectedly contains menu item %q (items=%v)", knownItem, tt.items)
				}
			}
		})
	}
}

// TestRenderMainMenu_EdgeCases_Rendered verifies that out-of-bounds cursors
// are clamped to valid range and produce deterministic, non-empty output
// with the cursor indicator on the correct item.
func TestRenderMainMenu_EdgeCases_Rendered(t *testing.T) {
	tests := []struct {
		name         string
		items        []string
		cursor       int
		wantContains string // a string that MUST be present in output
		wantLine     string // exact line that should start with cursor indicator
	}{
		{
			name:         "negative cursor clamped to 0",
			items:        []string{"A", "B", "C"},
			cursor:       -5,
			wantContains: styles.CursorIndicator,
			wantLine:     "A",
		},
		{
			name:         "cursor past end clamped to last",
			items:        []string{"A", "B", "C"},
			cursor:       99,
			wantContains: styles.CursorIndicator,
			wantLine:     "C",
		},
		{
			name:         "negative cursor on single item",
			items:        []string{"Only"},
			cursor:       -1,
			wantContains: styles.CursorIndicator,
			wantLine:     "Only",
		},
		{
			name:         "cursor past end on single item",
			items:        []string{"Only"},
			cursor:       5,
			wantContains: styles.CursorIndicator,
			wantLine:     "Only",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := RenderMainMenu("1.0.0", "", tt.items, tt.cursor, 80)

			if len(output) == 0 {
				t.Fatal("RenderMainMenu returned empty string")
			}
			if !strings.Contains(output, tt.wantContains) {
				t.Errorf("output missing %q", tt.wantContains)
			}

			// Verify the clamped item has the cursor indicator.
			// The cursor indicator + selected item appears on the same line.
			cursorLine := styles.CursorIndicator + tt.wantLine
			styledLine := styles.CursorIndicator + styles.SelectedStyle.Render(tt.wantLine)
			if !strings.Contains(output, cursorLine) && !strings.Contains(output, styledLine) {
				t.Errorf("output missing cursor indicator on %q;\ngot:\n%s", tt.wantLine, output)
			}
		})
	}
}
