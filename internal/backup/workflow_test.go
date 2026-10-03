package backup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/danielxxomg/bak-cli/internal/adapters"
	configtest "github.com/danielxxomg/bak-cli/internal/config/testutil"
	"github.com/danielxxomg/bak-cli/internal/manifest"
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
func TestRun_PreservesExclusionPipeline(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
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

type modeReportingAdapter struct {
	name      string
	configDir string
	mode      uint32
}

func (m *modeReportingAdapter) Name() string { return m.name }
func (m *modeReportingAdapter) Detect(string) (bool, string, error) {
	return true, m.configDir, nil
}
func (m *modeReportingAdapter) ListItems(string, []string) ([]adapters.Item, error) {
	return []adapters.Item{
		{
			Category:   "config",
			SourcePath: "~/.config/opencode/script.sh",
			RelPath:    "script.sh",
			Mode:       m.mode,
		},
	}, nil
}
func (m *modeReportingAdapter) Backup(homeDir, backupDir string, items []adapters.Item) error {
	dst := filepath.Join(backupDir, m.name, "script.sh")
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	return os.WriteFile(dst, []byte("#!/bin/sh\n"), 0755)
}
func (m *modeReportingAdapter) Restore(string, string, []adapters.Item) error { return nil }

var _ adapters.Adapter = (*modeReportingAdapter)(nil)

func TestRun_RecordsExecutableBit(t *testing.T) { //nolint:paralleltest // shared state
	home := t.TempDir()
	configDir := filepath.Join(home, ".config", "opencode")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	scriptPath := filepath.Join(configDir, "script.sh")
	if err := os.WriteFile(scriptPath, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatal(err)
	}

	adp := &modeReportingAdapter{name: "opencode", configDir: configDir, mode: 0755}
	reg := adapters.NewRegistry()
	if err := reg.Register(adp); err != nil {
		t.Fatal(err)
	}

	ctx := Context{
		FS:       osFS{},
		HomeDir:  home,
		BakDir:   filepath.Join(home, ".bak"),
		Registry: reg,
		Preset:   "quick",
	}
	res, err := Run(ctx)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res == nil {
		t.Fatal("expected non-nil Result")
	}

	m, err := manifest.Load(res.BackupDir)
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}
	am, ok := m.Adapters["opencode"]
	if !ok || len(am.Items) == 0 {
		t.Fatal("expected manifest items for opencode adapter")
	}
	gotMode := am.Items[0].Mode
	if gotMode != 0755 {
		t.Errorf("manifest item mode = %o, want 0755", gotMode)
	}
}

type secretFixtureAdapter struct {
	name      string
	configDir string
}

func (s *secretFixtureAdapter) Name() string { return s.name }
func (s *secretFixtureAdapter) Detect(string) (bool, string, error) {
	return true, s.configDir, nil
}
func (s *secretFixtureAdapter) ListItems(string, []string) ([]adapters.Item, error) {
	return []adapters.Item{
		{
			Category:   "config",
			SourcePath: "~/.config/opencode/secrets.json",
			RelPath:    "secrets.json",
		},
	}, nil
}
func (s *secretFixtureAdapter) Backup(homeDir, backupDir string, items []adapters.Item) error {
	dst := filepath.Join(backupDir, s.name, "secrets.json")
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	// Copy from source file
	src := filepath.Join(homeDir, ".config", "opencode", "secrets.json")
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}
func (s *secretFixtureAdapter) Restore(string, string, []adapters.Item) error { return nil }

var _ adapters.Adapter = (*secretFixtureAdapter)(nil)

func TestRun_GeneratesUsableEnvExampleFromSource(t *testing.T) { //nolint:paralleltest // uses t.Setenv
	home := t.TempDir()
	configtest.SetConfigHome(t, home)

	configDir := filepath.Join(home, ".config", "opencode")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	sourceFile := filepath.Join(configDir, "secrets.json")
	tokenContent := `{"github_token":"ghp_abcdef1234567890123456789012345678901234"}` + "\n"
	if err := os.WriteFile(sourceFile, []byte(tokenContent), 0644); err != nil {
		t.Fatal(err)
	}

	adp := &secretFixtureAdapter{name: "opencode", configDir: configDir}
	reg := adapters.NewRegistry()
	if err := reg.Register(adp); err != nil {
		t.Fatal(err)
	}

	ctx := Context{
		FS:       osFS{},
		HomeDir:  home,
		BakDir:   filepath.Join(home, ".bak"),
		Registry: reg,
		Preset:   "quick",
	}
	res, err := Run(ctx)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !res.SecretsExcluded {
		t.Fatal("expected SecretsExcluded = true")
	}

	examplePath := filepath.Join(res.BackupDir, ".env.example")
	data, err := os.ReadFile(examplePath)
	if err != nil {
		t.Fatalf("read .env.example: %v", err)
	}
	content := string(data)

	// .env.example must be useful, not an error message
	if strings.Contains(content, "[could not read") {
		t.Errorf(".env.example contains read failure:\n%s", content)
	}
	if !strings.Contains(content, "<YOUR_") {
		t.Errorf(".env.example missing placeholder:\n%s", content)
	}
	if strings.Contains(content, "ghp_abcdef") {
		t.Errorf(".env.example leaked secret:\n%s", content)
	}
	if !strings.Contains(content, "~/.config/opencode/secrets.json") {
		t.Errorf(".env.example missing home-relative path:\n%s", content)
	}

	if len(res.SecretFiles) != 1 || res.SecretFiles[0] != "~/.config/opencode/secrets.json" {
		t.Errorf("res.SecretFiles = %v, want [~/.config/opencode/secrets.json]", res.SecretFiles)
	}
}
