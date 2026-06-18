package cmd

import (
	"os"

	"github.com/mattn/go-isatty"
)

// isTTY reports whether stdin is a terminal.
// Exposed as a package-level variable so tests can override it
// (follows the var execCommand pattern from AGENTS.md).
var isTTY = func() bool {
	return isatty.IsTerminal(os.Stdin.Fd())
}
