package backup

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"
)

// TestResolveHostname_InjectedFnReturns verifies the injected hostname
// function's value is returned verbatim on success.
func TestResolveHostname_InjectedFnReturns(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	got := ResolveHostname(func() (string, error) { return "myhost", nil }, false, nil)
	if got != "myhost" {
		t.Errorf("ResolveHostname = %q, want %q", got, "myhost")
	}
}

// TestResolveHostname_InjectedFnTriangulation forces real logic with a
// different value (guards against a hardcoded Fake It).
func TestResolveHostname_InjectedFnTriangulation(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	got := ResolveHostname(func() (string, error) { return "build-server-02", nil }, false, nil)
	if got != "build-server-02" {
		t.Errorf("ResolveHostname = %q, want %q", got, "build-server-02")
	}
}

// TestResolveHostname_NilFallsBackToOsHostname verifies a nil fn delegates
// to os.Hostname and returns the same value os.Hostname reports.
func TestResolveHostname_NilFallsBackToOsHostname(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	got := ResolveHostname(nil, false, nil)
	want, err := os.Hostname()
	if err != nil {
		t.Skipf("os.Hostname unavailable on this host: %v", err)
	}
	if got != want {
		t.Errorf("ResolveHostname(nil) = %q, want os.Hostname() = %q", got, want)
	}
}

// TestResolveHostname_FnErrorDefaultsToUnknown_VerboseWarns verifies that
// when the injected fn returns an error, the result is "unknown" and a
// warning is written to errOut when verbose is true.
func TestResolveHostname_FnErrorDefaultsToUnknown_VerboseWarns(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	var buf bytes.Buffer
	got := ResolveHostname(func() (string, error) { return "", errors.New("lookup failed") }, true, &buf)
	if got != "unknown" {
		t.Errorf("ResolveHostname on fn error = %q, want %q", got, "unknown")
	}
	if !strings.Contains(buf.String(), "hostname") {
		t.Errorf("expected hostname warning in stderr, got %q", buf.String())
	}
}

// TestResolveHostname_FnErrorDefaultsToUnknown_QuietNoWarn verifies no
// warning is emitted when verbose is false.
func TestResolveHostname_FnErrorDefaultsToUnknown_QuietNoWarn(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	var buf bytes.Buffer
	got := ResolveHostname(func() (string, error) { return "", errors.New("lookup failed") }, false, &buf)
	if got != "unknown" {
		t.Errorf("ResolveHostname on fn error (quiet) = %q, want %q", got, "unknown")
	}
	if buf.String() != "" {
		t.Errorf("expected no warning when verbose=false, got %q", buf.String())
	}
}

// TestResolveHostname_NilAndOsHostnameFails_DefaultsUnknown verifies the
// spec scenario: fn is nil AND os.Hostname fails → "unknown" + verbose warn.
func TestResolveHostname_NilAndOsHostnameFails_DefaultsUnknown(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	orig := osHostname
	osHostname = func() (string, error) { return "", errors.New("unavailable") }
	defer func() { osHostname = orig }()

	var buf bytes.Buffer
	got := ResolveHostname(nil, true, &buf)
	if got != "unknown" {
		t.Errorf("ResolveHostname(nil, os.Hostname fails) = %q, want %q", got, "unknown")
	}
	if !strings.Contains(buf.String(), "hostname") {
		t.Errorf("expected hostname warning in stderr, got %q", buf.String())
	}
}
