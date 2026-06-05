package manifest

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
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

func TestSaveLoadRoundTrip(t *testing.T) {
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

func TestValidate_HappyPath(t *testing.T) {
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

	if err := m.Validate(dir); err != nil {
		t.Errorf("Validate: unexpected error: %v", err)
	}
}

func TestValidate_HashMismatch(t *testing.T) {
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

	err := m.Validate(dir)
	if err == nil {
		t.Error("Validate: expected hash mismatch error, got nil")
	}
}

func TestValidate_EmptyVersion(t *testing.T) {
	m := &Manifest{Adapters: map[string]AdapterManifest{"x": {}}}
	if err := m.Validate("."); err == nil {
		t.Error("expected error for empty version")
	}
}

func TestValidate_NoAdapters(t *testing.T) {
	m := &Manifest{Version: "0.1.0"}
	if err := m.Validate("."); err == nil {
		t.Error("expected error for no adapters")
	}
}

func TestLoad_NonExistent(t *testing.T) {
	_, err := Load(t.TempDir())
	if err == nil {
		t.Error("expected error for missing manifest.json")
	}
}

// ---- Encryption struct tests ----

func TestSetEncryption(t *testing.T) {
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

func TestManifest_JSON_EncryptionRoundTrip(t *testing.T) {
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

func TestManifest_JSON_PlaintextOmitsEncryption(t *testing.T) {
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
	if bytesContains(raw, `"encryption"`) {
		t.Error("raw JSON contains 'encryption' key for plaintext manifest")
	}
}

func bytesContains(data []byte, substr string) bool {
	for i := 0; i <= len(data)-len(substr); i++ {
		if string(data[i:i+len(substr)]) == substr {
			return true
		}
	}
	return false
}

func TestSetEncryption_NilSaltNonce(t *testing.T) {
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
