// Package screens provides screen-specific render functions and sub-models
// for the bak-cli TUI. This file contains TDD tests written BEFORE the
// production code (strict RED phase) for the progress screen.
package screens

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// =============================================================================
// TestNewProgressModel — RED (progress.go does not exist yet)
// =============================================================================

func TestNewProgressModel(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewProgressModel()

	// Initial state: steps should be empty.
	if len(m.steps) != 0 {
		t.Errorf("NewProgressModel().steps length = %d, want 0", len(m.steps))
	}

	// Running should be false initially.
	if m.running {
		t.Error("NewProgressModel().running = true, want false")
	}

	// Spinner should be initialized (non-zero model).
	_ = m.spinner.View() // should not panic
}

// =============================================================================
// TestProgress_Update_StepMsg — RED
// =============================================================================

func TestProgress_Update_StepMsg(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewProgressModel()
	m.Width = 80
	m.Height = 24

	// Send a progress step.
	newModel, _ := m.Update(ProgressStepMsg{
		Step:    "Scanning files",
		Current: 1,
		Total:   5,
	})
	result := newModel.(ProgressModel)

	if len(result.steps) != 1 {
		t.Fatalf("after ProgressStepMsg: steps length = %d, want 1", len(result.steps))
	}
	if result.steps[0].Name != "Scanning files" {
		t.Errorf("step[0].Name = %q, want %q", result.steps[0].Name, "Scanning files")
	}
	// First step is the current step, so it's Running.
	if result.steps[0].Status != StepRunning {
		t.Errorf("step[0].Status = %v, want StepRunning", result.steps[0].Status)
	}
	if !result.running {
		t.Error("after ProgressStepMsg: running = false, want true")
	}

	// Send another progress step.
	newModel, _ = result.Update(ProgressStepMsg{
		Step:    "Compressing",
		Current: 2,
		Total:   5,
	})
	result = newModel.(ProgressModel)

	if len(result.steps) != 2 {
		t.Fatalf("after second ProgressStepMsg: steps length = %d, want 2", len(result.steps))
	}
	// Previous step is now marked done.
	if result.steps[0].Name != "Scanning files" {
		t.Errorf("step[0] preserved: Name = %q, want 'Scanning files'", result.steps[0].Name)
	}
	if result.steps[0].Status != StepDone {
		t.Errorf("step[0].Status = %v, want StepDone", result.steps[0].Status)
	}
	// Latest step is running.
	if result.steps[1].Name != "Compressing" {
		t.Errorf("step[1].Name = %q, want 'Compressing'", result.steps[1].Name)
	}
	if result.steps[1].Status != StepRunning {
		t.Errorf("step[1].Status = %v, want StepRunning", result.steps[1].Status)
	}
}

// =============================================================================
// TestProgress_Update_DoneMsg — RED
// =============================================================================

func TestProgress_Update_DoneMsg(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewProgressModel()
	m.running = true
	m.steps = []Step{
		{Name: "Scanning files", Status: StepDone},
		{Name: "Compressing", Status: StepRunning},
	}

	// Send Done message.
	_, cmd := m.Update(ProgressDoneMsg{})

	if cmd == nil {
		t.Fatal("Update(ProgressDoneMsg) returned nil cmd, want ScreenBackMsg")
	}
	msg := cmd()
	if _, ok := msg.(ScreenBackMsg); !ok {
		t.Errorf("Update(ProgressDoneMsg) returned %T, want ScreenBackMsg", msg)
	}
}

// =============================================================================
// TestProgress_Update_Back — RED
// =============================================================================

func TestProgress_Update_Back(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	tests := []struct {
		name    string
		code    rune
		running bool
		wantCmd bool
	}{
		{"q when not running", 'q', false, true},
		{"esc when not running", 27, false, true},
		{"q when running is blocked", 'q', true, false},
		{"esc when running is blocked", 27, true, false},
	}

	for _, tt := range tests { //nolint:paralleltest // subtests share table/struct state
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
			m := NewProgressModel()
			m.running = tt.running

			_, cmd := m.Update(tea.KeyPressMsg{Code: tt.code})

			if tt.wantCmd {
				if cmd == nil {
					t.Error("expected back command, got nil")
				}
			} else {
				if cmd != nil {
					t.Error("expected no command while running, got non-nil")
				}
			}
		})
	}
}

// =============================================================================
// TestProgress_View_Running — RED
// =============================================================================

func TestProgress_View_Running(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewProgressModel()
	m.Width = 80
	m.Height = 24
	m.running = true
	m.steps = []Step{
		{Name: "Scanning files", Status: StepDone},
		{Name: "Compressing", Status: StepRunning},
		{Name: "Uploading", Status: StepPending},
	}
	// Set progress to 40% and process through Update.
	pgCmd := m.progress.SetPercent(0.4)
	pgMsg := pgCmd()
	m.progress, _ = m.progress.Update(pgMsg)

	output := m.View().Content

	// Must contain step names.
	if !strings.Contains(output, "Scanning files") {
		t.Error("View() missing step 'Scanning files'")
	}
	if !strings.Contains(output, "Compressing") {
		t.Error("View() missing step 'Compressing'")
	}
	if !strings.Contains(output, "Uploading") {
		t.Error("View() missing step 'Uploading'")
	}

	// Progress bar must render (check for progress bar characters or percentage).
	// Spring animation means exact percentage varies, but we verify bar output is non-empty.
	if !strings.Contains(output, "%") {
		t.Errorf("View() missing progress bar (no '%%' found):\n%s", output)
	}

	// Output must be non-empty.
	if len(output) == 0 {
		t.Error("View() returned empty string")
	}
}

// =============================================================================
// TestProgress_View_Complete — RED
// =============================================================================

func TestProgress_View_Complete(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewProgressModel()
	m.Width = 80
	m.Height = 24
	m.running = false
	m.steps = []Step{
		{Name: "Scanning files", Status: StepDone},
		{Name: "Compressing", Status: StepDone},
		{Name: "Uploading", Status: StepDone},
	}
	pgCmd := m.progress.SetPercent(1.0)
	pgMsg := pgCmd()
	m.progress, _ = m.progress.Update(pgMsg)

	output := m.View().Content

	// All steps should show checkmark.
	doneCount := strings.Count(output, "✓")
	if doneCount < 3 {
		t.Errorf("View() complete: ✓ count = %d, want >= 3", doneCount)
	}

	// Should indicate completion.
	if !strings.Contains(output, "Complete") {
		t.Error("View() complete: missing 'Complete!' indication")
	}
}

// =============================================================================
// TestProgress_View_EmptySteps — RED
// =============================================================================

func TestProgress_View_EmptySteps(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewProgressModel()
	m.Width = 80
	m.Height = 24

	output := m.View().Content

	// Should not panic for empty steps.
	if len(output) == 0 {
		t.Error("View() with empty steps returned empty string")
	}
}

// =============================================================================
// TestProgress_Init — RED
// =============================================================================

func TestProgress_Init(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewProgressModel()

	cmd := m.Init()

	if cmd == nil {
		t.Error("Init() returned nil, want spinner.Tick")
	}
}

// =============================================================================
// TestProgress_SpinnerTick — RED
// =============================================================================

func TestProgress_SpinnerTick(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewProgressModel()
	m.running = true

	// Simulate a spinner tick. Spinner tick msg comes from spinner.Tick().
	// The specific type is spinner.TickMsg which is internal to bubbles.
	// We trigger a tick by calling the spinner's Tick method.
	tickMsg := m.spinner.Tick()

	newModel, cmd := m.Update(tickMsg)
	result := newModel.(ProgressModel)

	// After a tick, another Tick should be requested to keep animation going.
	if result.running && cmd == nil {
		t.Error("after spinner tick while running: cmd = nil, want new Tick")
	}

	// When not running, no new Tick should be requested.
	m2 := NewProgressModel()
	m2.running = false
	_, cmd2 := m2.Update(tickMsg)
	if cmd2 != nil {
		t.Error("after spinner tick while not running: expected nil cmd")
	}
}

// =============================================================================
// TestProgress_Update_WindowSize — RED
// =============================================================================

func TestProgress_Update_WindowSize(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewProgressModel()

	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	result := newModel.(ProgressModel)

	if result.Width != 100 {
		t.Errorf("WindowSize width = %d, want 100", result.Width)
	}
	if result.Height != 30 {
		t.Errorf("WindowSize height = %d, want 30", result.Height)
	}
}

// =============================================================================
// TestProgress_Running — exercises the Running() accessor across the full
// state machine: idle (false) → running (true) → done (false). Each transition
// is driven by a real message through Update, proving Running() reflects the
// model's internal running flag.
// =============================================================================

func TestProgress_Running(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	tests := []struct {
		name string
		// msgs is the ordered sequence of messages applied to a fresh model.
		msgs []tea.Msg
		want bool
	}{
		{
			name: "fresh model is not running",
			msgs: nil,
			want: false,
		},
		{
			name: "running after a step message",
			msgs: []tea.Msg{ProgressStepMsg{Step: "copying", Current: 1, Total: 2}},
			want: true,
		},
		{
			name: "not running after done message",
			msgs: []tea.Msg{
				ProgressStepMsg{Step: "copying", Current: 1, Total: 2},
				ProgressDoneMsg{},
			},
			want: false,
		},
	}

	for _, tt := range tests { //nolint:paralleltest // subtests share table/struct state
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
			var model tea.Model = NewProgressModel()
			for _, msg := range tt.msgs {
				model, _ = model.Update(msg)
			}
			got := model.(ProgressModel).Running()
			if got != tt.want {
				t.Errorf("Running() = %v, want %v", got, tt.want)
			}
		})
	}
}
