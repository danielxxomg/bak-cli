// Package screens provides screen-specific render functions and sub-models
// for the bak-cli TUI. This file contains strict-TDD tests for RestoreModel
// written BEFORE the production code.
package screens

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/danielxxomg/bak-cli/internal/tui/components"
)

// restoreTestDeps provides test function fields for RestoreModel tests.
type restoreTestDeps struct {
	listBackupsFn func() ([]BackupInfo, error)
	runRestoreFn  func(backupID string, dryRun bool) (string, error)
}

// =============================================================================
// TestRestore_NewRestoreModel — RED (restore.go does not exist yet)
// =============================================================================

func TestRestore_NewRestoreModel(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	deps := restoreTestDeps{
		listBackupsFn: func() ([]BackupInfo, error) {
			return []BackupInfo{
				{ID: "20240101-120000", Date: "2024-01-01 12:00:00", Size: "2MB", Status: "ok", Cloud: "none"},
			}, nil
		},
	}

	m := NewRestoreModel(deps.listBackupsFn, deps.runRestoreFn)

	if m.State != restoreStateList {
		t.Errorf("initial state = %v, want restoreStateList (%d)", m.State, restoreStateList)
	}
	if m.Cursor != 0 {
		t.Errorf("initial cursor = %d, want 0", m.Cursor)
	}
}

// =============================================================================
// TestRestore_Init_PopulatesBackups — RED
// =============================================================================

func TestRestore_Init_PopulatesBackups(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	deps := restoreTestDeps{
		listBackupsFn: func() ([]BackupInfo, error) {
			return []BackupInfo{
				{ID: "backup-1", Date: "2024-01-01", Size: "1MB", Status: "ok", Cloud: "none"},
				{ID: "backup-2", Date: "2024-02-01", Size: "2MB", Status: "ok", Cloud: "github"},
			}, nil
		},
	}

	m := NewRestoreModel(deps.listBackupsFn, deps.runRestoreFn)

	cmd := m.Init()
	// Init should return a command that triggers backup loading.
	if cmd == nil {
		t.Fatal("Init() returned nil cmd, want backup load command")
	}

	// Execute the command to trigger loading.
	msg := cmd()
	// Apply the load result.
	newModel, _ := m.Update(msg)
	rm := newModel.(RestoreModel)

	if len(rm.Backups) != 2 {
		t.Fatalf("backups length = %d, want 2", len(rm.Backups))
	}
	if rm.Backups[0].ID != "backup-1" {
		t.Errorf("backups[0].ID = %q, want %q", rm.Backups[0].ID, "backup-1")
	}
	if rm.Backups[1].ID != "backup-2" {
		t.Errorf("backups[1].ID = %q, want %q", rm.Backups[1].ID, "backup-2")
	}
}

// =============================================================================
// TestRestore_EmptyState — RED
// =============================================================================

func TestRestore_EmptyState(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	deps := restoreTestDeps{
		listBackupsFn: func() ([]BackupInfo, error) {
			return nil, nil // no backups
		},
	}

	m := NewRestoreModel(deps.listBackupsFn, deps.runRestoreFn)
	m.Width = 80
	m.Height = 24
	m.State = restoreStateList

	output := m.View().Content

	if !strings.Contains(output, "No backups found") {
		t.Errorf("empty state missing 'No backups found': %q", output)
	}
}

// TestRestore_EmptyState_Styled verifies the empty restore list renders the
// shared styled empty-state block (icon + message + hint) via
// components.RenderEmptyState, not a bare string (tui-personality REQ-TP-007).
func TestRestore_EmptyState_Styled(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	deps := restoreTestDeps{
		listBackupsFn: func() ([]BackupInfo, error) {
			return nil, nil // no backups
		},
	}

	m := NewRestoreModel(deps.listBackupsFn, deps.runRestoreFn)
	m.Width = 80
	m.Height = 24
	m.State = restoreStateList

	output := m.View().Content

	if !strings.Contains(output, "No backups found") {
		t.Errorf("styled empty state missing message 'No backups found': %q", output)
	}
	if !strings.Contains(output, "bak backup") {
		t.Errorf("styled empty state missing hint 'bak backup': %q", output)
	}
}

// =============================================================================
// TestRestore_StateTransition_ListToDryRun — RED
// =============================================================================

func TestRestore_StateTransition_ListToDryRun(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	var dryRunCalled string
	deps := restoreTestDeps{
		listBackupsFn: func() ([]BackupInfo, error) {
			return []BackupInfo{
				{ID: "backup-1", Date: "2024-01-01", Size: "1MB", Status: "ok", Cloud: "none"},
			}, nil
		},
		runRestoreFn: func(backupID string, dryRun bool) (string, error) {
			if dryRun {
				dryRunCalled = backupID
				return "diff: 3 files changed", nil
			}
			return "", nil
		},
	}

	m := NewRestoreModel(deps.listBackupsFn, deps.runRestoreFn)
	m.Width = 80
	m.Height = 24
	m.State = restoreStateList

	// Manually populate backups.
	m.Backups = []BackupInfo{
		{ID: "backup-1", Date: "2024-01-01", Size: "1MB", Status: "ok", Cloud: "none"},
	}

	// Press Enter on selected backup (cursor 0) to trigger dry run.
	newModel, cmd := m.Update(tea.KeyPressMsg{Code: '\r'})
	rm := newModel.(RestoreModel)

	if cmd != nil {
		// Execute the command to start the dry run.
		msg := cmd()
		newModel2, _ := rm.Update(msg)
		rm = newModel2.(RestoreModel)
	}

	// State should transition to dryRun.
	if rm.State != restoreStateDryRun {
		t.Errorf("state after Enter = %v, want restoreStateDryRun", rm.State)
	}
	if dryRunCalled != "backup-1" {
		t.Errorf("dryRun called with %q, want %q", dryRunCalled, "backup-1")
	}
	if !strings.Contains(rm.DryRunOutput, "3 files changed") {
		t.Errorf("DryRunOutput = %q, want to contain %q", rm.DryRunOutput, "3 files changed")
	}
}

// =============================================================================
// TestRestore_StateTransition_DryRunToConfirm — RED
// =============================================================================

func TestRestore_StateTransition_DryRunToConfirm(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	deps := restoreTestDeps{
		runRestoreFn: func(backupID string, dryRun bool) (string, error) {
			return "no changes", nil
		},
	}

	m := NewRestoreModel(deps.listBackupsFn, deps.runRestoreFn)
	m.Width = 80
	m.Height = 24
	m.State = restoreStateDryRun
	m.SelectedID = "backup-1"
	m.Backups = []BackupInfo{{ID: "backup-1"}}

	// Press Enter on dry run screen to confirm.
	newModel, _ := m.Update(tea.KeyPressMsg{Code: '\r'})
	rm := newModel.(RestoreModel)

	if rm.State != restoreStateConfirm {
		t.Errorf("state after Enter on dryRun = %v, want restoreStateConfirm", rm.State)
	}
}

// =============================================================================
// TestRestore_ConfirmExecute — RED
// =============================================================================

func TestRestore_ConfirmExecute(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	var actualExecuted string
	deps := restoreTestDeps{
		runRestoreFn: func(backupID string, dryRun bool) (string, error) {
			if !dryRun {
				actualExecuted = backupID
				return "restored successfully", nil
			}
			return "diff output", nil
		},
	}

	m := NewRestoreModel(deps.listBackupsFn, deps.runRestoreFn)
	m.Width = 80
	m.Height = 24
	m.State = restoreStateConfirm
	m.SelectedID = "backup-1"
	m.Modal = makeTestModal()

	// Simulate modal confirm.
	newModel, cmd := m.Update(components.ModalResultMsg{Confirmed: true})
	rm := newModel.(RestoreModel)

	if cmd != nil {
		msg := cmd()
		newModel2, _ := rm.Update(msg)
		rm = newModel2.(RestoreModel)
	}

	if actualExecuted != "backup-1" {
		t.Errorf("RunRestore executed with %q, want %q", actualExecuted, "backup-1")
	}
	// After confirm + execution, state should be restoreStateDone (4)
	// since the async cmd completes immediately in test context.
	if rm.State != restoreStateDone {
		t.Errorf("state after confirm+exec = %v, want restoreStateDone", rm.State)
	}
}

// =============================================================================
// TestRestore_ConfirmCancel — RED
// =============================================================================

func TestRestore_ConfirmCancel(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	var restoreCalled bool
	deps := restoreTestDeps{
		runRestoreFn: func(backupID string, dryRun bool) (string, error) {
			restoreCalled = true
			return "", nil
		},
	}

	m := NewRestoreModel(deps.listBackupsFn, deps.runRestoreFn)
	m.Width = 80
	m.Height = 24
	m.State = restoreStateConfirm
	m.Modal = makeTestModal()

	// Simulate modal cancel.
	newModel, _ := m.Update(components.ModalResultMsg{Confirmed: false})
	rm := newModel.(RestoreModel)

	if restoreCalled {
		t.Error("RunRestore was called, should NOT be called on cancel")
	}
	// After cancel, return to list state.
	if rm.State != restoreStateList {
		t.Errorf("state after cancel = %v, want restoreStateList", rm.State)
	}
	if rm.Modal != nil {
		t.Error("Modal should be nil after cancel")
	}
}

// =============================================================================
// TestRestore_ErrorHandling — RED
// =============================================================================

func TestRestore_ErrorHandling(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	deps := restoreTestDeps{
		listBackupsFn: func() ([]BackupInfo, error) {
			return nil, errors.New("disk read error")
		},
	}

	m := NewRestoreModel(deps.listBackupsFn, deps.runRestoreFn)
	m.Width = 80
	m.Height = 24

	cmd := m.Init()
	msg := cmd()
	newModel, _ := m.Update(msg)
	rm := newModel.(RestoreModel)

	if rm.Err == nil {
		t.Error("error should be set when listBackups fails")
	}
}

// =============================================================================
// TestRestore_View_DryRun — RED
// =============================================================================

func TestRestore_View_DryRun(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewRestoreModel(nil, nil)
	m.Width = 80
	m.Height = 24
	m.State = restoreStateDryRun
	m.DryRunOutput = "--- a/file.txt\n+++ b/file.txt\n+added line"
	m.SelectedID = "backup-1"

	output := m.View().Content

	if !strings.Contains(output, "backup-1") {
		t.Errorf("dry run view missing backup ID: %q", output)
	}
	if !strings.Contains(output, "Dry Run") {
		t.Errorf("dry run view missing 'Dry Run' heading: %q", output)
	}
}

// =============================================================================
// TestRestore_View_Confirm — RED
// =============================================================================

func TestRestore_View_Confirm(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewRestoreModel(nil, nil)
	m.Width = 80
	m.Height = 24
	m.State = restoreStateConfirm
	m.SelectedID = "backup-1"

	output := m.View().Content

	if !strings.Contains(output, "Confirm") {
		t.Errorf("confirm view missing 'Confirm': %q", output)
	}
}

func makeTestModal() *components.ModalModel {
	modal := components.NewModal("Confirm Restore", "This will overwrite current config.", []string{"Confirm", "Cancel"})
	return &modal
}

// TestRestore_ConfirmModal_KeyForwarding verifies that keypresses in the
// confirm state reach the modal so the user can confirm/cancel via the
// keyboard (Enter confirms, Escape cancels). Without forwarding the modal is
// rendered but non-interactive.
func TestRestore_ConfirmModal_KeyForwarding(t *testing.T) { //nolint:paralleltest // matches established codebase convention across all tui tests
	makeConfirmModel := func() RestoreModel {
		m := NewRestoreModel(nil, nil)
		m.Width = 80
		m.Height = 24
		m.State = restoreStateConfirm
		m.SelectedID = "backup-1"
		modal := components.NewModal("Confirm Restore", "msg", []string{"Confirm", "Cancel"})
		m.Modal = &modal
		return m
	}

	t.Run("enter confirms", func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
		m := makeConfirmModel()
		_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
		if cmd == nil {
			t.Fatal("enter in confirm modal returned nil cmd, want ModalResultMsg")
		}
		msg := cmd()
		result, ok := msg.(components.ModalResultMsg)
		if !ok {
			t.Fatalf("enter cmd returned %T, want ModalResultMsg", msg)
		}
		if !result.Confirmed {
			t.Error("enter on first button should confirm, got Confirmed=false")
		}
	})

	t.Run("escape cancels", func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
		m := makeConfirmModel()
		_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
		if cmd == nil {
			t.Fatal("escape in confirm modal returned nil cmd, want ModalResultMsg")
		}
		msg := cmd()
		result, ok := msg.(components.ModalResultMsg)
		if !ok {
			t.Fatalf("escape cmd returned %T, want ModalResultMsg", msg)
		}
		if result.Confirmed {
			t.Error("escape should cancel, got Confirmed=true")
		}
	})
}

// =============================================================================
// Phase 3: Render helper coverage for restore.go (0% before backfill)
// =============================================================================

func TestRestore_RenderErrorState(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewRestoreModel(nil, nil)
	m.Width = 80
	m.Height = 24
	m.Err = errors.New("connection refused")

	output := m.View().Content

	// Error state shows the error message.
	if !strings.Contains(output, "connection refused") {
		t.Errorf("renderErrorState missing error: %q", output)
	}
	if !strings.Contains(output, "Error") {
		t.Errorf("renderErrorState missing 'Error': %q", output)
	}
	if !strings.Contains(output, "q") {
		t.Errorf("renderErrorState missing help key: %q", output)
	}
}

// TestRestore_View_TooSmall verifies the dry-run screen guards against
// terminals below the minimum dimensions (AGENTS.md TUI Responsiveness).
func TestRestore_View_TooSmall(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewRestoreModel(nil, nil)
	m.Width = 20
	m.Height = 10
	m.State = restoreStateList
	m.Backups = []BackupInfo{{ID: "x"}}

	output := m.View().Content

	if !strings.Contains(output, "Terminal too small") {
		t.Errorf("too-small guard missing message: %q", output)
	}
}

func TestRestore_RenderBackupList(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewRestoreModel(nil, nil)
	m.Width = 80
	m.Height = 24
	m.State = restoreStateList
	m.Backups = []BackupInfo{
		{ID: "backup-1", Date: "2024-01-01", Size: "1MB", Status: "ok", Cloud: "none"},
		{ID: "backup-2", Date: "2024-02-01", Size: "2MB", Status: "ok", Cloud: "github"},
	}
	m.Cursor = 0

	output := m.View().Content

	if !strings.Contains(output, "backup-1") {
		t.Errorf("renderBackupList missing backup-1: %q", output)
	}
	if !strings.Contains(output, "backup-2") {
		t.Errorf("renderBackupList missing backup-2: %q", output)
	}
	if !strings.Contains(output, "navigate") || !strings.Contains(output, "select") {
		t.Errorf("renderBackupList missing help: %q", output)
	}
}

func TestRestore_RenderBackupList_Empty(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewRestoreModel(nil, nil)
	m.Width = 80
	m.Height = 24
	m.State = restoreStateList
	m.Backups = nil

	output := m.View().Content

	// Empty state should show "No backups found".
	if !strings.Contains(output, "No backups found") {
		t.Errorf("empty backup list missing message: %q", output)
	}
}

// TestRestore_ListNavigation_CursorBounds verifies that list navigation keeps
// the cursor in a valid range when it starts negative or out-of-bounds
// (AGENTS.md TUI Testing: MUST test negative/out-of-bounds cursor edge cases).
func TestRestore_ListNavigation_CursorBounds(t *testing.T) { //nolint:paralleltest // matches established codebase convention across all tui tests
	backups := []BackupInfo{{ID: "1"}, {ID: "2"}, {ID: "3"}}

	tests := []struct {
		name     string
		startCur int
		key      tea.KeyPressMsg
	}{
		{"negative cursor + j", -1, tea.KeyPressMsg{Code: 'j'}},
		{"out-of-bounds cursor + j", 99, tea.KeyPressMsg{Code: 'j'}},
		{"negative cursor + k", -1, tea.KeyPressMsg{Code: 'k'}},
		{"out-of-bounds cursor + k", 99, tea.KeyPressMsg{Code: 'k'}},
	}

	for _, tt := range tests { //nolint:paralleltest // subtests share table/struct state
		t.Run(tt.name, func(t *testing.T) {
			m := NewRestoreModel(nil, nil)
			m.Width = 80
			m.Height = 24
			m.State = restoreStateList
			m.Backups = backups
			m.Cursor = tt.startCur

			nm, _ := m.Update(tt.key)
			r := nm.(RestoreModel)

			if r.Cursor < 0 || r.Cursor >= len(backups) {
				t.Errorf("%s: cursor = %d, want in [0, %d)", tt.name, r.Cursor, len(backups))
			}
		})
	}
}

func TestRestore_RenderRunning(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewRestoreModel(nil, nil)
	m.Width = 80
	m.Height = 24
	m.State = restoreStateRunning
	m.SelectedID = "backup-1"

	output := m.View().Content

	if !strings.Contains(output, "Restore") {
		t.Errorf("renderRunning missing title: %q", output)
	}
	if !strings.Contains(output, "backup-1") {
		t.Errorf("renderRunning missing backup ID: %q", output)
	}
}

func TestRestore_RenderDone(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewRestoreModel(nil, nil)
	m.Width = 80
	m.Height = 24
	m.State = restoreStateDone

	output := m.View().Content

	if !strings.Contains(output, "Restore") {
		t.Errorf("renderDone missing title: %q", output)
	}
	if !strings.Contains(output, "successfully") {
		t.Errorf("renderDone missing success message: %q", output)
	}
	if !strings.Contains(output, "back to menu") {
		t.Errorf("renderDone missing footer: %q", output)
	}
}

func TestRestore_RenderDone_Error(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewRestoreModel(nil, nil)
	m.Width = 80
	m.Height = 24
	m.State = restoreStateDone
	m.Err = errors.New("disk full")

	output := m.View().Content

	if !strings.Contains(output, "disk full") {
		t.Errorf("renderDone error missing message: %q", output)
	}
	if !strings.Contains(output, "Error") {
		t.Errorf("renderDone error missing 'Error': %q", output)
	}
}

// =============================================================================
// Phase 3: Viewport Dry-Run (PR 2 — Tier 2a) — RED
// =============================================================================

// buildLongDiff returns an n-line diff string where each line is unique and
// numbered, so tests can assert which lines are visible inside the viewport.
func buildLongDiff(n int) string {
	var b strings.Builder
	for i := 1; i <= n; i++ {
		fmt.Fprintf(&b, "diff line %d\n", i)
	}
	return b.String()
}

// TestRestore_DryRun_ViewportRendersBoundedContent verifies that the dry-run
// diff renders inside a bubbles/viewport (REQ-TP-005): the first diff line is
// visible (content was set + rendered by viewport.View()) while the last line
// is NOT (the viewport bounds content to its height instead of dumping the raw
// string). On the current raw-dump implementation this fails because line 80
// leaks into the output.
func TestRestore_DryRun_ViewportRendersBoundedContent(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewRestoreModel(nil, nil)
	m.SelectedID = "backup-1"
	// Size the embedded viewport via WindowSizeMsg.
	nm, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = nm.(RestoreModel)
	// Deliver the dry-run result → viewport.SetContent + state transition.
	nm, _ = m.Update(restoreDryRunResultMsg{output: buildLongDiff(80)})
	m = nm.(RestoreModel)

	if m.State != restoreStateDryRun {
		t.Fatalf("state = %v, want restoreStateDryRun", m.State)
	}

	out := m.View().Content

	// Heading + selected id must still be rendered.
	if !strings.Contains(out, "Dry Run") {
		t.Errorf("dry-run view missing 'Dry Run' heading: %q", out)
	}
	if !strings.Contains(out, "backup-1") {
		t.Errorf("dry-run view missing selected id: %q", out)
	}
	// The viewport must render the first diff line (content was set).
	if !strings.Contains(out, "diff line 1") {
		t.Errorf("viewport did not render first diff line: %q", out)
	}
	// The viewport bounds content to its height: the last line must NOT leak
	// (a raw dump would include it).
	if strings.Contains(out, "diff line 80") {
		t.Errorf("viewport leaked last diff line — content not bounded (raw dump, not viewport): %q", out)
	}
}

// sizedDryRunModel returns a RestoreModel sized to w x h with a multi-line diff
// loaded into the viewport and the state set to restoreStateDryRun.
func sizedDryRunModel(w, h, diffLines int) RestoreModel {
	m := NewRestoreModel(nil, nil)
	m.SelectedID = "backup-1"
	nm, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	m = nm.(RestoreModel)
	nm, _ = m.Update(restoreDryRunResultMsg{output: buildLongDiff(diffLines)})
	m = nm.(RestoreModel)
	return m
}

// TestRestore_DryRun_ScrollKeys verifies the viewport scroll bindings in
// restoreStateDryRun (REQ-TP-005 / tui-interactive-preview). j/k, arrows, and
// PgUp/PgDn forward to the viewport's default keymap; g/G jump to top/bottom.
// Table-driven so each binding is exercised with its own key + expectation.
func TestRestore_DryRun_ScrollKeys(t *testing.T) { //nolint:paralleltest // matches established codebase convention across all tui tests
	tests := []struct {
		name      string
		key       tea.KeyPressMsg
		preScroll bool // scroll down with PgDn before pressing key (creates room to scroll up)
		wantZero  bool // expect YOffset == 0 after the key (goto top)
		wantDown  bool // expect YOffset to increase after the key (scroll/bottom)
	}{
		{"j scrolls down one line", tea.KeyPressMsg{Code: 'j'}, false, false, true},
		{"arrow down scrolls down", tea.KeyPressMsg{Code: tea.KeyDown}, false, false, true},
		{"PgDn scrolls down a page", tea.KeyPressMsg{Code: tea.KeyPgDown}, false, false, true},
		{"G jumps to bottom", tea.KeyPressMsg{Code: 'G'}, false, false, true},
		{"k scrolls up one line", tea.KeyPressMsg{Code: 'k'}, true, false, false},
		{"PgUp scrolls up a page", tea.KeyPressMsg{Code: tea.KeyPgUp}, true, false, false},
		{"g jumps to top", tea.KeyPressMsg{Code: 'g'}, true, true, false},
	}

	for _, tt := range tests { //nolint:paralleltest // subtests share viewport/struct state
		t.Run(tt.name, func(t *testing.T) {
			m := sizedDryRunModel(80, 24, 60)

			if tt.preScroll {
				nm, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyPgDown})
				m = nm.(RestoreModel)
			}
			before := m.viewport.YOffset()
			if tt.preScroll && before == 0 {
				t.Fatalf("precondition: pre-scroll did not move YOffset, cannot assert %s", tt.name)
			}

			nm, _ := m.Update(tt.key)
			m = nm.(RestoreModel)
			after := m.viewport.YOffset()

			switch {
			case tt.wantZero:
				if after != 0 {
					t.Errorf("%s: YOffset = %d, want 0 (top)", tt.name, after)
				}
			case tt.wantDown:
				if after <= before {
					t.Errorf("%s: YOffset did not increase: %d -> %d", tt.name, before, after)
				}
			default: // up-direction: expect YOffset to decrease
				if after >= before {
					t.Errorf("%s: YOffset did not decrease: %d -> %d", tt.name, before, after)
				}
			}
		})
	}
}

// TestRestore_DryRun_QReturnsToList verifies 'q' returns to the backup list
// (REQ-TP-005 scenario "q returns to backup list").
func TestRestore_DryRun_QReturnsToList(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := sizedDryRunModel(80, 24, 30)

	nm, _ := m.Update(tea.KeyPressMsg{Code: 'q'})
	m = nm.(RestoreModel)

	if m.State != restoreStateList {
		t.Errorf("after 'q' in dry-run: state = %v, want restoreStateList", m.State)
	}
}
