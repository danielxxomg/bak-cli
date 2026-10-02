package crypto

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

// envPasswordKey is the environment variable read for the encryption password.
const envPasswordKey = "BAK_ENCRYPTION_PASSWORD"

// Injected functions for terminal detection and masked password reading.
// Default to golang.org/x/term implementations.
var (
	isTerminalFn   = term.IsTerminal
	readPasswordFn = term.ReadPassword
)

// GetPassword returns the encryption password.
// Priority: BAK_ENCRYPTION_PASSWORD env var, then interactive terminal prompt.
// Empty passwords (whether from environment variable or interactive prompt) are rejected.
// Returns an error if stdin is not a terminal and the env var is not set.
func GetPassword(prompt string) (string, error) {
	if pass, ok := resolveFromEnv(); ok {
		if pass == "" {
			return "", fmt.Errorf("read password: %s is set but empty; set a non-empty password", envPasswordKey)
		}
		return pass, nil
	}
	return promptPassword(prompt)
}

// resolveFromEnv checks BAK_ENCRYPTION_PASSWORD. Returns (password, true) if set.
func resolveFromEnv() (string, bool) {
	pass, ok := os.LookupEnv(envPasswordKey)
	return pass, ok
}

// promptPassword reads a password from stdin using masked input without echo.
// Stdin must be an interactive terminal.
func promptPassword(prompt string) (string, error) {
	fd := int(os.Stdin.Fd())
	if !isTerminalFn(fd) {
		return "", fmt.Errorf("read password: stdin is not a terminal and %s is not set", envPasswordKey)
	}

	// Write prompt to stderr so it is visible even when stdout is redirected.
	fmt.Fprint(os.Stderr, prompt)

	passBytes, err := readPasswordFn(fd)
	if err != nil {
		return "", fmt.Errorf("read password: %w", err)
	}

	// Move cursor to the next line on stderr since term.ReadPassword disables echo of newline.
	fmt.Fprintln(os.Stderr)

	pass := strings.TrimRight(string(passBytes), "\r\n")
	if pass == "" {
		return "", fmt.Errorf("read password: empty password entered; provide a password or set %s", envPasswordKey)
	}

	return pass, nil
}
