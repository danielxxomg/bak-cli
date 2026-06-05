package crypto

import (
	"os"
	"strings"
	"testing"
)

func TestGetPassword_EnvVar(t *testing.T) {
	os.Setenv("BAK_ENCRYPTION_PASSWORD", "env-secret")
	defer os.Unsetenv("BAK_ENCRYPTION_PASSWORD")

	password, err := GetPassword("Enter password: ")
	if err != nil {
		t.Fatalf("GetPassword with env var: %v", err)
	}
	if password != "env-secret" {
		t.Errorf("password = %q, want %q", password, "env-secret")
	}
}

func TestGetPassword_EnvVar_EmptyString(t *testing.T) {
	os.Setenv("BAK_ENCRYPTION_PASSWORD", "")
	defer os.Unsetenv("BAK_ENCRYPTION_PASSWORD")

	password, err := GetPassword("Enter password: ")
	if err != nil {
		t.Fatalf("GetPassword with empty env var: %v", err)
	}
	if password != "" {
		t.Errorf("password = %q, want empty", password)
	}
}

func TestGetPassword_StdinPipe(t *testing.T) {
	// Unset env var so stdin is used.
	os.Unsetenv("BAK_ENCRYPTION_PASSWORD")

	// Simulate stdin via a pipe by setting os.Stdin temporarily.
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	input := "stdin-secret\n"
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	os.Stdin = r

	// Write the password and close the write end.
	go func() {
		w.Write([]byte(input))
		w.Close()
	}()

	password, err := GetPassword("Prompt: ")
	if err != nil {
		t.Fatalf("GetPassword with stdin: %v", err)
	}
	if password != "stdin-secret" {
		t.Errorf("password = %q, want %q", password, "stdin-secret")
	}
}

func TestGetPassword_Stdin_TrailingNewline(t *testing.T) {
	os.Unsetenv("BAK_ENCRYPTION_PASSWORD")

	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	input := "password-with-newline\n\n"
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	os.Stdin = r

	go func() {
		w.Write([]byte(input))
		w.Close()
	}()

	password, err := GetPassword("Prompt: ")
	if err != nil {
		t.Fatalf("GetPassword with trailing newline: %v", err)
	}
	// Should trim trailing whitespace/newlines.
	password = strings.TrimSpace(password)
	if password != "password-with-newline" {
		t.Errorf("password = %q, want %q", password, "password-with-newline")
	}
}

func TestGetPassword_NoTerminal_NoEnvVar(t *testing.T) {
	os.Unsetenv("BAK_ENCRYPTION_PASSWORD")

	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	// Close stdin to simulate non-terminal (pipe with EOF).
	r, w, _ := os.Pipe()
	w.Close() // close immediately to produce EOF
	os.Stdin = r

	_, err := GetPassword("Prompt: ")
	if err == nil {
		t.Fatal("GetPassword: expected error when stdin is closed and no env var, got nil")
	}
}

func TestGetPassword_EnvVar_SpecialChars(t *testing.T) {
	os.Setenv("BAK_ENCRYPTION_PASSWORD", "p@$$w0rd!%^&*()")
	defer os.Unsetenv("BAK_ENCRYPTION_PASSWORD")

	password, err := GetPassword("Enter: ")
	if err != nil {
		t.Fatalf("GetPassword with special chars: %v", err)
	}
	if password != "p@$$w0rd!%^&*()" {
		t.Errorf("password = %q, want %q", password, "p@$$w0rd!%^&*()")
	}
}

func TestResolveFromEnv_NotSet(t *testing.T) {
	os.Unsetenv("BAK_ENCRYPTION_PASSWORD")

	_, ok := resolveFromEnv()
	if ok {
		t.Error("resolveFromEnv returned true when env var is not set")
	}
}
