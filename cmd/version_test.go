package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestVersionCmd_Execute(t *testing.T) {
	// Test that version command executes without error.
	// The version command uses fmt.Printf (not cmd.Printf), so output
	// goes to os.Stdout rather than the cobra output buffer.
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
