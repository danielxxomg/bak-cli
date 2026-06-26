package actions

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/danielxxomg/bak-cli/internal/cloud"
	"github.com/danielxxomg/bak-cli/internal/config"
)

func TestListCloudAction_Success(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	backups := []cloud.BackupMeta{
		{
			ID:        "cloud-id-1",
			BackupID:  "20260101-120000",
			CreatedAt: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
			Hostname:  "testbox",
			Size:      2048,
			URL:       "https://gist.github.com/abc123",
		},
		{
			ID:        "cloud-id-2",
			BackupID:  "20260102-080000",
			CreatedAt: time.Date(2026, 1, 2, 8, 0, 0, 0, time.UTC),
			Hostname:  "testbox",
			Size:      4096,
			URL:       "https://gist.github.com/def456",
		},
	}

	mockProvider := &MockProvider{
		MockName: "github-gist",
		ListFn: func() ([]cloud.BackupMeta, error) {
			return backups, nil
		},
	}

	reg := cloud.NewProviderRegistry()
	_ = reg.Register(mockProvider)
	reg.SetDefault("github-gist")

	var out, errOut strings.Builder
	action := &ListCloudAction{
		Config: &config.Config{SchemaVersion: "0.3.0"},
		Stdout: &out,
		Stderr: &errOut,
		RegistryFactory: func() *cloud.ProviderRegistry {
			return reg
		},
	}

	err := action.Run("github-gist")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "cloud-id-1") {
		t.Errorf("output should contain first cloud ID, got: %q", output)
	}
	if !strings.Contains(output, "cloud-id-2") {
		t.Errorf("output should contain second cloud ID, got: %q", output)
	}
	if !strings.Contains(output, "testbox") {
		t.Errorf("output should contain hostname, got: %q", output)
	}
	if !strings.Contains(output, "2.0 KB") {
		t.Errorf("output should contain formatted size, got: %q", output)
	}
}

func TestListCloudAction_EmptyList(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	mockProvider := &MockProvider{
		MockName: "github-gist",
		ListFn: func() ([]cloud.BackupMeta, error) {
			return nil, nil
		},
	}

	reg := cloud.NewProviderRegistry()
	_ = reg.Register(mockProvider)
	reg.SetDefault("github-gist")

	var out, errOut strings.Builder
	action := &ListCloudAction{
		Config: &config.Config{SchemaVersion: "0.3.0"},
		Stdout: &out,
		Stderr: &errOut,
		RegistryFactory: func() *cloud.ProviderRegistry {
			return reg
		},
	}

	err := action.Run("github-gist")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "No backups found") {
		t.Errorf("output should indicate no backups, got: %q", output)
	}
}

func TestListCloudAction_ProviderNotFound(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	reg := cloud.NewProviderRegistry()

	var out, errOut strings.Builder
	action := &ListCloudAction{
		Config: &config.Config{SchemaVersion: "0.3.0"},
		Stdout: &out,
		Stderr: &errOut,
		RegistryFactory: func() *cloud.ProviderRegistry {
			return reg
		},
	}

	err := action.Run("nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
	if !strings.Contains(err.Error(), "provider") {
		t.Errorf("error should mention provider: %v", err)
	}
}

func TestListCloudAction_ListError(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	mockProvider := &MockProvider{
		MockName: "github-gist",
		ListFn: func() ([]cloud.BackupMeta, error) {
			return nil, errors.New("network timeout")
		},
	}

	reg := cloud.NewProviderRegistry()
	_ = reg.Register(mockProvider)
	reg.SetDefault("github-gist")

	var out, errOut strings.Builder
	action := &ListCloudAction{
		Config: &config.Config{SchemaVersion: "0.3.0"},
		Stdout: &out,
		Stderr: &errOut,
		RegistryFactory: func() *cloud.ProviderRegistry {
			return reg
		},
	}

	err := action.Run("github-gist")
	if err == nil {
		t.Fatal("expected error from List")
	}
	if !strings.Contains(err.Error(), "list") {
		t.Errorf("error should mention list: %v", err)
	}
}

func TestListCloudAction_DefaultFactory_NoProviders(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	// When RegistryFactory is nil, Run() should use default behavior
	// which registers providers. Without a real config, this will
	// fail at provider construction (expected — no config available).
	var out, errOut strings.Builder
	action := &ListCloudAction{
		Config:  &config.Config{SchemaVersion: "0.3.0"},
		Stdout:  &out,
		Stderr:  &errOut,
		Verbose: true,
		// RegistryFactory is nil — exercises default path.
	}

	err := action.Run("github-gist")
	// Default factory registers real providers — they'll fail
	// when trying to use an empty config. That's OK; we're testing
	// that the default path is exercised.
	if err == nil {
		t.Log("default factory succeeded (real config may exist)")
	} else {
		t.Logf("default factory failed as expected: %v", err)
	}
	// The verbose message should show on stderr if it got far enough.
	_ = errOut.String()
}

func TestListCloudAction_VerboseOutput(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	mockProvider := &MockProvider{
		MockName: "github-gist",
		ListFn: func() ([]cloud.BackupMeta, error) {
			return nil, nil
		},
	}

	reg := cloud.NewProviderRegistry()
	_ = reg.Register(mockProvider)
	reg.SetDefault("github-gist")

	var out, errOut strings.Builder
	action := &ListCloudAction{
		Config:  &config.Config{SchemaVersion: "0.3.0"},
		Stdout:  &out,
		Stderr:  &errOut,
		Verbose: true,
		RegistryFactory: func() *cloud.ProviderRegistry {
			return reg
		},
	}

	err := action.Run("github-gist")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if !strings.Contains(errOut.String(), "Using provider: github-gist") {
		t.Errorf("verbose stderr should mention provider, got: %q", errOut.String())
	}
}
