package components

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/danielxxomg/bak-cli/internal/tui/styles"
)

// Search provides fuzzy search filtering for lists of strings. It wraps a
// bubbles/textinput sub-model and performs case-insensitive substring matching.
//
// bubbles/textinput is justified because the interactive search bar needs
// cursor positioning, text editing, clipboard support, and focus/blur
// management — capabilities not achievable with lipgloss alone.
//
// The zero value is usable: Activate() lazily initializes the textinput
// if needed.
type Search struct {
	textinput textinput.Model
	active    bool
	initd     bool
}

// NewSearch creates a Search with an initialized but unfocused text input.
// The placeholder text is "/ to search, esc to close".
func NewSearch() Search {
	ti := textinput.New()
	ti.Placeholder = "/ to search, esc to close"
	return Search{textinput: ti}
}

// Activate focuses the text input and marks the search as active.
// The textinput is lazily initialized if the zero value was used.
func (s *Search) Activate() {
	s.ensureInit()
	s.active = true
	s.textinput.Focus()
}

// Deactivate clears the query, blurs the text input, and marks the search
// as inactive.
func (s *Search) Deactivate() {
	s.active = false
	s.ensureInit()
	s.textinput.Blur()
	s.textinput.SetValue("")
}

// Query returns the current search query string.
func (s Search) Query() string {
	if !s.initd {
		return ""
	}
	return s.textinput.Value()
}

// SetQuery sets the query string directly.
func (s *Search) SetQuery(q string) {
	s.ensureInit()
	s.textinput.SetValue(q)
}

// Update processes incoming messages. When the search is active, key
// presses are forwarded to the textinput sub-model. When inactive,
// messages are ignored.
func (s Search) Update(msg tea.Msg) (Search, tea.Cmd) {
	if !s.active {
		return s, nil
	}
	s.ensureInit()
	newTi, cmd := s.textinput.Update(msg)
	s.textinput = newTi
	return s, cmd
}

// ensureInit lazily initializes the textinput model if it hasn't been
// initialized yet. This makes the zero value of Search usable.
func (s *Search) ensureInit() {
	if s.initd {
		return
	}
	s.textinput = textinput.New()
	s.textinput.Placeholder = "/ to search, esc to close"
	s.initd = true
}

// Filter returns items that contain the current query as a case-insensitive
// substring. If the query is empty or the search is not initialized, all
// items are returned.
func (s Search) Filter(items []string) []string {
	query := strings.ToLower(s.Query())
	if query == "" || len(items) == 0 {
		return items
	}

	var result []string
	for _, item := range items {
		if strings.Contains(strings.ToLower(item), query) {
			result = append(result, item)
		}
	}
	return result
}

// View renders the search bar using SearchStyle when the search is active.
// Returns an empty string when inactive or uninitialized.
func (s Search) View() string {
	if !s.active || !s.initd {
		return ""
	}
	return styles.SearchStyle.Render(s.textinput.View())
}
