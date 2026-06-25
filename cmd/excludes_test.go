package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/danielxxomg/bak-cli/internal/config"
	configtest "github.com/danielxxomg/bak-cli/internal/config/testutil"
)

// defaultMaxFileSize mirrors config.applyDefaults' MaxFileSize default
// (1 MiB), applied by config.Load when the setting is zero-value.
const defaultMaxFileSize int64 = 1048576

// TestLoadExcludes verifies the shared loadExcludes helper resolves scan
// options from config + the ignore file, isolated via SetConfigHome.
func TestLoadExcludes(t *testing.T) {
	tests := []struct {
		name          string
		ignoreContent string   // empty = do not create an ignore file
		wantExtra     []string // patterns appended after the defaults
		wantMaxSize   int64
	}{
		{
			name:        "returns default excludes when no ignore file",
			wantExtra:   nil,
			wantMaxSize: defaultMaxFileSize,
		},
		{
			name:          "appends custom ignore-file pattern to defaults",
			ignoreContent: "*.secret\n",
			wantExtra:     []string{"*.secret"},
			wantMaxSize:   defaultMaxFileSize,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			configtest.SetConfigHome(t, dir)

			if tt.ignoreContent != "" {
				// ConfigDir("bak") resolves to <configHome>/bak.
				cfgDir := filepath.Join(dir, "bak")
				if err := os.MkdirAll(cfgDir, 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(cfgDir, "ignore"), []byte(tt.ignoreContent), 0o644); err != nil {
					t.Fatal(err)
				}
			}

			opts, err := loadExcludes()
			if err != nil {
				t.Fatalf("loadExcludes() error = %v", err)
			}

			wantLen := len(config.DefaultExcludes) + len(tt.wantExtra)
			if len(opts.Excludes) != wantLen {
				t.Fatalf("loadExcludes() Excludes len = %d, want %d", len(opts.Excludes), wantLen)
			}

			// Defaults must appear first, in order.
			for i, want := range config.DefaultExcludes {
				if opts.Excludes[i] != want {
					t.Errorf("Excludes[%d] = %q, want default %q", i, opts.Excludes[i], want)
				}
			}

			// Custom patterns must be appended after the defaults, in order.
			for j, want := range tt.wantExtra {
				idx := len(config.DefaultExcludes) + j
				if opts.Excludes[idx] != want {
					t.Errorf("Excludes[%d] = %q, want appended %q", idx, opts.Excludes[idx], want)
				}
			}

			if opts.MaxFileSize != tt.wantMaxSize {
				t.Errorf("MaxFileSize = %d, want %d", opts.MaxFileSize, tt.wantMaxSize)
			}
		})
	}
}
