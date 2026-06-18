// Package tui provides the root TUI model and action dispatch for the
// bak-cli interactive TUI.
package tui

import (
	"errors"
	"testing"
)

// TestRouteSelection verifies the RouteSelection pure function routes
// menu cursor positions to the appropriate action calls in Deps.
// Table-driven per AGENTS.md and go-testing skill.
func TestRouteSelection(t *testing.T) {
	tests := []struct {
		name string
		sel  MenuSelection
		// runBackup is the spy for Deps.RunBackup. When nil, the test
		// installs a guard that fails if RunBackup is called unexpectedly.
		runBackup func(cats []string, ch chan<- ProgressUpdate) error
		wantErr   bool
		wantCall  bool // true when RunBackup should be called
	}{
		{
			name:      "cursor 0 Selected=true calls RunBackup",
			sel:       MenuSelection{Selected: true, Cursor: 0, Item: "Create backup"},
			runBackup: func(cats []string, ch chan<- ProgressUpdate) error { return nil },
			wantCall:  true,
		},
		{
			name: "cursor 0 Selected=false does NOT call RunBackup (q gate)",
			sel:  MenuSelection{Selected: false, Cursor: 0, Item: "Create backup"},
			runBackup: func(cats []string, ch chan<- ProgressUpdate) error {
				t.Error("RunBackup called unexpectedly")
				return nil
			},
			wantCall: false,
		},
		{
			name: "cursor 1 Selected=true Restore no-op",
			sel:  MenuSelection{Selected: true, Cursor: 1, Item: "Restore"},
		},
		{
			name: "cursor 6 Selected=true Quit no-op",
			sel:  MenuSelection{Selected: true, Cursor: 6, Item: "Quit"},
		},
		{
			name: "empty selection no-op",
			sel:  MenuSelection{},
		},
		{
			name:      "cursor 0 propagates RunBackup error",
			sel:       MenuSelection{Selected: true, Cursor: 0, Item: "Create backup"},
			runBackup: func(cats []string, ch chan<- ProgressUpdate) error { return errors.New("backup failed") },
			wantErr:   true,
			wantCall:  true,
		},
		{
			name: "cursor 0 Selected=false non-zero cursor no-op",
			sel:  MenuSelection{Selected: false, Cursor: 2, Item: "Browse backups"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var called bool
			deps := Deps{}

			if tt.runBackup != nil {
				// Install spy that records invocation and delegates.
				orig := tt.runBackup
				deps.RunBackup = func(cats []string, ch chan<- ProgressUpdate) error {
					called = true
					return orig(cats, ch)
				}
			} else if tt.sel.Cursor == 0 {
				// Guard: RunBackup should NOT be called for these cases.
				deps.RunBackup = func(cats []string, ch chan<- ProgressUpdate) error {
					t.Error("RunBackup was called unexpectedly")
					return nil
				}
			}

			err := RouteSelection(tt.sel, deps)

			if (err != nil) != tt.wantErr {
				t.Errorf("RouteSelection() error = %v, wantErr = %v", err, tt.wantErr)
			}

			if tt.wantCall && !called {
				t.Error("RunBackup was not called when expected")
			}
			if !tt.wantCall && called {
				t.Error("RunBackup was called unexpectedly")
			}
		})
	}
}
