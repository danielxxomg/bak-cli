package crypto

import (
	"bytes"
	"testing"
)

// ---- Round-trip tests ----

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	tests := []struct {
		name     string
		password string
		data     []byte
	}{
		{
			name:     "short text",
			password: "correct-horse-battery-staple",
			data:     []byte("hello world"),
		},
		{
			name:     "binary data",
			password: "p@ssw0rd!",
			data:     []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD},
		},
		{
			name:     "empty plaintext",
			password: "secret",
			data:     []byte{},
		},
		{
			name:     "large payload (~1KB)",
			password: "large-secret",
			data:     bytes.Repeat([]byte("A"), 1024),
		},
		{
			name:     "unicode password",
			password: "contraseña🔐",
			data:     []byte("protegido"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encrypted, err := Encrypt(tt.data, tt.password)
			if err != nil {
				t.Fatalf("Encrypt: unexpected error: %v", err)
			}

			// Verify magic bytes prefix.
			if !IsEncrypted(encrypted) {
				t.Error("IsEncrypted returned false for freshly encrypted data")
			}

			// Verify format: magic(7) + salt(32) + nonce(12) + ciphertext (at least GCM tag)
			minLen := 7 + 32 + 12 + 16 // magic + salt + nonce + GCM tag
			if len(encrypted) < minLen+len(tt.data) {
				t.Errorf("encrypted too short: got %d bytes, want >= %d", len(encrypted), minLen+len(tt.data))
			}

			decrypted, err := Decrypt(encrypted, tt.password)
			if err != nil {
				t.Fatalf("Decrypt: unexpected error: %v", err)
			}

			if !bytes.Equal(decrypted, tt.data) {
				t.Errorf("round-trip mismatch: got %q, want %q", decrypted, tt.data)
			}
		})
	}
}

// ---- Wrong password tests ----

func TestDecrypt_WrongPassword(t *testing.T) {
	plaintext := []byte("sensitive data")
	password := "correct-password"

	encrypted, err := Encrypt(plaintext, password)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	decrypted, err := Decrypt(encrypted, "wrong-password")
	if err == nil {
		t.Fatal("Decrypt: expected error for wrong password, got nil")
	}
	if decrypted != nil {
		t.Errorf("Decrypt: expected nil plaintext on error, got %v", decrypted)
	}
}

// ---- Distinct salts ----

func TestEncrypt_DistinctSalts(t *testing.T) {
	password := "same-password"
	data := []byte("same data")

	enc1, err := Encrypt(data, password)
	if err != nil {
		t.Fatalf("Encrypt #1: %v", err)
	}

	enc2, err := Encrypt(data, password)
	if err != nil {
		t.Fatalf("Encrypt #2: %v", err)
	}

	// Salt is at bytes 7-39.
	salt1 := enc1[7:39]
	salt2 := enc2[7:39]

	if bytes.Equal(salt1, salt2) {
		t.Error("DistinctSalts: two encryptions produced the same salt — non-deterministic salt required")
	}

	// Both must decrypt successfully with the same password.
	d1, err := Decrypt(enc1, password)
	if err != nil {
		t.Fatalf("Decrypt #1: %v", err)
	}
	d2, err := Decrypt(enc2, password)
	if err != nil {
		t.Fatalf("Decrypt #2: %v", err)
	}
	if !bytes.Equal(d1, data) || !bytes.Equal(d2, data) {
		t.Error("DistinctSalts: decryption mismatch")
	}
}

// ---- IsEncrypted ----

func TestIsEncrypted(t *testing.T) {
	plaintext := []byte("hello")

	encrypted, err := Encrypt(plaintext, "pw")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	if !IsEncrypted(encrypted) {
		t.Error("IsEncrypted: encrypted data not detected")
	}

	if IsEncrypted(plaintext) {
		t.Error("IsEncrypted: plaintext falsely detected as encrypted")
	}

	if IsEncrypted([]byte{}) {
		t.Error("IsEncrypted: empty slice falsely detected as encrypted")
	}

	if IsEncrypted(nil) {
		t.Error("IsEncrypted: nil slice falsely detected as encrypted")
	}

	// Slightly corrupted prefix — should fail detection.
	corrupted := make([]byte, len(encrypted))
	copy(corrupted, encrypted)
	corrupted[0] = 'X'
	if IsEncrypted(corrupted) {
		t.Error("IsEncrypted: corrupted magic bytes falsely detected as encrypted")
	}
}

// ---- deriveKey ----

func TestDeriveKey_Deterministic(t *testing.T) {
	password := "my-secret"
	salt := make([]byte, 32)
	for i := range salt {
		salt[i] = byte(i)
	}

	key1 := deriveKey(password, salt)
	key2 := deriveKey(password, salt)

	if !bytes.Equal(key1, key2) {
		t.Error("deriveKey: same password+salt produced different keys")
	}

	if len(key1) != 32 {
		t.Errorf("deriveKey: key length = %d, want 32", len(key1))
	}
}

func TestDeriveKey_DistinctInputs(t *testing.T) {
	salt1 := bytes.Repeat([]byte{0xAA}, 32)
	salt2 := bytes.Repeat([]byte{0xBB}, 32)

	key1 := deriveKey("pw", salt1)
	key2 := deriveKey("pw", salt2)

	if bytes.Equal(key1, key2) {
		t.Error("deriveKey: different salts produced identical keys")
	}

	key3 := deriveKey("other-pw", salt1)
	if bytes.Equal(key1, key3) {
		t.Error("deriveKey: different passwords produced identical keys")
	}
}

// ---- Edge cases ----

func TestDecrypt_CorruptedArchive(t *testing.T) {
	encrypted, err := Encrypt([]byte("data"), "pw")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	// Tamper with the ciphertext (after header).
	tampered := make([]byte, len(encrypted))
	copy(tampered, encrypted)
	tampered[len(tampered)-1] ^= 0xFF // flip last byte

	_, err = Decrypt(tampered, "pw")
	if err == nil {
		t.Fatal("Decrypt: expected error for tampered archive, got nil")
	}
}

func TestDecrypt_TooShort(t *testing.T) {
	// Not enough bytes for magic + salt + nonce + GCM tag.
	tooShort := []byte("BAK_ENC\x01" + string(make([]byte, 32+12)))
	_, err := Decrypt(tooShort, "pw")
	if err == nil {
		t.Fatal("Decrypt: expected error for too-short archive, got nil")
	}
}

func TestDecrypt_WrongMagic(t *testing.T) {
	// Simulate a plaintext tar.gz — no magic bytes.
	plaintext := []byte{0x1f, 0x8b, 0x08} // gzip header
	_, err := Decrypt(plaintext, "pw")
	if err == nil {
		t.Fatal("Decrypt: expected error for missing magic bytes, got nil")
	}
}

func TestDecrypt_WrongMagicPrefix(t *testing.T) {
	// Magic bytes present but version byte wrong.
	data := append([]byte("BAK_ENC\x02"), make([]byte, 32+12+16)...)
	_, err := Decrypt(data, "pw")
	if err == nil {
		t.Fatal("Decrypt: expected error for wrong magic version, got nil")
	}
}

func TestEncrypt_EmptyPassword(t *testing.T) {
	// Empty password should still work (Argon2id accepts empty input).
	encrypted, err := Encrypt([]byte("data"), "")
	if err != nil {
		t.Fatalf("Encrypt with empty password: %v", err)
	}
	decrypted, err := Decrypt(encrypted, "")
	if err != nil {
		t.Fatalf("Decrypt with empty password: %v", err)
	}
	if !bytes.Equal(decrypted, []byte("data")) {
		t.Error("empty password round-trip mismatch")
	}
}

func TestEncrypt_IdenticalInputs_DifferentOutputs(t *testing.T) {
	// Same password, same plaintext → different ciphertexts (unique salt + nonce).
	enc1, _ := Encrypt([]byte("hello"), "pw")
	enc2, _ := Encrypt([]byte("hello"), "pw")

	if bytes.Equal(enc1, enc2) {
		t.Error("identical inputs produced identical ciphertext — must be unique per encryption")
	}
}
