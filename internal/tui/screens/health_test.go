package screens

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// =============================================================================
// TestNewHealthModel — RED (health.go does not exist yet)
// =============================================================================

func TestNewHealthModel(t *testing.T) {
	m := NewHealthModel()

	if len(m.checks) != 0 {
		t.Errorf("NewHealthModel().checks len = %d, want 0", len(m.checks))
	}
	if m.running {
		t.Error("NewHealthModel().running = true, want false")
	}
}

// =============================================================================
// TestHealth_Update_Run — RED
// =============================================================================

func TestHealth_Update_Run(t *testing.T) {
	m := NewHealthModel()

	// Press enter to start the health check.
	newM, cmd := m.Update(tea.KeyPressMsg{Code: '\r'})
	model := newM.(HealthModel)

	if cmd == nil {
		t.Fatal("Update(enter) returned nil cmd, want run command")
	}
	if !model.running {
		t.Error("Update(enter): running = false, want true")
	}
	if len(model.checks) == 0 {
		t.Error("Update(enter): checks empty, want populated")
	}
}

// =============================================================================
// TestHealth_Update_Back — RED
// =============================================================================

func TestHealth_Update_Back(t *testing.T) {
	tests := []struct {
		name string
		key  rune
	}{
		{"q goes back", 'q'},
		{"esc goes back", 27},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewHealthModel()

			_, cmd := m.Update(tea.KeyPressMsg{Code: tt.key})

			if cmd == nil {
				t.Fatalf("%s returned nil cmd, want ScreenBackMsg", tt.name)
			}
			msg := cmd()
			if _, ok := msg.(ScreenBackMsg); !ok {
				t.Errorf("%s returned %T, want ScreenBackMsg", tt.name, msg)
			}
		})
	}
}

// =============================================================================
// TestHealth_View_Running — RED
// =============================================================================

func TestHealth_View_Running(t *testing.T) {
	m := NewHealthModel()
	m.width = 80
	m.height = 24
	m.running = true
	m.checks = []HealthCheck{
		{Name: "Config check", Status: StepRunning, Detail: "checking..."},
		{Name: "Backup dir", Status: StepPending, Detail: ""},
	}

	output := m.View().Content

	if len(output) == 0 {
		t.Fatal("View() running returned empty content")
	}
	if !strings.Contains(output, "Config check") {
		t.Error("View() running missing check name 'Config check'")
	}
	if !strings.Contains(output, "Backup dir") {
		t.Error("View() running missing check name 'Backup dir'")
	}
	// Should show running indicator.
	if !strings.Contains(output, "\u28f9") {
		t.Errorf("View() running missing spinner indicator: %q", output)
	}
}

// =============================================================================
// TestHealth_View_Done — RED
// =============================================================================

func TestHealth_View_Done(t *testing.T) {
	m := NewHealthModel()
	m.width = 80
	m.height = 24
	m.running = false
	m.checks = []HealthCheck{
		{Name: "Config check", Status: StepDone, Detail: "OK"},
		{Name: "Backup dir", Status: StepDone, Detail: "/home/user/bak"},
		{Name: "Cloud reachable", Status: StepPending, Detail: ""},
	}

	output := m.View().Content

	if !strings.Contains(output, "\u2713") {
		t.Errorf("View() done missing check mark: %q", output)
	}
	if !strings.Contains(output, "OK") {
		t.Error("View() done missing detail 'OK'")
	}
	if !strings.Contains(output, "Cloud reachable") {
		t.Error("View() done missing check 'Cloud reachable'")
	}
}

// =============================================================================
// TestHealth_View_TooSmall — RED (triangulation)
// =============================================================================

func TestHealth_View_TooSmall(t *testing.T) {
	m := NewHealthModel()
	m.width = 10
	m.height = 5

	output := m.View().Content

	if !strings.Contains(output, "Terminal too small") {
		t.Errorf("View() too-small missing warning: %q", output)
	}
}

// =============================================================================
// TestHealth_ResultMessage — RED (triangulation)
// =============================================================================

func TestHealth_ResultMessage(t *testing.T) {
	m := NewHealthModel()
	m.running = true
	m.checks = []HealthCheck{
		{Name: "Config", Status: StepRunning, Detail: ""},
	}

	// Send a result for the first check.
	newM, _ := m.Update(healthCheckResultMsg{index: 0, detail: "OK"})
	model := newM.(HealthModel)

	if model.checks[0].Status != StepDone {
		t.Errorf("after result: status = %v, want StepDone", model.checks[0].Status)
	}
	if model.checks[0].Detail != "OK" {
		t.Errorf("after result: detail = %q, want %q", model.checks[0].Detail, "OK")
	}
}

// =============================================================================
// TestHealth_AllDone — RED (triangulation)
// =============================================================================

func TestHealth_AllDone(t *testing.T) {
	m := NewHealthModel()
	m.running = true
	m.checks = []HealthCheck{
		{Name: "Config", Status: StepRunning, Detail: ""},
		{Name: "Backup dir", Status: StepRunning, Detail: ""},
	}

	// Complete both checks.
	newM, _ := m.Update(healthCheckResultMsg{index: 0, detail: "OK"})
	newM, _ = newM.Update(healthCheckResultMsg{index: 1, detail: "OK"})
	model := newM.(HealthModel)

	if model.running {
		t.Error("all done: running = true, want false")
	}
}

// =============================================================================
// TestHealth_WindowSize — RED (triangulation)
// =============================================================================

func TestHealth_WindowSize(t *testing.T) {
	m := NewHealthModel()

	newM, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	model := newM.(HealthModel)

	if model.width != 100 {
		t.Errorf("WindowSize width = %d, want 100", model.width)
	}
	if model.height != 30 {
		t.Errorf("WindowSize height = %d, want 30", model.height)
	}
}

// =============================================================================
// TestHealth_View_Idle — RED (triangulation)
// =============================================================================

func TestHealth_View_Idle(t *testing.T) {
	m := NewHealthModel()
	m.width = 80
	m.height = 24

	output := m.View().Content

	if !strings.Contains(output, "Press enter to run health check") {
		t.Errorf("idle view missing prompt: %q", output)
	}
}

// =============================================================================
// TestHealth_View_RerunFooter — RED (triangulation)
// =============================================================================

func TestHealth_View_RerunFooter(t *testing.T) {
	m := NewHealthModel()
	m.width = 80
	m.height = 24
	m.checks = []HealthCheck{
		{Name: "Config", Status: StepDone, Detail: "OK"},
	}

	output := m.View().Content

	if !strings.Contains(output, "rerun") {
		t.Errorf("done view missing 'rerun' footer: %q", output)
	}
}

// =============================================================================
// TestHealth_ResultOutOfBounds — RED (triangulation)
// =============================================================================

func TestHealth_ResultOutOfBounds(t *testing.T) {
	tests := []struct {
		name  string
		index int
	}{
		{"negative index", -1},
		{"index beyond length", 99},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewHealthModel()
			m.running = true
			m.checks = []HealthCheck{
				{Name: "Config", Status: StepRunning},
			}

			// Should not panic.
			newM, _ := m.Update(healthCheckResultMsg{index: tt.index})
			model := newM.(HealthModel)

			// Check state is unchanged.
			if model.checks[0].Status != StepRunning {
				t.Errorf("out-of-bounds result: status changed to %v, want StepRunning",
					model.checks[0].Status)
			}
		})
	}
}
