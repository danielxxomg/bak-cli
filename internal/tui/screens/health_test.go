package screens

import (
	"strings"
	"testing"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
)

// =============================================================================
// TestNewHealthModel — RED (health.go does not exist yet)
// =============================================================================

func TestNewHealthModel(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewHealthModel()

	if len(m.checks) != 0 {
		t.Errorf("NewHealthModel().checks len = %d, want 0", len(m.checks))
	}
	if m.running {
		t.Error("NewHealthModel().running = true, want false")
	}
}

// =============================================================================
// Phase 3: Init nil-return coverage
// =============================================================================

func TestHealthModel_Init_StartsSpinner(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewHealthModel()
	cmd := m.Init()
	if cmd == nil {
		t.Error("Init() returned nil, want spinner.Tick so the running step rotates")
	}
}

// =============================================================================
// TestHealth_Update_Run — RED
// =============================================================================

func TestHealth_Update_Run(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
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

func TestHealth_Update_Back(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	tests := []struct {
		name string
		key  rune
	}{
		{"q goes back", 'q'},
		{"esc goes back", 27},
	}

	for _, tt := range tests { //nolint:paralleltest // subtests share table/struct state
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
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

func TestHealth_View_Running(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewHealthModel()
	// Custom spinner with a unique frame so the live indicator is deterministic
	// and distinct from the old static "\u28f9" glyph.
	m.spinner = spinner.New(spinner.WithSpinner(spinner.Spinner{
		Frames: []string{"alpha", "beta", "gamma", "delta"},
		FPS:    time.Second / 10, //nolint:mnd
	}))
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
	// The running row must no longer render the old static glyph.
	if strings.Contains(output, "\u28f9") {
		t.Errorf("View() running still shows static indicator \\u28f9, want live spinner frame: %q", output)
	}
}

// TestHealth_View_RunningStepShowsSpinnerFrame verifies the running health
// check row renders the live spinner.Model frame (m.spinner.View()) instead of
// a static glyph (REQ-TP-002). Mirrors the progress screen assertion.
func TestHealth_View_RunningStepShowsSpinnerFrame(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewHealthModel()
	m.spinner = spinner.New(spinner.WithSpinner(spinner.Spinner{
		Frames: []string{"alpha", "beta", "gamma", "delta"},
		FPS:    time.Second / 10, //nolint:mnd
	}))
	m.width = 80
	m.height = 24
	m.running = true
	m.checks = []HealthCheck{{Name: "Config exists", Status: StepRunning}}

	// Advance the spinner two ticks → frame index 2 → "gamma".
	for range 2 { //nolint:mnd
		tickMsg := m.spinner.Tick()
		nm, _ := m.Update(tickMsg)
		m = nm.(HealthModel)
	}

	output := m.View().Content
	const wantFrame = "gamma"
	var stepRow string
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "Config exists") {
			stepRow = line
			break
		}
	}
	if stepRow == "" {
		t.Fatalf("running check row not found in output:\n%s", output)
	}
	if !strings.Contains(stepRow, wantFrame) {
		t.Errorf("running check row must show live spinner frame %q, got row %q", wantFrame, stepRow)
	}
}

// TestHealth_SpinnerTick verifies the spinner advances while running and
// stops ticking when idle (mirrors the progress screen TickMsg handling).
func TestHealth_SpinnerTick(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	// While running, a TickMsg advances the spinner and re-issues a tick.
	m := NewHealthModel()
	m.running = true
	tickMsg := m.spinner.Tick()

	nm, cmd := m.Update(tickMsg)
	result := nm.(HealthModel)
	if cmd == nil {
		t.Error("after spinner tick while running: cmd = nil, want new Tick")
	}
	_ = result

	// While idle, a TickMsg does not re-tick (stops the animation).
	idle := NewHealthModel()
	idle.running = false
	_, cmd2 := idle.Update(idle.spinner.Tick())
	if cmd2 != nil {
		t.Error("after spinner tick while idle: expected nil cmd, got non-nil")
	}
}

// =============================================================================
// TestHealth_View_Done — RED
// =============================================================================

func TestHealth_View_Done(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
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
// TestHealth_View_MinSizeGuard — threshold guard at 40×12
// =============================================================================

func TestHealth_View_MinSizeGuard(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	tests := []struct {
		name     string
		width    int
		height   int
		tooSmall bool
	}{
		{"below width (29x20)", 29, 20, true},
		{"below height (60x14)", 60, 14, true},
		{"both below (20x10)", 20, 10, true},
		{"exactly min (30x15)", 30, 15, false},
		{"above min (80x24)", 80, 24, false},
	}

	for _, tt := range tests { //nolint:paralleltest // subtests share table/struct state
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
			m := NewHealthModel()
			m.width = tt.width
			m.height = tt.height

			output := m.View().Content

			if tt.tooSmall {
				if !strings.Contains(output, "Terminal too small") {
					t.Errorf("View() %dx%d: expected 'Terminal too small', got %q",
						tt.width, tt.height, output)
				}
			} else {
				if strings.Contains(output, "Terminal too small") {
					t.Errorf("View() %dx%d: got 'Terminal too small', expected normal content",
						tt.width, tt.height)
				}
			}
		})
	}
}

// TestHealth_View_TooSmall verifies the too-small view shows
// the warning message (legacy test, kept for coverage).
func TestHealth_View_TooSmall(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
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

func TestHealth_ResultMessage(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
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

func TestHealth_AllDone(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
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

func TestHealth_WindowSize(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
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

func TestHealth_View_Idle(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
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

func TestHealth_View_RerunFooter(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
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

	// Phase 3: after RenderHelp replacement, the footer uses "q back • enter rerun"
	if !strings.Contains(output, "back") {
		t.Errorf("done view help bar missing 'back': %q", output)
	}
}

// =============================================================================
// TestHealth_View_HelpBar_Idle — RED (Phase 3: help bar persistence)
// =============================================================================

func TestHealth_View_HelpBar_Idle(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewHealthModel()
	m.width = 80
	m.height = 24

	output := m.View().Content

	// Idle state must show the prompt...
	if !strings.Contains(output, "Press enter to run health check") {
		t.Errorf("idle view missing prompt: %q", output)
	}

	// ...AND the help bar: enter run • q back
	if !strings.Contains(output, "back") {
		t.Errorf("idle view help bar missing 'back': %q", output)
	}
}

// =============================================================================
// TestHealth_ResultOutOfBounds — RED (triangulation)
// =============================================================================

func TestHealth_ResultOutOfBounds(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	tests := []struct {
		name  string
		index int
	}{
		{"negative index", -1},
		{"index beyond length", 99},
	}

	for _, tt := range tests { //nolint:paralleltest // subtests share table/struct state
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
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
