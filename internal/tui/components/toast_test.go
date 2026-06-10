// Package components provides reusable TUI components for bak-cli.
// This file contains TDD tests written BEFORE the production code
// (strict RED phase) for the toast notification component.
package components

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

// =============================================================================
// TestNewToast — RED (toast.go does not exist yet)
// =============================================================================

func TestNewToast(t *testing.T) {
	tst := NewToast()

	if tst.message != "" {
		t.Errorf("NewToast().message = %q, want empty string", tst.message)
	}
	if tst.visible {
		t.Error("NewToast().visible = true, want false")
	}
	if tst.ttl != 0 {
		t.Errorf("NewToast().ttl = %d, want 0", tst.ttl)
	}
	if tst.ticks != 0 {
		t.Errorf("NewToast().ticks = %d, want 0", tst.ticks)
	}
}

// =============================================================================
// TestToast_Show — RED
// =============================================================================

func TestToast_Show(t *testing.T) {
	tst := NewToast()
	tst.Show("Backup complete", 3)

	if tst.message != "Backup complete" {
		t.Errorf("Show() message = %q, want %q", tst.message, "Backup complete")
	}
	if !tst.visible {
		t.Error("Show() visible = false, want true")
	}
	if tst.ttl != 3 {
		t.Errorf("Show() ttl = %d, want 3", tst.ttl)
	}
	// ticks should reset to 0 on Show.
	if tst.ticks != 0 {
		t.Errorf("Show() ticks = %d, want 0", tst.ticks)
	}
}

// =============================================================================
// TestToast_Update_Tick — RED
// =============================================================================

func TestToast_Update_Tick(t *testing.T) {
	tests := []struct {
		name        string
		ttl         int
		ticksToSend int
		wantVisible bool
	}{
		{
			name:        "hides after TTL expires",
			ttl:         2,
			ticksToSend: 2,
			wantVisible: false,
		},
		{
			name:        "still visible before TTL",
			ttl:         3,
			ticksToSend: 2,
			wantVisible: true,
		},
		{
			name:        "ttl of 1 hides after 1 tick",
			ttl:         1,
			ticksToSend: 1,
			wantVisible: false,
		},
		{
			name:        "ttl 0 stays hidden",
			ttl:         0,
			ticksToSend: 1,
			wantVisible: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tst := NewToast()
			tst.Show("test", tt.ttl)

			for i := 0; i < tt.ticksToSend; i++ {
				tst, _ = tst.Update(toastTickMsg{})
			}

			if tst.visible != tt.wantVisible {
				t.Errorf("after %d ticks (ttl=%d): visible = %v, want %v",
					tt.ticksToSend, tt.ttl, tst.visible, tt.wantVisible)
			}
		})
	}
}

// =============================================================================
// TestToast_Update_ReturnsTick — RED
// =============================================================================

func TestToast_Update_ReturnsTick(t *testing.T) {
	tst := NewToast()
	tst.Show("test", 3)

	// First Update when visible should return a tick command.
	_, cmd := tst.Update(tea.KeyPressMsg{Code: 'a'}) // irrelevant msg
	if cmd == nil {
		t.Error("Update() visible toast returned nil cmd, want tick command")
	}

	// After ticking to expiry, should return nil.
	for i := 0; i < 3; i++ {
		tst, _ = tst.Update(toastTickMsg{})
	}
	_, cmd = tst.Update(toastTickMsg{})
	if cmd != nil {
		t.Error("Update() hidden toast returned non-nil cmd, want nil")
	}
}

// =============================================================================
// TestToast_Update_TickReturnsTick — RED
// =============================================================================

func TestToast_Update_TickReturnsTick(t *testing.T) {
	tst := NewToast()
	tst.Show("test", 3)

	// Tick should advance and return another tick for the next second.
	newTst, cmd := tst.Update(toastTickMsg{})
	if cmd == nil {
		t.Error("tick msg returned nil cmd, want tick command")
	}
	if newTst.ticks != 1 {
		t.Errorf("after 1 tick: ticks = %d, want 1", newTst.ticks)
	}
}

// =============================================================================
// TestToast_View_Visible — RED
// =============================================================================

func TestToast_View_Visible(t *testing.T) {
	tst := NewToast()
	tst.Show("Backup complete!", 3)

	output := tst.View()

	if len(output) == 0 {
		t.Fatal("View() visible returned empty string, want styled message")
	}
	if !strings.Contains(output, "Backup complete!") {
		t.Errorf("View() visible missing message %q: %q", "Backup complete!", output)
	}
}

// =============================================================================
// TestToast_View_Hidden — RED
// =============================================================================

func TestToast_View_Hidden(t *testing.T) {
	tst := NewToast()

	output := tst.View()

	if output != "" {
		t.Errorf("View() hidden = %q, want empty string", output)
	}
}

// =============================================================================
// TestToast_TickIsPeriodic — RED (triangulation)
// =============================================================================

func TestToast_TickIsPeriodic(t *testing.T) {
	tst := NewToast()
	tst.Show("test", 5)

	// Simulate 5 seconds via tick messages.
	for i := 0; i < 5; i++ {
		tst, _ = tst.Update(toastTickMsg{})
	}

	if tst.visible {
		t.Error("after ttl=5 and 5 ticks: visible = true, want false")
	}
	if tst.ticks != 5 {
		t.Errorf("after 5 ticks: ticks = %d, want 5", tst.ticks)
	}
}

// =============================================================================
// TestToast_ShowResets — RED (triangulation)
// =============================================================================

func TestToast_ShowResets(t *testing.T) {
	tst := NewToast()
	tst.Show("first", 2)
	// Tick once.
	tst, _ = tst.Update(toastTickMsg{})

	// Show again should reset counter.
	tst.Show("second", 3)

	if tst.ticks != 0 {
		t.Errorf("after Show() reset: ticks = %d, want 0", tst.ticks)
	}
	if tst.message != "second" {
		t.Errorf("after Show() reset: message = %q, want %q", tst.message, "second")
	}
	if tst.ttl != 3 {
		t.Errorf("after Show() reset: ttl = %d, want 3", tst.ttl)
	}
}

// =============================================================================
// TestToast_NoDoubleTickWhenHidden — RED (triangulation)
// =============================================================================

func TestToast_NoDoubleTickWhenHidden(t *testing.T) {
	tst := NewToast()

	// Hidden toast should not return tick cmd.
	_, cmd := tst.Update(toastTickMsg{})
	if cmd != nil {
		t.Error("hidden toast Update(tick) returned cmd, want nil")
	}
	if tst.visible {
		t.Error("hidden toast became visible after tick")
	}
}

// Ensure tea and time imports are used in production code.
var _ = tea.Quit
var _ = time.Second
