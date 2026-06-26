package adapters_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/danielxxomg/bak-cli/internal/adapters"
)

func TestScanOptions(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{"zero value", testScanOptionsZeroValue},
		{"set and read back", testScanOptionsSetAndRead},
		{"excludes dirs via ListItems", testScanOptionsExcludesDirs},
		{"max file size via ListItems", testScanOptionsMaxFileSize},
		{"zero value fall-through preserves items", testScanOptionsZeroValueFallThrough},
	}

	for _, tt := range tests { //nolint:paralleltest // subtests share table/struct state
		t.Run(tt.name, tt.test)
	}
}

func testScanOptionsZeroValue(t *testing.T) {
	opts := adapters.ScanOptions{}
	if len(opts.Excludes) != 0 {
		t.Errorf("zero-value ScanOptions.Excludes should be empty, got %d", len(opts.Excludes))
	}
	if opts.MaxFileSize != 0 {
		t.Errorf("zero-value ScanOptions.MaxFileSize should be 0, got %d", opts.MaxFileSize)
	}
}

func testScanOptionsSetAndRead(t *testing.T) {
	adapter := newTestAdapter("test-scan")
	adapter.ScanOpts = adapters.ScanOptions{
		Excludes:    []string{"node_modules", "*.tmp"},
		MaxFileSize: 1048576,
	}

	if len(adapter.ScanOpts.Excludes) != 2 {
		t.Errorf("expected 2 excludes, got %d", len(adapter.ScanOpts.Excludes))
	}
	if adapter.ScanOpts.MaxFileSize != 1048576 {
		t.Errorf("expected MaxFileSize 1048576, got %d", adapter.ScanOpts.MaxFileSize)
	}
}

func testScanOptionsExcludesDirs(t *testing.T) {
	home := setupConfigHome(t)
	configDir := filepath.Join(home, ".test")

	// Create an excluded directory with files.
	nodeModulesDir := filepath.Join(configDir, "node_modules")
	if err := os.MkdirAll(nodeModulesDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nodeModulesDir, "package.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nodeModulesDir, "index.js"), []byte("console.log(1)"), 0644); err != nil {
		t.Fatal(err)
	}

	adapter := newTestAdapter("test-scan")
	adapter.ScanOpts = adapters.ScanOptions{
		Excludes: []string{"node_modules/"},
	}

	items, err := adapter.ListItems(home, []string{"config", "scripts"})
	if err != nil {
		t.Fatal(err)
	}

	for _, item := range items {
		if contains(item.SourcePath, "node_modules") {
			t.Errorf("expected node_modules to be excluded, but found item: %s", item.SourcePath)
		}
	}
}

func testScanOptionsMaxFileSize(t *testing.T) {
	home := setupConfigHome(t)
	configDir := filepath.Join(home, ".test")

	scriptsDir := filepath.Join(configDir, "scripts")
	if err := os.MkdirAll(scriptsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(scriptsDir, "small.sh"), []byte("#!/bin/bash\n"), 0644); err != nil {
		t.Fatal(err)
	}
	largeContent := make([]byte, 1024)
	for i := range largeContent {
		largeContent[i] = 'x'
	}
	largePath := filepath.Join(scriptsDir, "large.log")
	if err := os.WriteFile(largePath, largeContent, 0644); err != nil {
		t.Fatal(err)
	}

	adapter := newTestAdapter("test-scan")
	adapter.ScanOpts = adapters.ScanOptions{
		MaxFileSize: 512,
	}

	items, err := adapter.ListItems(home, []string{"scripts"})
	if err != nil {
		t.Fatal(err)
	}

	for _, item := range items {
		if item.RelPath == "scripts/large.log" {
			t.Errorf("expected large.log to be excluded (size > MaxFileSize), but it was included")
		}
	}

	foundSmall := false
	for _, item := range items {
		if item.RelPath == "scripts/small.sh" {
			foundSmall = true
			break
		}
	}
	if !foundSmall {
		t.Errorf("expected small.sh to be included (size < MaxFileSize), but it was excluded")
	}
}

func testScanOptionsZeroValueFallThrough(t *testing.T) {
	home := setupConfigHome(t)

	adapter := newTestAdapter("test-scan")
	adapter.ScanOpts = adapters.ScanOptions{} // zero value

	items, err := adapter.ListItems(home, []string{"config", "scripts"})
	if err != nil {
		t.Fatal(err)
	}

	if len(items) == 0 {
		t.Fatal("expected items with zero-value ScanOptions, got none")
	}

	foundConfig := false
	foundScript := false
	for _, item := range items {
		if item.RelPath == "settings.json" && item.Category == "config" {
			foundConfig = true
		}
		if item.RelPath == "scripts/deploy.sh" {
			foundScript = true
		}
	}
	if !foundConfig {
		t.Error("expected settings.json in config category")
	}
	if !foundScript {
		t.Error("expected deploy.sh in scripts category")
	}
}

func contains(s, substr string) bool {
	return strings.Contains(
		strings.ReplaceAll(s, "\\", "/"),
		strings.ReplaceAll(substr, "\\", "/"),
	)
}
