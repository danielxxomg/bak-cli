package adapters

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// failingWriter is an io.Writer that always returns an error on Write.
type failingWriter struct{}

func (failingWriter) Write(p []byte) (int, error) {
	return 0, errSimulatedStderr
}

var errSimulatedStderr = &simpleErr{"simulated stderr write failure"}

type simpleErr struct{ msg string }

func (e *simpleErr) Error() string { return e.msg }

// captureStderrInternal redirects os.Stderr for the duration of fn and returns
// whatever was written. Restores the original stderr afterward. (Internal-test
// copy of the helpers in generic_test.go, which lives in package adapters_test
// and cannot be imported here.)
func captureStderrInternal(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stderr = w
	defer func() { os.Stderr = old }()

	fn()

	if cerr := w.Close(); cerr != nil {
		t.Fatalf("close pipe writer: %v", cerr)
	}
	var buf bytes.Buffer
	if _, cerr := io.Copy(&buf, r); cerr != nil {
		t.Fatalf("read stderr pipe: %v", cerr)
	}
	return buf.String()
}

// TestEmitOversizeWarning_StderrWriteFailureContinuesScan covers the spec
// scenario "stderr write failure does not abort scan" (W5): when the writer
// behind emitOversizeWarning returns an error, the scan MUST log that error
// to verbose output and continue — it MUST NOT abort the walk or return the
// write error. A subsequent small file MUST still be reported.
//
// This lives in package adapters (internal test) so it can swap the
// unexported stderrWriter seam.
func TestEmitOversizeWarning_StderrWriteFailureContinuesScan(t *testing.T) {
	tests := []struct {
		name       string
		adapter    GenericAdapter
		categories []string
		largeFile  string // oversized -> triggers emitOversizeWarning -> failingWriter
		smallFile  string // MUST still be reported after the warning fails
	}{
		{
			name: "multi-category adapter (root files)",
			adapter: GenericAdapter{
				AdapterName:   "stderr-fail-multicat",
				ConfigRelPath: ".test",
				Categories: map[string]CategoryDir{
					"config": {SubPath: "", IsDir: false},
					"mcp":    {SubPath: "", IsDir: false},
				},
				DetectErrContext: "stat stderr-fail-multicat config dir",
				RootConfigFiles: map[string]string{
					"opencode.json": "config",
					"mcp.json":      "mcp",
				},
			},
			categories: []string{"config", "mcp"},
			largeFile:  "mcp.json",
			smallFile:  "opencode.json",
		},
		{
			name: "legacy config-only adapter (nil RootConfigFiles)",
			adapter: GenericAdapter{
				AdapterName:   "stderr-fail-legacy",
				ConfigRelPath: ".test",
				Categories: map[string]CategoryDir{
					"config": {SubPath: "", IsDir: false},
				},
				DetectErrContext: "stat stderr-fail-legacy config dir",
			},
			categories: []string{"config"},
			largeFile:  "large.log",
			smallFile:  "small.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home := t.TempDir()
			configDir := filepath.Join(home, ".test")
			if err := os.MkdirAll(configDir, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(configDir, tt.largeFile), bytes.Repeat([]byte("x"), 200), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(configDir, tt.smallFile), []byte("ok"), 0o644); err != nil {
				t.Fatal(err)
			}

			ga := tt.adapter
			ga.ScanOpts = ScanOptions{MaxFileSize: 100}

			// Inject a writer that always fails. Restore afterward.
			origWriter := stderrWriter
			stderrWriter = failingWriter{}
			defer func() { stderrWriter = origWriter }()

			// Redirect the REAL os.Stderr so the verbose log of the write error
			// is captured (and does not pollute test output). We confirm the
			// write error was logged, not abortive.
			warnLog := captureStderrInternal(t, func() {
				items, err := ga.ListItems(home, tt.categories)
				if err != nil {
					t.Fatalf("ListItems must not return the stderr write error, got: %v", err)
				}

				// The oversized file MUST be skipped.
				for _, it := range items {
					if it.RelPath == tt.largeFile {
						t.Errorf("oversized %q should have been skipped, got %+v", tt.largeFile, it)
					}
				}
				// The subsequent small file MUST still be reported — the walk continued.
				foundSmall := false
				for _, it := range items {
					if it.RelPath == tt.smallFile {
						foundSmall = true
					}
				}
				if !foundSmall {
					t.Errorf("%q should still be reported after stderr write failure; walk aborted. items=%+v", tt.smallFile, items)
				}
			})

			// The write error MUST have been logged (to verbose/stderr), proving
			// it was observed rather than silently swallowed — and not returned.
			if !strings.Contains(warnLog, "simulated stderr write failure") {
				t.Errorf("expected the stderr write error to be logged to verbose output, got:\n%s", warnLog)
			}
		})
	}
}
