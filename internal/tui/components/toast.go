package components

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/danielxxomg/bak-cli/internal/tui/styles"
)

// toastTickMsg is sent internally by tea.Tick every second to advance
// the toast countdown. It is unexported because external packages should
// not send ticks directly; they should use the Show method instead.
type toastTickMsg struct{}

// Toast is a non-intrusive notification component that displays a message
// for a set duration (TTL in seconds) at the bottom-right of the screen.
// After the TTL expires, the toast auto-hides.
//
// The zero value is a valid hidden toast. Use NewToast() for a clean
// starting state, then call Show(message, ttl) to display a message.
type Toast struct {
	message string
	visible bool
	ttl     int // duration in seconds
	ticks   int // elapsed seconds
}

// NewToast creates a new Toast in a hidden state with no message.
func NewToast() Toast {
	return Toast{}
}

// Show activates the toast with the given message and auto-hide duration.
// The toast becomes visible immediately and counts down from ttl seconds.
// If ttl is 0 or negative, the toast remains hidden.
func (t *Toast) Show(message string, ttl int) {
	t.message = message
	t.visible = ttl > 0
	t.ttl = ttl
	t.ticks = 0
}

// Update processes incoming messages. When the toast is visible and
// receives any message, it starts a tea.Tick that fires every second.
// Each toastTickMsg increments an internal counter; when the counter
// reaches ttl, the toast hides and stops ticking.
func (t Toast) Update(msg tea.Msg) (Toast, tea.Cmd) {
	if !t.visible {
		return t, nil
	}
	switch msg.(type) {
	case toastTickMsg:
		t.ticks++
		if t.ticks >= t.ttl {
			t.visible = false
			return t, nil
		}
	}
	// While visible, schedule the next tick.
	return t, tea.Tick(time.Second, func(_ time.Time) tea.Msg {
		return toastTickMsg{}
	})
}

// View renders the toast message as a styled overlay. When hidden, it
// returns an empty string.
func (t Toast) View() string {
	if !t.visible || t.message == "" {
		return ""
	}
	return styles.ToastStyle.Render(t.message)
}
