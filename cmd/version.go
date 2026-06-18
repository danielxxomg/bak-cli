package cmd

import (
	"runtime"

	"github.com/spf13/cobra"
)

var (
	// Version is set at build time via ldflags. Defaults to "dev".
	Version = "dev"
	// Commit is the git commit hash at build time. Defaults to "unknown".
	Commit = "unknown"
	// Date is the build timestamp. Defaults to "unknown".
	Date = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long:  "Display the current version, commit hash, build date, and Go runtime details.",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Println("bak " + Version)
		cmd.Printf("  commit:  %s\n", Commit)
		cmd.Printf("  built:   %s\n", Date)
		cmd.Printf("  runtime: %s %s/%s\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)
	},
}

func init() {
	rootCmd.Version = Version
	rootCmd.SetVersionTemplate("bak {{.Version}}\n")
	// Register --version flag manually without -v shorthand to avoid
	// conflict with --verbose which uses -v. Cobra's initDefaultVersionFlag
	// would auto-register with -v if the flag doesn't exist yet.
	if rootCmd.Flags().Lookup("version") == nil {
		rootCmd.Flags().Bool("version", false, "version for bak")
	}
	rootCmd.AddCommand(versionCmd)
}
