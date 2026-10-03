package manifest

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNew(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := New("20260604-test", "linux", "testbox", "0.1.0", "full", []string{"skills", "config"})

	if m.Version != ManifestVersion {
		t.Errorf("version = %q, want %q", m.Version, ManifestVersion)
	}
	if m.ID != "20260604-test" {
		t.Errorf("id = %q, want %q", m.ID, "20260604-test")
	}
	if m.OSSource != "linux" {
		t.Errorf("os_source = %q, want %q", m.OSSource, "linux")
	}
	if m.Preset != "full" {
		t.Errorf("preset = %q, want %q", m.Preset, "full")
	}
	if len(m.Categories) != 2 {
		t.Errorf("categories len = %d, want 2", len(m.Categories))
	}
	if m.CreatedAt.After(time.Now()) {
		t.Error("created_at is in the future")
	}
	if m.Adapters == nil {
		t.Error("adapters map is nil")
	}
}

func TestSaveLoadRoundTrip(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	dir := t.TempDir()

	m := New("20260604-roundtrip", "darwin", "macbook", "0.1.0", "quick", []string{"config"})
	m.AddAdapter("opencode", "1.5.0", "~/.config/opencode", []Item{
		{Category: "config", SourcePath: "~/.config/opencode/config.json", BackupPath: "opencode/config.json", Hash: "sha256:abc123", Size: 512},
	})

	if err := m.Save(dir); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.ID != m.ID {
		t.Errorf("id = %q, want %q", loaded.ID, m.ID)
	}
	if loaded.Preset != m.Preset {
		t.Errorf("preset = %q, want %q", loaded.Preset, m.Preset)
	}
	if loaded.FileCount != 1 {
		t.Errorf("file_count = %d, want 1", loaded.FileCount)
	}
	if loaded.TotalSize != 512 {
		t.Errorf("total_size = %d, want 512", loaded.TotalSize)
	}

	am, ok := loaded.Adapters["opencode"]
	if !ok {
		t.Fatal("opencode adapter not found in loaded manifest")
	}
	if am.VersionDetected != "1.5.0" {
		t.Errorf("version_detected = %q, want %q", am.VersionDetected, "1.5.0")
	}
	if len(am.Items) != 1 {
		t.Fatalf("items len = %d, want 1", len(am.Items))
	}
	if am.Items[0].Hash != "sha256:abc123" {
		t.Errorf("hash = %q, want %q", am.Items[0].Hash, "sha256:abc123")
	}
}

func TestValidate_HappyPath(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	dir := t.TempDir()

	// Create a real file with known content.
	backupFile := filepath.Join(dir, "opencode", "config.json")
	if err := os.MkdirAll(filepath.Dir(backupFile), 0755); err != nil {
		t.Fatal(err)
	}
	content := []byte(`{"key": "value"}`)
	if err := os.WriteFile(backupFile, content, 0644); err != nil {
		t.Fatal(err)
	}

	m := New("validate-test", "linux", "box", "0.1.0", "full", []string{"config"})
	m.AddAdapter("opencode", "0.1.0", "~/.config/opencode", []Item{
		{Category: "config", SourcePath: "~/.config/opencode/config.json", BackupPath: "opencode/config.json", Hash: "sha256:9724c1e20e6e3e4d7f57ed25f9d4efb006e508590d528c90da597f6a775c13e5", Size: 16},
	})

	if err := m.Validate(dir, nil); err != nil {
		t.Errorf("Validate: unexpected error: %v", err)
	}
}

func TestValidate_HashMismatch(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	dir := t.TempDir()

	backupFile := filepath.Join(dir, "opencode", "config.json")
	if err := os.MkdirAll(filepath.Dir(backupFile), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(backupFile, []byte(`wrong`), 0644); err != nil {
		t.Fatal(err)
	}

	m := New("mismatch-test", "linux", "box", "0.1.0", "full", []string{"config"})
	m.AddAdapter("opencode", "0.1.0", "~/.config/opencode", []Item{
		{Category: "config", SourcePath: "~/.config/opencode/config.json", BackupPath: "opencode/config.json", Hash: "sha256:0000000000000000000000000000000000000000000000000000000000000000", Size: 5},
	})

	err := m.Validate(dir, nil)
	if err == nil {
		t.Error("Validate: expected hash mismatch error, got nil")
	}
}

func TestValidate_EmptyVersion(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := &Manifest{Adapters: map[string]AdapterManifest{"x": {}}}
	if err := m.Validate(".", nil); err == nil {
		t.Error("expected error for empty version")
	}
}

func TestValidate_NoAdapters(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := &Manifest{Version: "0.1.0"}
	if err := m.Validate(".", nil); err == nil {
		t.Error("expected error for no adapters")
	}
}

func TestLoad_NonExistent(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	_, err := Load(t.TempDir())
	if err == nil {
		t.Error("expected error for missing manifest.json")
	}
}

// ---- Encryption struct tests ----

func TestSetEncryption(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := New("enc-test", "linux", "box", "0.1.0", "full", []string{"config"})
	salt := []byte{0x01, 0x02, 0x03}
	nonce := []byte{0xAA, 0xBB, 0xCC}

	m.SetEncryption("AES-256-GCM", "Argon2id", salt, nonce, 3, 65536, 4)

	if m.Encryption == nil {
		t.Fatal("Encryption is nil after SetEncryption")
	}
	e := m.Encryption
	if e.Algorithm != "AES-256-GCM" {
		t.Errorf("algorithm = %q, want %q", e.Algorithm, "AES-256-GCM")
	}
	if e.KDF != "Argon2id" {
		t.Errorf("kdf = %q, want %q", e.KDF, "Argon2id")
	}
	if e.Salt != "010203" {
		t.Errorf("salt = %q, want %q", e.Salt, "010203")
	}
	if e.Nonce != "aabbcc" {
		t.Errorf("nonce = %q, want %q", e.Nonce, "aabbcc")
	}
	if e.Iterations != 3 {
		t.Errorf("iterations = %d, want 3", e.Iterations)
	}
	if e.MemoryKB != 65536 {
		t.Errorf("memory_kb = %d, want 65536", e.MemoryKB)
	}
	if e.Parallelism != 4 {
		t.Errorf("parallelism = %d, want 4", e.Parallelism)
	}
}

func TestManifest_JSON_EncryptionRoundTrip(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	dir := t.TempDir()

	m := New("enc-json-test", "darwin", "mac", "0.3.0", "full", []string{"config"})
	m.AddAdapter("opencode", "1.0.0", "~/.config/opencode", []Item{
		{Category: "config", SourcePath: "~/.config/opencode/config.json", BackupPath: "opencode/config.json", Hash: "sha256:abc123", Size: 512},
	})

	fullSalt := make([]byte, 32)
	fullNonce := make([]byte, 12)
	for i := range fullSalt {
		fullSalt[i] = byte(i % 256)
	}
	for i := range fullNonce {
		fullNonce[i] = byte((i * 17) % 256)
	}

	m.SetEncryption("AES-256-GCM", "Argon2id", fullSalt, fullNonce, 3, 65536, 4)

	// Save and reload.
	if err := m.Save(dir); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.Encryption == nil {
		t.Fatal("loaded Encryption is nil — should be preserved")
	}

	e := loaded.Encryption
	if e.Algorithm != "AES-256-GCM" {
		t.Errorf("algorithm = %q, want AES-256-GCM", e.Algorithm)
	}
	if e.KDF != "Argon2id" {
		t.Errorf("kdf = %q, want Argon2id", e.KDF)
	}
	if e.Salt != "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f" {
		t.Errorf("salt = %q", e.Salt)
	}
	if e.Nonce != "00112233445566778899aabb" {
		t.Errorf("nonce = %q", e.Nonce)
	}
	if e.Iterations != 3 {
		t.Errorf("iterations = %d, want 3", e.Iterations)
	}
	if e.MemoryKB != 65536 {
		t.Errorf("memory_kb = %d, want 65536", e.MemoryKB)
	}
	if e.Parallelism != 4 {
		t.Errorf("parallelism = %d, want 4", e.Parallelism)
	}
}

func TestManifest_JSON_PlaintextOmitsEncryption(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	dir := t.TempDir()

	m := New("plain-json-test", "linux", "srv", "0.3.0", "quick", []string{"config"})
	m.AddAdapter("opencode", "1.0.0", "~/.config/opencode", []Item{
		{Category: "config", SourcePath: "~/.config/opencode/config.json", BackupPath: "opencode/config.json", Hash: "sha256:abc123", Size: 512},
	})

	// Do NOT call SetEncryption — should be nil.
	if err := m.Save(dir); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.Encryption != nil {
		t.Error("Encryption should be nil for plaintext manifest")
	}

	// Read raw JSON and verify "encryption" key is absent.
	raw, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		t.Fatalf("read raw manifest: %v", err)
	}
	if bytes.Contains(raw, []byte(`"encryption"`)) {
		t.Error("raw JSON contains 'encryption' key for plaintext manifest")
	}
}

func TestSetEncryption_NilSaltNonce(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := New("nil-test", "linux", "box", "0.1.0", "full", []string{"config"})

	// nil salt and nonce should produce empty hex strings.
	m.SetEncryption("AES-256-GCM", "Argon2id", nil, nil, 3, 65536, 4)

	if m.Encryption == nil {
		t.Fatal("Encryption is nil")
	}
	if m.Encryption.Salt != "" {
		t.Errorf("salt = %q, want empty", m.Encryption.Salt)
	}
	if m.Encryption.Nonce != "" {
		t.Errorf("nonce = %q, want empty", m.Encryption.Nonce)
	}
}

func TestManifest_NewVersion_040(t *testing.T) { //nolint:paralleltest // shared state
	m := New("test-v040", "linux", "box", "0.4.0", "quick", []string{"config"})
	if m.Version != "0.4.0" {
		t.Errorf("manifest version = %q, want 0.4.0", m.Version)
	}
}

func TestManifest_ModeRoundTrip(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		mode uint32
	}{
		{"executable script", 0755},
		{"regular file", 0644},
		{"private file", 0600},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			m := New("test-mode", "linux", "host", "0.4.0", "quick", []string{"config"})
			m.AddAdapter("opencode", "1.0.0", "~/.config/opencode", []Item{
				{Category: "config", SourcePath: "~/test", BackupPath: "opencode/test", Hash: "sha256:123", Size: 10, Mode: tt.mode},
			})
			if err := m.Save(dir); err != nil {
				t.Fatalf("Save: %v", err)
			}
			loaded, err := Load(dir)
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if len(loaded.Adapters["opencode"].Items) != 1 {
				t.Fatalf("expected 1 item")
			}
			got := loaded.Adapters["opencode"].Items[0].Mode
			if got != tt.mode {
				t.Errorf("Mode = %o, want %o", got, tt.mode)
			}
		})
	}
}

func TestManifest_LoadV030WithoutMode(t *testing.T) { //nolint:paralleltest // shared state
	dir := t.TempDir()
	rawJSON := `{
  "version": "0.3.0",
  "id": "20260101-legacy",
  "created_at": "2026-01-01T00:00:00Z",
  "os_source": "linux",
  "bak_version": "0.3.0",
  "preset": "quick",
  "categories": ["config"],
  "adapters": {
    "opencode": {
      "config_dir": "~/.config/opencode",
      "items": [
        {
          "category": "config",
          "source_path": "~/.config/opencode/config.json",
          "backup_path": "opencode/config.json",
          "hash": "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
          "size": 0
        }
      ]
    }
  }
}`
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(rawJSON), 0644); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.Version != "0.3.0" {
		t.Errorf("version = %q, want 0.3.0", loaded.Version)
	}
	am := loaded.Adapters["opencode"]
	if len(am.Items) != 1 {
		t.Fatalf("items len = %d, want 1", len(am.Items))
	}
	if am.Items[0].Mode != 0 {
		t.Errorf("expected Mode == 0 for 0.3.0 manifest, got %o", am.Items[0].Mode)
	}
	// Also verify that 0.3.0 manifest passes Validate (after writing backing file).
	backupFile := filepath.Join(dir, "opencode", "config.json")
	if err := os.MkdirAll(filepath.Dir(backupFile), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(backupFile, []byte{}, 0644); err != nil {
		t.Fatal(err)
	}
	if err := loaded.Validate(dir, nil); err != nil {
		t.Errorf("Validate 0.3.0 manifest: %v", err)
	}
}

func TestCompareVersions(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	tests := []struct {
		name    string
		v1      string
		v2      string
		want    int
		wantErr bool
	}{
		{"equal_same", "0.4.0", "0.4.0", 0, false},
		{"older_minor", "0.3.0", "0.4.0", -1, false},
		{"newer_minor", "0.5.0", "0.4.0", 1, false},
		{"older_major", "0.4.0", "1.0.0", -1, false},
		{"newer_major", "1.0.0", "0.4.0", 1, false},
		{"newer_patch", "0.4.1", "0.4.0", 1, false},
		{"older_patch", "0.4.0", "0.4.1", -1, false},
		{"two_components", "0.4", "0.4.0", 0, false},
		{"multi_digit_newer", "0.10.0", "0.4.0", 1, false},
		{"multi_digit_older", "0.4.0", "0.10.0", -1, false},
		{"v_prefix", "v0.4.0", "0.4.0", 0, false},
		{"prerelease_older", "0.4.0-rc1", "0.4.0", -1, false},
		{"empty_v1", "", "0.4.0", 0, true},
		{"empty_v2", "0.4.0", "", 0, true},
		{"invalid_non_numeric", "abc", "0.4.0", 0, true},
	}

	for _, tt := range tests { //nolint:paralleltest
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest
			got, err := CompareVersions(tt.v1, tt.v2)
			if (err != nil) != tt.wantErr {
				t.Fatalf("CompareVersions(%q, %q) error = %v, wantErr %v", tt.v1, tt.v2, err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("CompareVersions(%q, %q) = %d, want %d", tt.v1, tt.v2, got, tt.want)
			}
		})
	}
}

func TestValidateSchemaVersion(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	tests := []struct {
		name       string
		version    string
		wantErr    bool
		errContain string
	}{
		{"current_schema_040", "0.4.0", false, ""},
		{"older_schema_030", "0.3.0", false, ""},
		{"older_schema_010", "0.1.0", false, ""},
		{"newer_minor_050", "0.5.0", true, "unsupported manifest schema version 0.5.0 (maximum supported is 0.4.0): upgrade bak"},
		{"newer_major_100", "1.0.0", true, "unsupported manifest schema version 1.0.0 (maximum supported is 0.4.0): upgrade bak"},
		{"empty_version", "", true, "manifest version is empty"},
		{"invalid_version", "not-a-version", true, "unsupported manifest schema version"},
	}

	for _, tt := range tests { //nolint:paralleltest
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest
			err := ValidateSchemaVersion(tt.version)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateSchemaVersion(%q) error = %v, wantErr %v", tt.version, err, tt.wantErr)
			}
			if tt.wantErr && tt.errContain != "" && (err == nil || !strings.Contains(err.Error(), tt.errContain)) {
				t.Errorf("ValidateSchemaVersion(%q) error = %v, want to contain %q", tt.version, err, tt.errContain)
			}
		})
	}
}

func TestValidate_NewerSchemaVersion(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := &Manifest{
		Version:  "0.5.0",
		Adapters: map[string]AdapterManifest{"test": {}},
	}
	err := m.Validate(".", nil)
	if err == nil {
		t.Fatal("Validate() expected error for newer schema 0.5.0, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported manifest schema version") {
		t.Errorf("Validate() error %q should mention unsupported manifest schema version", err.Error())
	}
}
