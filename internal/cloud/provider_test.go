package cloud

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// mockProvider implements Provider for testing the ProviderRegistry.
type mockProvider struct {
	name   string
	pushFn func([]byte, PushMeta) (string, error)
	pullFn func(string) ([]byte, error)
	listFn func() ([]BackupMeta, error)
}

func (m *mockProvider) Name() string                              { return m.name }
func (m *mockProvider) Push(a []byte, meta PushMeta) (string, error) {
	if m.pushFn != nil {
		return m.pushFn(a, meta)
	}
	return "mock-id", nil
}
func (m *mockProvider) Pull(id string) ([]byte, error) {
	if m.pullFn != nil {
		return m.pullFn(id)
	}
	return []byte("mock-data"), nil
}
func (m *mockProvider) List() ([]BackupMeta, error) {
	if m.listFn != nil {
		return m.listFn()
	}
	return nil, nil
}

func TestProviderRegistry_RegisterAndGet(t *testing.T) {
	reg := NewProviderRegistry()
	p := &mockProvider{name: "test-provider"}

	if err := reg.Register(p); err != nil {
		t.Fatalf("Register: unexpected error: %v", err)
	}

	got, err := reg.Get("test-provider")
	if err != nil {
		t.Fatalf("Get: unexpected error: %v", err)
	}
	if got.Name() != "test-provider" {
		t.Errorf("got name %q, want test-provider", got.Name())
	}
}

func TestProviderRegistry_GetUnknown(t *testing.T) {
	reg := NewProviderRegistry()
	_, err := reg.Get("nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
	if !strings.Contains(err.Error(), "unknown provider") {
		t.Errorf("error = %v, want mention of 'unknown provider'", err)
	}
	if !strings.Contains(err.Error(), "nonexistent") {
		t.Errorf("error = %v, want provider name in error", err)
	}
}

func TestProviderRegistry_GetDefault(t *testing.T) {
	reg := NewProviderRegistry()
	p := &mockProvider{name: "default-test"}

	if err := reg.Register(p); err != nil {
		t.Fatalf("Register: %v", err)
	}
	reg.SetDefault("default-test")

	got, err := reg.Get("")
	if err != nil {
		t.Fatalf("Get empty: unexpected error: %v", err)
	}
	if got.Name() != "default-test" {
		t.Errorf("got default %q, want default-test", got.Name())
	}
}

func TestProviderRegistry_GetDefault_NotSet(t *testing.T) {
	reg := NewProviderRegistry()
	_, err := reg.Get("")
	if err == nil {
		t.Fatal("expected error when no default set")
	}
}

func TestProviderRegistry_RegisterDuplicate(t *testing.T) {
	reg := NewProviderRegistry()
	p1 := &mockProvider{name: "dup"}
	p2 := &mockProvider{name: "dup"}

	if err := reg.Register(p1); err != nil {
		t.Fatalf("Register first: %v", err)
	}
	err := reg.Register(p2)
	if err == nil {
		t.Fatal("expected error for duplicate registration")
	}
}

func TestProviderRegistry_SetDefault_Unknown(t *testing.T) {
	reg := NewProviderRegistry()
	// SetDefault should be a no-op for unregistered names.
	reg.SetDefault("ghost")

	_, err := reg.Get("")
	if err == nil {
		t.Fatal("expected error when default points to unregistered provider")
	}
}

func TestPushMeta_Fields(t *testing.T) {
	now := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)
	meta := PushMeta{
		BackupID:  "20260605-120000",
		CreatedAt: now,
		Hostname:  "devbox",
		OS:        "windows",
		Agents:    []string{"opencode", "claude-code"},
	}

	if meta.BackupID != "20260605-120000" {
		t.Errorf("BackupID = %q, want 20260605-120000", meta.BackupID)
	}
	if !meta.CreatedAt.Equal(now) {
		t.Errorf("CreatedAt = %v, want %v", meta.CreatedAt, now)
	}
	if meta.Hostname != "devbox" {
		t.Errorf("Hostname = %q, want devbox", meta.Hostname)
	}
	if meta.OS != "windows" {
		t.Errorf("OS = %q, want windows", meta.OS)
	}
	if len(meta.Agents) != 2 {
		t.Errorf("Agents length = %d, want 2", len(meta.Agents))
	}
}

func TestBackupMeta_Fields(t *testing.T) {
	now := time.Date(2026, 6, 4, 10, 30, 0, 0, time.UTC)
	meta := BackupMeta{
		ID:        "gist-abc123",
		BackupID:  "20260604-103000",
		CreatedAt: now,
		Hostname:  "laptop",
		Size:      102400,
		URL:       "https://gist.github.com/abc123",
	}

	if meta.ID != "gist-abc123" {
		t.Errorf("ID = %q, want gist-abc123", meta.ID)
	}
	if meta.BackupID != "20260604-103000" {
		t.Errorf("BackupID = %q, want 20260604-103000", meta.BackupID)
	}
	if meta.Hostname != "laptop" {
		t.Errorf("Hostname = %q, want laptop", meta.Hostname)
	}
	if meta.Size != 102400 {
		t.Errorf("Size = %d, want 102400", meta.Size)
	}
	if meta.URL != "https://gist.github.com/abc123" {
		t.Errorf("URL = %q, want https://gist.github.com/abc123", meta.URL)
	}
}

func TestProviderRegistry_MultipleProviders(t *testing.T) {
	reg := NewProviderRegistry()
	reg.Register(&mockProvider{name: "first"})
	reg.Register(&mockProvider{name: "second"})
	reg.Register(&mockProvider{name: "third"})

	for _, name := range []string{"first", "second", "third"} {
		p, err := reg.Get(name)
		if err != nil {
			t.Errorf("Get(%q): unexpected error: %v", name, err)
			continue
		}
		if p.Name() != name {
			t.Errorf("Get(%q).Name() = %q, want %q", name, p.Name(), name)
		}
	}
}

func TestProviderRegistry_NilProvider(t *testing.T) {
	reg := NewProviderRegistry()
	err := reg.Register(nil)
	if err == nil {
		t.Fatal("expected error for nil provider")
	}
}

func TestProviderRegistry_PushPullList_Delegation(t *testing.T) {
	reg := NewProviderRegistry()

	calledPush := false
	calledPull := false
	calledList := false

	mp := &mockProvider{
		name: "delegator",
		pushFn: func(_ []byte, _ PushMeta) (string, error) {
			calledPush = true
			return "delegated-id", nil
		},
		pullFn: func(id string) ([]byte, error) {
			calledPull = true
			if id != "delegated-id" {
				return nil, errors.New("unexpected id")
			}
			return []byte("delegated-data"), nil
		},
		listFn: func() ([]BackupMeta, error) {
			calledList = true
			return []BackupMeta{
				{ID: "b1", BackupID: "20250601-120000"},
			}, nil
		},
	}
	reg.Register(mp)

	// Push.
	id, err := mp.Push([]byte("archive"), PushMeta{BackupID: "test"})
	if err != nil {
		t.Fatalf("Push: %v", err)
	}
	if id != "delegated-id" {
		t.Errorf("Push id = %q, want delegated-id", id)
	}
	if !calledPush {
		t.Error("pushFn was not called")
	}

	// Pull.
	data, err := mp.Pull("delegated-id")
	if err != nil {
		t.Fatalf("Pull: %v", err)
	}
	if string(data) != "delegated-data" {
		t.Errorf("Pull data = %q, want delegated-data", string(data))
	}
	if !calledPull {
		t.Error("pullFn was not called")
	}

	// List.
	metas, err := mp.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(metas) != 1 {
		t.Fatalf("List length = %d, want 1", len(metas))
	}
	if metas[0].ID != "b1" {
		t.Errorf("List[0].ID = %q, want b1", metas[0].ID)
	}
	if !calledList {
		t.Error("listFn was not called")
	}
}
