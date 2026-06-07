package cmd

import (
	"io"
	"os"

	"github.com/danielxxomg/bak-cli/internal/config"
	"github.com/spf13/cobra"
)

// cmdDeps holds injectable dependencies for command execution.
// Tests override these via runXWithDeps to isolate from the real filesystem.
type cmdDeps struct {
	ConfigLoader func() (*config.Config, error)
	Stdout       io.Writer
	Stderr       io.Writer
	Stdin        io.Reader
}

// defaultDeps provides production defaults using real OS handles
// and the config.Load function from internal/config.
var defaultDeps = cmdDeps{
	ConfigLoader: config.Load,
	Stdout:       os.Stdout,
	Stderr:       os.Stderr,
	Stdin:        os.Stdin,
}

// depsFromCmd builds cmdDeps from a cobra command, preserving
// cobra's testability via SetOut/SetErr/SetIn.
// If cmd is nil, returns defaultDeps.
func depsFromCmd(cmd *cobra.Command) cmdDeps {
	if cmd == nil {
		return defaultDeps
	}
	return cmdDeps{
		ConfigLoader: config.Load,
		Stdout:       cmd.OutOrStdout(),
		Stderr:       cmd.ErrOrStderr(),
		Stdin:        cmd.InOrStdin(),
	}
}
