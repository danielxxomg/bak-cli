package cmd

import (
	"testing"
)

// TestIsTTY_NotTerminal verifies the isTTY function exists and returns
// a boolean without panicking. In test environments, os.Stdin is typically
// not a terminal so the exact return value is environment-dependent.
func TestIsTTY_NotTerminal(t *testing.T) {
	result := isTTY()
	// Just check it doesn't panic.
	_ = result
}
