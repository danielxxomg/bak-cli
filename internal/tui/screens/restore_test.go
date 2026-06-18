// Package screens provides screen-specific render functions and sub-models
// for the bak-cli TUI. This file contains strict-TDD tests for RestoreModel
// written BEFORE the production code.
package screens

import (
	"errors"
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

func TestRestore_NewRestoreModel(t *testing.T) {
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

func TestRestore_Init_PopulatesBackups(t *testing.T) {
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

func TestRestore_EmptyState(t *testing.T) {
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

// =============================================================================
// TestRestore_StateTransition_ListToDryRun — RED
// =============================================================================

func TestRestore_StateTransition_ListToDryRun(t *testing.T) {
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

func TestRestore_StateTransition_DryRunToConfirm(t *testing.T) {
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

func TestRestore_ConfirmExecute(t *testing.T) {
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

func TestRestore_ConfirmCancel(t *testing.T) {
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

func TestRestore_ErrorHandling(t *testing.T) {
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

func TestRestore_View_DryRun(t *testing.T) {
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

func TestRestore_View_Confirm(t *testing.T) {
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

// =============================================================================
// Phase 3: Render helper coverage for restore.go (0% before backfill)
// =============================================================================

func TestRestore_RenderErrorState(t *testing.T) {
	m := NewRestoreModel(nil, nil)
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

func TestRestore_RenderBackupList(t *testing.T) {
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

func TestRestore_RenderBackupList_Empty(t *testing.T) {
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

func TestRestore_RenderRunning(t *testing.T) {
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

func TestRestore_RenderDone(t *testing.T) {
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

func TestRestore_RenderDone_Error(t *testing.T) {
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
