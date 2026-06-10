package components

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// =============================================================================
// TestNewSearch — RED (search.go does not exist yet)
// =============================================================================

func TestNewSearch(t *testing.T) {
	s := NewSearch()

	if s.active {
		t.Error("NewSearch().active = true, want false")
	}
	if s.Query() != "" {
		t.Errorf("NewSearch().Query() = %q, want empty", s.Query())
	}
}

// =============================================================================
// TestSearch_ActivateDeactivate — RED
// =============================================================================

func TestSearch_ActivateDeactivate(t *testing.T) {
	s := NewSearch()

	s.Activate()
	if !s.active {
		t.Error("Activate(): active = false, want true")
	}

	s.Deactivate()
	if s.active {
		t.Error("Deactivate(): active = true, want false")
	}
	if s.Query() != "" {
		t.Errorf("Deactivate(): query = %q, want empty", s.Query())
	}
}

// =============================================================================
// TestSearch_Filter — RED
// =============================================================================

func TestSearch_Filter(t *testing.T) {
	tests := []struct {
		name  string
		items []string
		query string
		want  []string
	}{
		{
			name:  "exact match",
			items: []string{"config", "backup", "restore"},
			query: "backup",
			want:  []string{"backup"},
		},
		{
			name:  "case insensitive",
			items: []string{"Config", "Backup", "Restore"},
			query: "config",
			want:  []string{"Config"},
		},
		{
			name:  "partial match",
			items: []string{"config.yaml", "backup.sh", "restore.sh"},
			query: ".sh",
			want:  []string{"backup.sh", "restore.sh"},
		},
		{
			name:  "no match",
			items: []string{"config", "backup", "restore"},
			query: "xyz",
			want:  nil,
		},
		{
			name:  "empty query returns all",
			items: []string{"a", "b", "c"},
			query: "",
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "empty items",
			items: nil,
			query: "test",
			want:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewSearch()
			s.SetQuery(tt.query)
			result := s.Filter(tt.items)

			if len(result) != len(tt.want) {
				t.Errorf("Filter() len = %d, want %d (got %v)", len(result), len(tt.want), result)
				return
			}
			for i, v := range result {
				if v != tt.want[i] {
					t.Errorf("Filter()[%d] = %q, want %q", i, v, tt.want[i])
				}
			}
		})
	}
}

// =============================================================================
// TestSearch_View — RED
// =============================================================================

func TestSearch_View(t *testing.T) {
	s := NewSearch()

	// Hidden when not active.
	hidden := s.View()
	if hidden != "" {
		t.Errorf("View() inactive = %q, want empty", hidden)
	}

	// Visible when active.
	s.Activate()
	visible := s.View()
	if len(visible) == 0 {
		t.Fatal("View() active returned empty string")
	}
	// Should show search bar with placeholder.
	if !strings.Contains(visible, "/") && !strings.Contains(visible, "search") {
		t.Errorf("View() active missing search indicator: %q", visible)
	}
}

// =============================================================================
// TestSearch_Update — RED
// =============================================================================

func TestSearch_Update(t *testing.T) {
	s := NewSearch()
	s.Activate()

	// Type "conf" through Update.
	for _, ch := range []string{"c", "o", "n", "f"} {
		msg := tea.KeyPressMsg{Code: []rune(ch)[0], Text: ch}
		newS, _ := s.Update(msg)
		s = newS
	}

	q := s.Query()
	if q != "conf" {
		t.Errorf("after Update key sequence: query = %q, want %q", q, "conf")
	}
}

// =============================================================================
// TestSearch_Update_Inactive — RED (triangulation)
// =============================================================================

func TestSearch_Update_Inactive(t *testing.T) {
	s := NewSearch()

	// When not active, Update should still work but not register input.
	newS, cmd := s.Update(tea.KeyPressMsg{Code: 'x'})
	if cmd != nil {
		t.Error("inactive Update returned non-nil cmd")
	}
	if newS.Query() != "" {
		t.Errorf("inactive Update: query = %q, want empty", newS.Query())
	}
}
