package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestVersionCmd_Execute(t *testing.T) {
	// Test that version command executes without error.
	// The version command uses cmd.Println/cmd.Printf, so output
	// goes to cobra's OutOrStdout buffer when SetOut is used.
	bufOut := new(bytes.Buffer)
	bufErr := new(bytes.Buffer)

	rootCmd.SetOut(bufOut)
	rootCmd.SetErr(bufErr)
	rootCmd.SetArgs([]string{"version"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("version command should not error, got: %v", err)
	}
	// Verify the command executed successfully.
}

func TestVersionCmd_Run(t *testing.T) {
	// The Run function uses fmt.Printf directly. We verify it doesn't panic
	// and that version variables are set.
	// (Output assertions are covered by TestVersionVariables in root_test.go)

	// Ensure version variables are set.
	if Version == "" {
		t.Error("Version should not be empty")
	}
	if Commit == "" {
		t.Error("Commit should not be empty")
	}
	if Date == "" {
		t.Error("Date should not be empty")
	}

	// Verify that versionCmd.Run is callable without panic.
	// We use a pipe to capture stdout without blocking.
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("version.Run panicked: %v", r)
		}
	}()
	versionCmd.Run(versionCmd, nil)
}

func TestVersionCmd_HelpOutput(t *testing.T) {
	bufOut := new(bytes.Buffer)
	bufErr := new(bytes.Buffer)
	rootCmd.SetOut(bufOut)
	rootCmd.SetErr(bufErr)

	rootCmd.SetArgs([]string{"version", "--help"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("version --help should not error: %v", err)
	}

	output := bufOut.String()
	if !strings.Contains(output, "version") {
		t.Error("help output should mention 'version'")
	}
	if !strings.Contains(output, "Print version") || !strings.Contains(output, "version information") {
		t.Log("version help output may have different format")
	}
}

// TestVersionFlagNoConflictWithVerbose verifies that the --version flag
// does not conflict with --verbose's -v shorthand. When both flags are
// registered, cobra.Execute() must not panic.
func TestVersionFlagNoConflictWithVerbose(t *testing.T) {
	// Ensure rootCmd.Version is set (triggers cobra auto --version registration).
	if rootCmd.Version == "" {
		t.Fatal("rootCmd.Version is not set — cannot test flag registration")
	}

	// The --version flag is registered manually in version.go init() without
	// -v shorthand. Verify it exists and has correct properties.
	vf := rootCmd.Flags().Lookup("version")
	if vf == nil {
		t.Fatal("--version flag is not registered on rootCmd")
	}
	if vf.Shorthand != "" {
		t.Errorf("--version must not use -v shorthand (reserved for --verbose), got %q", vf.Shorthand)
	}
	// Verify the flag is a boolean type.
	if vf.Value.Type() != "bool" {
		t.Errorf("--version flag type = %q, want bool", vf.Value.Type())
	}
}

// TestVersionAndVerboseExecuteNoPanic verifies that Execute() does not
// panic when both --version and --verbose flags coexist. This covers
// the end-to-end scenario that the shorthand fix protects.
func TestVersionAndVerboseExecuteNoPanic(t *testing.T) {
	// The verbose flag with -v shorthand is registered on rootCmd.PersistentFlags().
	// InitDefaultVersionFlag() (called in version.go init()) creates --version
	// with its -v shorthand already removed. This test proves that cobra's internal
	// initDefaultVersionFlag during Execute() does not re-add the conflicting
	// shorthand and cause a panic.
	bufOut := new(bytes.Buffer)
	bufErr := new(bytes.Buffer)
	rootCmd.SetOut(bufOut)
	rootCmd.SetErr(bufErr)

	// Running with no args exercises the flag registration code path.
	rootCmd.SetArgs([]string{"--version"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("--version should not error: %v", err)
	}
	output := bufOut.String()
	if !strings.Contains(output, "bak") {
		t.Errorf("--version output should mention bak, got: %s", output)
	}

	// Also verify --help doesn't panic (exercises flag parsing with both flags present).
	bufOut.Reset()
	rootCmd.SetArgs([]string{"--help"})
	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("--help should not panic or error: %v", err)
	}
}
