//go:build windows

package schedule

import (
	"os/exec"
	"strings"
	"testing"
)

func TestSchtasksScheduler_Create_NoAdmin(t *testing.T) {
	origIsAdmin := isAdminFn
	defer func() { isAdminFn = origIsAdmin }()

	// Simulate non-admin environment.
	isAdminFn = func() bool { return false }

	s := &SchtasksScheduler{}
	err := s.Create("work", "daily")
	if err == nil {
		t.Fatal("expected error when not admin")
	}
	if !strings.Contains(err.Error(), "administrator") {
		t.Errorf("error should mention administrator, got: %v", err)
	}
}

func TestSchtasksScheduler_Create_Admin_ProceedsToSchtasks(t *testing.T) {
	origIsAdmin := isAdminFn
	origExec := execCommand
	defer func() {
		isAdminFn = origIsAdmin
		execCommand = origExec
	}()

	// Simulate admin environment.
	isAdminFn = func() bool { return true }

	// Mock execCommand to capture that schtasks was invoked.
	called := false
	execCommand = func(name string, arg ...string) *exec.Cmd {
		if name == "schtasks" {
			called = true
		}
		// Return a command that will succeed.
		// Use "cmd /c exit 0" as a Windows no-op.
		return exec.Command("cmd", "/c", "exit", "0")
	}

	s := &SchtasksScheduler{}
	err := s.Create("work", "daily")
	if err != nil {
		t.Logf("Create error (expected from mock environment): %v", err)
	}
	if !called {
		t.Error("schtasks was never invoked after admin check passed")
	}
}

func TestIsAdmin_ReturnsBool(t *testing.T) {
	result := isAdmin()
	// Just verify it doesn't panic and returns a boolean.
	_ = result
}

func TestSchtasksScheduler_Remove_NoAdminRequired(t *testing.T) {
	// Remove does not check admin; verify it proceeds.
	origExec := execCommand
	defer func() { execCommand = origExec }()

	called := false
	execCommand = func(name string, arg ...string) *exec.Cmd {
		if name == "schtasks" {
			called = true
		}
		return exec.Command("cmd", "/c", "exit", "0")
	}

	s := &SchtasksScheduler{}
	err := s.Remove("work")
	if err != nil {
		t.Logf("Remove error (expected from mock): %v", err)
	}
	if !called {
		t.Error("schtasks delete was not invoked")
	}
}
