package cmd

import (
	"errors"
	"testing"

	"github.com/danielxxomg/bak-cli/internal/tui"
)

// =============================================================================
// TestIsTTY — RED (tty.go does not exist yet)
// =============================================================================

// TestIsTTY_ReturnsFalseInTestEnv verifies the real isTTY (which reads
// os.Stdin) returns false in piped test environments. We also verify the
// injection point works by overriding it.
func TestIsTTY_ReturnsFalseInTestEnv(t *testing.T) {
	// The real isTTY should return false because test runners pipe stdin.
	if isTTY() {
		t.Error("isTTY() = true in test environment, want false (stdin is piped)")
	}
}

// TestIsTTY_OverridePointWorks verifies the injection variable can be
// overridden for testing, following the AGENTS.md pattern.
func TestIsTTY_OverridePointWorks(t *testing.T) {
	orig := isTTY
	defer func() { isTTY = orig }()

	// Override to return true.
	isTTY = func() bool { return true }
	if !isTTY() {
		t.Error("isTTY override to true did not take effect")
	}

	// Override to return false.
	isTTY = func() bool { return false }
	if isTTY() {
		t.Error("isTTY override to false did not take effect")
	}
}

// =============================================================================
// TestRunTUI_InjectionPoint — RED
// =============================================================================

func TestRunTUI_InjectionPoint(t *testing.T) {
	// Save the original and restore after the test.
	orig := runTUI
	defer func() { runTUI = orig }()

	called := false
	var receivedDeps tui.Deps

	runTUI = func(deps tui.Deps) error {
		called = true
		receivedDeps = deps
		return nil
	}

	deps := tui.Deps{
		Version:      "test-version",
		ConfigExists: func() bool { return true },
	}

	err := runTUI(deps)

	if err != nil {
		t.Errorf("runTUI returned unexpected error: %v", err)
	}
	if !called {
		t.Error("runTUI was not called")
	}
	if receivedDeps.Version != "test-version" {
		t.Errorf("receivedDeps.Version = %q, want %q", receivedDeps.Version, "test-version")
	}
}

// TestRunTUI_PropagatesError verifies errors from runTUI are propagated.
func TestRunTUI_PropagatesError(t *testing.T) {
	orig := runTUI
	defer func() { runTUI = orig }()

	wantErr := errors.New("TUI failed")
	runTUI = func(deps tui.Deps) error {
		return wantErr
	}

	err := runTUI(tui.Deps{Version: "1.0.0"})
	if err == nil {
		t.Fatal("runTUI should propagate errors")
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("runTUI error = %v, want %v", err, wantErr)
	}
}

// =============================================================================
// TestRunTUI_Initialized — RED
// =============================================================================

func TestRunTUI_Initialized(t *testing.T) {
	// runTUI must be initialized to defaultRunTUI at package init.
	if runTUI == nil {
		t.Error("runTUI is nil, should be initialized to defaultRunTUI")
	}
}
