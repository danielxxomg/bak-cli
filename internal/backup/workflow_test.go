package backup

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/danielxxomg/bak-cli/internal/adapters"
)

// scanTrackingAdapter records whether SetScanOptions was called so the
// exclusion-pipeline test can assert it is invoked before ListItems.
type scanTrackingAdapter struct {
	name        string
	configDir   string
	scanOptsSet bool
	listCalled  bool
}

func (s *scanTrackingAdapter) Name() string { return s.name }
func (s *scanTrackingAdapter) Detect(string) (bool, string, error) {
	return true, s.configDir, nil
}
func (s *scanTrackingAdapter) SetScanOptions(_ adapters.ScanOptions) {
	s.scanOptsSet = true
}
func (s *scanTrackingAdapter) ListItems(_ string, _ []string) ([]adapters.Item, error) {
	s.listCalled = true
	return nil, nil
}
func (s *scanTrackingAdapter) Backup(_, _ string, _ []adapters.Item) error  { return nil }
func (s *scanTrackingAdapter) Restore(_, _ string, _ []adapters.Item) error { return nil }

var _ adapters.Adapter = (*scanTrackingAdapter)(nil)
var _ adapters.ScanConfigurable = (*scanTrackingAdapter)(nil)

// TestRun_PreservesExclusionPipeline asserts the spec scenario "Consolidated
// engine preserves exclusion pipeline": when ExcludesLoader is set, it MUST
// be called and SetScanOptions MUST be invoked on every ScanConfigurable
// adapter before ListItems runs.
func TestRun_PreservesExclusionPipeline(t *testing.T) {
	home := t.TempDir()
	configDir := filepath.Join(home, ".config", "opencode")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	adp := &scanTrackingAdapter{name: "tracker", configDir: configDir}
	reg := adapters.NewRegistry()
	if err := reg.Register(adp); err != nil {
		t.Fatal(err)
	}

	loaderCalled := false
	ctx := Context{
		FS:       osFS{},
		HomeDir:  home,
		BakDir:   filepath.Join(home, ".bak"),
		Registry: reg,
		Preset:   "quick",
		ExcludesLoader: func() (adapters.ScanOptions, error) {
			loaderCalled = true
			return adapters.ScanOptions{Excludes: []string{"*.log"}}, nil
		},
	}
	if _, err := Run(ctx); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if !loaderCalled {
		t.Error("ExcludesLoader was not called")
	}
	if !adp.scanOptsSet {
		t.Error("SetScanOptions was not called on the ScanConfigurable adapter")
	}
	if !adp.listCalled {
		t.Error("ListItems was not called")
	}
	if adp.listCalled && !adp.scanOptsSet {
		t.Error("ListItems ran before SetScanOptions — exclusion pipeline order violated")
	}
}
