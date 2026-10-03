package screens

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// TestRestorePicker_SelectsBackup verifies Enter sets SelectedID.
func TestRestorePicker_SelectsBackup(t *testing.T) { //nolint:paralleltest // pure model state transition test
	backups := []BackupInfo{
		{ID: "20260617-120000", Date: "2026-06-17", Size: "1.0 MB"},
		{ID: "20260618-090000", Date: "2026-06-18", Size: "2.5 MB"},
	}
	m := RestorePickerModel{Backups: backups, Cursor: 1}

	newM, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = newM.(RestorePickerModel)

	if !m.Confirmed {
		t.Error("Enter should set Confirmed=true")
	}
	if m.SelectedID() != "20260618-090000" {
		t.Errorf("SelectedID = %q, want %q", m.SelectedID(), "20260618-090000")
	}
}

// TestRestorePicker_EmptyList verifies empty backups list handled.
func TestRestorePicker_EmptyList(t *testing.T) { //nolint:paralleltest // pure model state transition test
	m := RestorePickerModel{Backups: nil}

	view := m.View()
	if !strings.Contains(view.Content, "No backups") {
		t.Error("empty list should show 'No backups' message")
	}

	newM, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = newM.(RestorePickerModel)
	if m.Confirmed {
		t.Error("Enter on empty list should not confirm")
	}
}

// TestRestorePicker_Cancel verifies q/Esc cancels.
func TestRestorePicker_Cancel(t *testing.T) { //nolint:paralleltest // pure model state transition test
	backups := []BackupInfo{
		{ID: "20260617-120000", Date: "2026-06-17", Size: "1.0 MB"},
	}
	m := RestorePickerModel{Backups: backups, Cursor: 0}
	newM, _ := m.Update(tea.KeyPressMsg{Code: 'q'})
	m = newM.(RestorePickerModel)

	if m.Confirmed {
		t.Error("q should not confirm selection")
	}
	if m.SelectedID() != "" {
		t.Errorf("SelectedID after q = %q, want empty", m.SelectedID())
	}
}

// TestRestorePicker_CursorBounds verifies cursor cannot go out of bounds.
func TestRestorePicker_CursorBounds(t *testing.T) { //nolint:paralleltest // pure model state transition test
	backups := []BackupInfo{
		{ID: "a", Date: "d1", Size: "1"},
		{ID: "b", Date: "d2", Size: "2"},
	}

	m := RestorePickerModel{Backups: backups, Cursor: 0}
	newM, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	m = newM.(RestorePickerModel)
	if m.Cursor != 0 {
		t.Errorf("cursor should stay 0, got %d", m.Cursor)
	}

	m2 := RestorePickerModel{Backups: backups, Cursor: 1}
	newM2, _ := m2.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m2 = newM2.(RestorePickerModel)
	if m2.Cursor != 1 {
		t.Errorf("cursor should stay 1, got %d", m2.Cursor)
	}
}

// TestRestorePicker_NarrowTerminal shows "too small" message.
func TestRestorePicker_NarrowTerminal(t *testing.T) { //nolint:paralleltest // pure model state transition test
	m := RestorePickerModel{Width: 19, Height: 9}
	view := m.View()

	if !strings.Contains(view.Content, "too small") {
		t.Error("narrow terminal should show 'too small' message")
	}
}

func TestRestorePickerModel_TableDriven_Update(t *testing.T) { //nolint:paralleltest // pure model state transition test
	sampleBackups := []BackupInfo{
		{ID: "20260101-100000", Date: "2026-01-01", Size: "1.2 MB"},
		{ID: "20260102-100000", Date: "2026-01-02", Size: "2.4 MB"},
		{ID: "20260103-100000", Date: "2026-01-03", Size: "3.6 MB"},
	}

	tests := []struct {
		name          string
		initial       RestorePickerModel
		msg           tea.Msg
		wantCursor    int
		wantConfirmed bool
		wantQuitting  bool
		wantCmdQuit   bool
		wantWidth     int
		wantHeight    int
	}{
		{
			name:       "window_size_msg_updates_dimensions",
			initial:    RestorePickerModel{Backups: sampleBackups, Cursor: 0},
			msg:        tea.WindowSizeMsg{Width: 80, Height: 24},
			wantCursor: 0,
			wantWidth:  80,
			wantHeight: 24,
		},
		{
			name:       "cursor_down_increments",
			initial:    RestorePickerModel{Backups: sampleBackups, Cursor: 0},
			msg:        tea.KeyPressMsg{Code: tea.KeyDown},
			wantCursor: 1,
		},
		{
			name:       "cursor_down_j_increments",
			initial:    RestorePickerModel{Backups: sampleBackups, Cursor: 1},
			msg:        tea.KeyPressMsg{Code: 'j'},
			wantCursor: 2,
		},
		{
			name:       "cursor_down_at_bottom_stays_clamped",
			initial:    RestorePickerModel{Backups: sampleBackups, Cursor: 2},
			msg:        tea.KeyPressMsg{Code: tea.KeyDown},
			wantCursor: 2,
		},
		{
			name:       "cursor_up_decrements",
			initial:    RestorePickerModel{Backups: sampleBackups, Cursor: 2},
			msg:        tea.KeyPressMsg{Code: tea.KeyUp},
			wantCursor: 1,
		},
		{
			name:       "cursor_up_k_decrements",
			initial:    RestorePickerModel{Backups: sampleBackups, Cursor: 1},
			msg:        tea.KeyPressMsg{Code: 'k'},
			wantCursor: 0,
		},
		{
			name:       "cursor_up_at_top_stays_clamped",
			initial:    RestorePickerModel{Backups: sampleBackups, Cursor: 0},
			msg:        tea.KeyPressMsg{Code: tea.KeyUp},
			wantCursor: 0,
		},
		{
			name:          "enter_with_items_confirms_and_quits",
			initial:       RestorePickerModel{Backups: sampleBackups, Cursor: 1},
			msg:           tea.KeyPressMsg{Code: tea.KeyEnter},
			wantCursor:    1,
			wantConfirmed: true,
			wantCmdQuit:   true,
		},
		{
			name:          "enter_with_empty_items_does_not_confirm",
			initial:       RestorePickerModel{Backups: []BackupInfo{}, Cursor: 0},
			msg:           tea.KeyPressMsg{Code: tea.KeyEnter},
			wantCursor:    0,
			wantConfirmed: false,
		},
		{
			name:          "enter_with_nil_items_does_not_confirm",
			initial:       RestorePickerModel{Backups: nil, Cursor: 0},
			msg:           tea.KeyPressMsg{Code: tea.KeyEnter},
			wantCursor:    0,
			wantConfirmed: false,
		},
		{
			name:         "quit_with_q",
			initial:      RestorePickerModel{Backups: sampleBackups, Cursor: 0},
			msg:          tea.KeyPressMsg{Code: 'q'},
			wantQuitting: true,
			wantCmdQuit:  true,
		},
		{
			name:         "quit_with_esc",
			initial:      RestorePickerModel{Backups: sampleBackups, Cursor: 0},
			msg:          tea.KeyPressMsg{Code: tea.KeyEsc},
			wantQuitting: true,
			wantCmdQuit:  true,
		},
		{
			name:         "quit_with_ctrl_c",
			initial:      RestorePickerModel{Backups: sampleBackups, Cursor: 0},
			msg:          tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl},
			wantQuitting: true,
			wantCmdQuit:  true,
		},
		{
			name:       "unhandled_key_ignored",
			initial:    RestorePickerModel{Backups: sampleBackups, Cursor: 1},
			msg:        tea.KeyPressMsg{Code: 'x'},
			wantCursor: 1,
		},
	}

	for _, tt := range tests { //nolint:paralleltest // subtests share table/struct state
		t.Run(tt.name, func(t *testing.T) {
			m := tt.initial
			if m.Init() != nil {
				t.Error("Init() should return nil")
			}

			updated, cmd := m.Update(tt.msg)
			gotModel, ok := updated.(RestorePickerModel)
			if !ok {
				t.Fatalf("Update() returned type %T, want RestorePickerModel", updated)
			}

			if gotModel.Cursor != tt.wantCursor {
				t.Errorf("Cursor = %d, want %d", gotModel.Cursor, tt.wantCursor)
			}
			if gotModel.Confirmed != tt.wantConfirmed {
				t.Errorf("Confirmed = %v, want %v", gotModel.Confirmed, tt.wantConfirmed)
			}
			if gotModel.Quitting != tt.wantQuitting {
				t.Errorf("Quitting = %v, want %v", gotModel.Quitting, tt.wantQuitting)
			}
			if tt.wantCmdQuit && cmd == nil {
				t.Error("expected tea.Quit command, got nil")
			}
			if !tt.wantCmdQuit && cmd != nil {
				t.Errorf("expected nil cmd, got %v", cmd)
			}
			if tt.wantWidth != 0 && gotModel.Width != tt.wantWidth {
				t.Errorf("Width = %d, want %d", gotModel.Width, tt.wantWidth)
			}
			if tt.wantHeight != 0 && gotModel.Height != tt.wantHeight {
				t.Errorf("Height = %d, want %d", gotModel.Height, tt.wantHeight)
			}
		})
	}
}

func TestRestorePickerModel_TableDriven_ViewAndSelection(t *testing.T) { //nolint:paralleltest // pure model state transition test
	sampleBackups := []BackupInfo{
		{ID: "20260101-100000", Date: "2026-01-01", Size: "1.2 MB"},
		{ID: "20260102-100000", Date: "2026-01-02", Size: "2.4 MB"},
	}

	tests := []struct {
		name           string
		model          RestorePickerModel
		wantViewSub    string
		wantViewEmpty  bool
		wantSelectedID string
	}{
		{
			name: "normal_view_shows_title_and_items",
			model: RestorePickerModel{
				Backups: sampleBackups,
				Cursor:  0,
				Width:   80,
				Height:  24,
			},
			wantViewSub:    "Select backup to restore",
			wantSelectedID: "", // unconfirmed
		},
		{
			name: "confirmed_selection_returns_id",
			model: RestorePickerModel{
				Backups:   sampleBackups,
				Cursor:    1,
				Confirmed: true,
				Width:     80,
				Height:    24,
			},
			wantViewSub:    "Select backup to restore",
			wantSelectedID: "20260102-100000",
		},
		{
			name: "quitting_returns_empty_view",
			model: RestorePickerModel{
				Backups:  sampleBackups,
				Cursor:   0,
				Quitting: true,
			},
			wantViewEmpty:  true,
			wantSelectedID: "",
		},
		{
			name: "narrow_terminal_width_shows_too_small",
			model: RestorePickerModel{
				Backups: sampleBackups,
				Width:   15,
				Height:  20,
			},
			wantViewSub:    "Terminal too small",
			wantSelectedID: "",
		},
		{
			name: "narrow_terminal_height_shows_too_small",
			model: RestorePickerModel{
				Backups: sampleBackups,
				Width:   40,
				Height:  8,
			},
			wantViewSub:    "Terminal too small",
			wantSelectedID: "",
		},
		{
			name: "empty_slice_shows_no_backups",
			model: RestorePickerModel{
				Backups: []BackupInfo{},
				Width:   80,
				Height:  24,
			},
			wantViewSub:    "No backups found.",
			wantSelectedID: "",
		},
		{
			name: "nil_slice_shows_no_backups",
			model: RestorePickerModel{
				Backups: nil,
				Width:   80,
				Height:  24,
			},
			wantViewSub:    "No backups found.",
			wantSelectedID: "",
		},
		{
			name: "negative_cursor_unconfirmed_renders_without_panic",
			model: RestorePickerModel{
				Backups: sampleBackups,
				Cursor:  -1,
				Width:   80,
				Height:  24,
			},
			wantViewSub:    "Select backup to restore",
			wantSelectedID: "",
		},
		{
			name: "out_of_bounds_cursor_renders_and_returns_empty_id",
			model: RestorePickerModel{
				Backups:   sampleBackups,
				Cursor:    99,
				Confirmed: true,
				Width:     80,
				Height:    24,
			},
			wantViewSub:    "Select backup to restore",
			wantSelectedID: "",
		},
	}

	for _, tt := range tests { //nolint:paralleltest // subtests share table/struct state
		t.Run(tt.name, func(t *testing.T) {
			view := tt.model.View()
			if tt.wantViewEmpty {
				if view.Content != "" {
					t.Errorf("View().Content = %q, want empty", view.Content)
				}
			} else if tt.wantViewSub != "" {
				if !strings.Contains(view.Content, tt.wantViewSub) {
					t.Errorf("View().Content %q does not contain %q", view.Content, tt.wantViewSub)
				}
			}

			if gotID := tt.model.SelectedID(); gotID != tt.wantSelectedID {
				t.Errorf("SelectedID() = %q, want %q", gotID, tt.wantSelectedID)
			}
		})
	}
}
