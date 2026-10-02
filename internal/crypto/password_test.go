package crypto

import (
	"errors"
	"os"
	"strings"
	"testing"
)

func TestGetPassword_EnvVar(t *testing.T) { //nolint:paralleltest // mutates environment and global hooks
	tests := []struct {
		name        string
		envValue    string
		wantPass    string
		wantErr     bool
		errContains string
	}{
		{
			name:     "valid password",
			envValue: "env-secret",
			wantPass: "env-secret",
			wantErr:  false,
		},
		{
			name:     "special characters",
			envValue: "p@$$w0rd!%^&*()",
			wantPass: "p@$$w0rd!%^&*()",
			wantErr:  false,
		},
		{
			name:        "empty env var rejected",
			envValue:    "",
			wantPass:    "",
			wantErr:     true,
			errContains: "BAK_ENCRYPTION_PASSWORD",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("BAK_ENCRYPTION_PASSWORD", tt.envValue)

			password, err := GetPassword("Enter password: ")
			if (err != nil) != tt.wantErr {
				t.Fatalf("GetPassword() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("GetPassword() error %q does not contain %q", err.Error(), tt.errContains)
				}
				return
			}
			if password != tt.wantPass {
				t.Errorf("GetPassword() = %q, want %q", password, tt.wantPass)
			}
		})
	}
}

func TestGetPassword_EnvVar_EmptyString(t *testing.T) { //nolint:paralleltest // mutates environment
	t.Setenv("BAK_ENCRYPTION_PASSWORD", "")

	_, err := GetPassword("Enter password: ")
	if err == nil {
		t.Fatal("GetPassword with empty env var: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "BAK_ENCRYPTION_PASSWORD") {
		t.Errorf("expected error to mention BAK_ENCRYPTION_PASSWORD, got %v", err)
	}
}

func TestGetPassword_Terminal(t *testing.T) { //nolint:paralleltest,tparallel // mutates package-level isTerminalFn/readPasswordFn hooks; subtests cannot run parallel
	// Ensure env var is unset so GetPassword falls through to terminal prompt.
	if err := os.Unsetenv("BAK_ENCRYPTION_PASSWORD"); err != nil {
		t.Fatal(err)
	}

	oldIsTerm := isTerminalFn
	oldReadPass := readPasswordFn
	t.Cleanup(func() {
		isTerminalFn = oldIsTerm
		readPasswordFn = oldReadPass
	})

	mockErr := errors.New("simulated terminal read error")

	tests := []struct {
		name             string
		isTerm           bool
		readPassBytes    []byte
		readPassErr      error
		wantPass         string
		wantErr          bool
		errContains      string
		wantMaskedCalled bool
	}{
		{
			name:             "valid masked terminal input",
			isTerm:           true,
			readPassBytes:    []byte("masked-secret"),
			wantPass:         "masked-secret",
			wantErr:          false,
			wantMaskedCalled: true,
		},
		{
			name:             "masked terminal input with trailing newline",
			isTerm:           true,
			readPassBytes:    []byte("masked-secret-newline\r\n"),
			wantPass:         "masked-secret-newline",
			wantErr:          false,
			wantMaskedCalled: true,
		},
		{
			name:             "empty terminal input rejected",
			isTerm:           true,
			readPassBytes:    []byte(""),
			wantErr:          true,
			errContains:      "BAK_ENCRYPTION_PASSWORD",
			wantMaskedCalled: true,
		},
		{
			name:             "empty terminal input with only newlines rejected",
			isTerm:           true,
			readPassBytes:    []byte("\r\n"),
			wantErr:          true,
			errContains:      "BAK_ENCRYPTION_PASSWORD",
			wantMaskedCalled: true,
		},
		{
			name:             "masked read error propagated",
			isTerm:           true,
			readPassErr:      mockErr,
			wantErr:          true,
			errContains:      "simulated terminal read error",
			wantMaskedCalled: true,
		},
		{
			name:             "stdin not a terminal returns error without reading password",
			isTerm:           false,
			wantErr:          true,
			errContains:      "stdin is not a terminal",
			wantMaskedCalled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if err := os.Unsetenv("BAK_ENCRYPTION_PASSWORD"); err != nil {
				t.Fatal(err)
			}

			maskedCalled := false
			isTerminalFn = func(fd int) bool {
				return tt.isTerm
			}
			readPasswordFn = func(fd int) ([]byte, error) {
				maskedCalled = true
				if tt.readPassErr != nil {
					return nil, tt.readPassErr
				}
				return tt.readPassBytes, nil
			}

			password, err := GetPassword("Enter password: ")
			if (err != nil) != tt.wantErr {
				t.Fatalf("GetPassword() error = %v, wantErr %v", err, tt.wantErr)
			}
			if maskedCalled != tt.wantMaskedCalled {
				t.Errorf("readPasswordFn called = %v, want %v", maskedCalled, tt.wantMaskedCalled)
			}
			if tt.wantErr {
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("GetPassword() error %q does not contain %q", err.Error(), tt.errContains)
				}
				return
			}
			if password != tt.wantPass {
				t.Errorf("GetPassword() = %q, want %q", password, tt.wantPass)
			}
		})
	}
}

func TestResolveFromEnv_NotSet(t *testing.T) { //nolint:paralleltest // mutates environment
	if err := os.Unsetenv("BAK_ENCRYPTION_PASSWORD"); err != nil {
		t.Fatal(err)
	}

	_, ok := resolveFromEnv()
	if ok {
		t.Error("resolveFromEnv returned true when env var is not set")
	}
}
