package cmd

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"

	"github.com/danielxxomg/bak-cli/internal/cloud"
	"github.com/danielxxomg/bak-cli/internal/crypto"
)

func TestPushEncrypt_RoundTrip(t *testing.T) {
	tests := []struct {
		name     string
		password string
		files    map[string]string
	}{
		{
			name:     "single file",
			password: "test-password-123",
			files: map[string]string{
				"config.json": `{"key":"value"}`,
			},
		},
		{
			name:     "multiple files",
			password: "another-secret!",
			files: map[string]string{
				"config.json":    `{"schema_version":"0.3.0"}`,
				"manifest.json":  `{"version":"0.3.0","id":"abc"}`,
				"data/settings":  `enabled=true`,
			},
		},
		{
			name:     "empty directory",
			password: "empty-pass",
			files:    map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 1. Create a temp backup directory with files.
			backupDir := t.TempDir()
			for relPath, content := range tt.files {
				fullPath := filepath.Join(backupDir, relPath)
				if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
					t.Fatal(err)
				}
			}

			// 2. Package as tar.gz (same as push command).
			archiveData, err := cloud.TarGzDirectory(backupDir)
			if err != nil {
				t.Fatalf("TarGzDirectory: %v", err)
			}

			// 3. Decode base64 (same as push command does before encryption).
			rawArchive, err := base64.StdEncoding.DecodeString(archiveData)
			if err != nil {
				t.Fatalf("decode base64: %v", err)
			}

			// 4. Encrypt.
			encrypted, err := crypto.Encrypt(rawArchive, tt.password)
			if err != nil {
				t.Fatalf("Encrypt: %v", err)
			}

			// 5. Verify magic bytes are present.
			if !crypto.IsEncrypted(encrypted) {
				t.Error("IsEncrypted returned false after encryption")
			}

			// 6. Verify encrypted data is different from raw.
			if len(encrypted) <= len(rawArchive) && len(tt.files) > 0 {
				t.Error("encrypted data should be larger than plaintext (header overhead)")
			}

			// 7. Decrypt (simulating pull flow).
			decrypted, err := crypto.Decrypt(encrypted, tt.password)
			if err != nil {
				t.Fatalf("Decrypt: %v", err)
			}

			// 8. Re-encode to base64 for UntarGz.
			reEncoded := base64.StdEncoding.EncodeToString(decrypted)

			// 9. Extract to verify round-trip.
			extractDir := t.TempDir()
			if err := cloud.UntarGz(reEncoded, extractDir); err != nil {
				t.Fatalf("UntarGz: %v", err)
			}

			// 10. Verify files match.
			for relPath, expectedContent := range tt.files {
				fullPath := filepath.Join(extractDir, relPath)
				got, err := os.ReadFile(fullPath)
				if err != nil {
					t.Errorf("missing file after round-trip: %s: %v", relPath, err)
					continue
				}
				if string(got) != expectedContent {
					t.Errorf("file %q: content mismatch\n got: %q\nwant: %q", relPath, string(got), expectedContent)
				}
			}
		})
	}
}

func TestPushEncrypt_WrongPassword(t *testing.T) {
	backupDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(backupDir, "test.txt"), []byte("secret"), 0644); err != nil {
		t.Fatal(err)
	}

	archiveData, err := cloud.TarGzDirectory(backupDir)
	if err != nil {
		t.Fatalf("TarGzDirectory: %v", err)
	}

	rawArchive, _ := base64.StdEncoding.DecodeString(archiveData)

	encrypted, err := crypto.Encrypt(rawArchive, "correct-password")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	// Decrypt with wrong password should fail.
	_, err = crypto.Decrypt(encrypted, "wrong-password")
	if err == nil {
		t.Fatal("Decrypt with wrong password should fail")
	}
}

func TestPushEncrypt_NonEncryptedDetection(t *testing.T) {
	backupDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(backupDir, "plain.txt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	archiveData, err := cloud.TarGzDirectory(backupDir)
	if err != nil {
		t.Fatalf("TarGzDirectory: %v", err)
	}

	rawArchive, _ := base64.StdEncoding.DecodeString(archiveData)

	// Plain tar.gz should NOT be detected as encrypted.
	if crypto.IsEncrypted(rawArchive) {
		t.Error("plain tar.gz should not be detected as encrypted")
	}
}

func TestPushEncrypt_MagicBytesIntegrity(t *testing.T) {
	backupDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(backupDir, "data.txt"), []byte("integrity check"), 0644); err != nil {
		t.Fatal(err)
	}

	archiveData, _ := cloud.TarGzDirectory(backupDir)
	rawArchive, _ := base64.StdEncoding.DecodeString(archiveData)

	encrypted, err := crypto.Encrypt(rawArchive, "test-pw")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	// Verify magic bytes at the beginning.
	if len(encrypted) < crypto.MagicLen {
		t.Fatal("encrypted data too short for magic bytes")
	}

	// Corrupted magic should fail decryption.
	corrupted := make([]byte, len(encrypted))
	copy(corrupted, encrypted)
	corrupted[0] = 'X'

	_, err = crypto.Decrypt(corrupted, "test-pw")
	if err == nil {
		t.Fatal("Decrypt with corrupted magic should fail")
	}
}
