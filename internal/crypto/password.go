package crypto

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// envPasswordKey is the environment variable read for the encryption password.
const envPasswordKey = "BAK_ENCRYPTION_PASSWORD"

// GetPassword returns the encryption password.
// Priority: BAK_ENCRYPTION_PASSWORD env var, then interactive stdin prompt.
// Returns an error if stdin is not available and the env var is not set.
func GetPassword(prompt string) (string, error) {
	if pass, ok := resolveFromEnv(); ok {
		return pass, nil
	}
	return promptPassword(prompt)
}

// resolveFromEnv checks BAK_ENCRYPTION_PASSWORD. Returns (password, true) if set.
func resolveFromEnv() (string, bool) {
	pass, ok := os.LookupEnv(envPasswordKey)
	return pass, ok
}

// promptPassword reads a password from stdin.
func promptPassword(prompt string) (string, error) {
	// Write prompt to stderr so it is visible even when stdout is redirected.
	fmt.Fprint(os.Stderr, prompt)

	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		if err == io.EOF {
			return "", fmt.Errorf("no password provided: stdin is not a terminal and %s is not set", envPasswordKey)
		}
		return "", fmt.Errorf("read password: %w", err)
	}

	return strings.TrimRight(line, "\r\n"), nil
}
