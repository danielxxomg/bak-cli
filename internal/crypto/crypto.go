// Package crypto provides AES-256-GCM encryption and Argon2id key derivation
// for backup archive encryption at rest.
//
// Dependency: golang.org/x/crypto/argon2 — required for Argon2id key derivation
// (memory-hard KDF, resistant to GPU/ASIC attacks, not available in stdlib).
package crypto

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"

	"golang.org/x/crypto/argon2"
)

// Magic bytes prefix for encrypted archives ("BAK_ENC" + version byte).
const magicBytes = "BAK_ENC\x01"

const (
	saltLen     = 32 // bytes
	nonceLen    = 12 // bytes (AES-GCM standard nonce)
	keyLen      = 32 // bytes (AES-256)
	tagLen      = 16 // bytes (GCM authentication tag)
	memoryKiB   = 65536
	iterations  = 3
	parallelism = 4
)

// Exported constants for archive layout offsets and KDF parameters.
const (
	MagicLen    = 8  // len(magicBytes)
	SaltLen     = saltLen
	NonceLen    = nonceLen
	ArgonMemory = memoryKiB
	ArgonIters  = iterations
	ArgonPar    = parallelism
)

// Encrypt encrypts plaintext with AES-256-GCM using a key derived from password
// via Argon2id. Returns a byte slice with format:
//
//	magic(8) + salt(32) + nonce(12) + ciphertext + gcm_tag(16)
func Encrypt(plaintext []byte, password string) ([]byte, error) {
	// Generate random salt.
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("generate salt: %w", err)
	}

	// Derive key.
	key := deriveKey(password, salt)

	// Create AES cipher.
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	// Create GCM.
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create gcm: %w", err)
	}

	// Generate random nonce.
	nonce := make([]byte, nonceLen)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}

	// Encrypt. GCM appends the authentication tag to the ciphertext.
	ciphertext := aesgcm.Seal(nil, nonce, plaintext, nil)

	// Assemble: magic + salt + nonce + ciphertext.
	out := make([]byte, 0, len(magicBytes)+saltLen+nonceLen+len(ciphertext))
	out = append(out, []byte(magicBytes)...)
	out = append(out, salt...)
	out = append(out, nonce...)
	out = append(out, ciphertext...)

	return out, nil
}

// Decrypt decrypts an archive produced by Encrypt. It validates the magic bytes,
// extracts salt and nonce, derives the key, and decrypts with AES-256-GCM.
func Decrypt(archive []byte, password string) ([]byte, error) {
	if len(archive) < len(magicBytes)+saltLen+nonceLen+tagLen {
		return nil, fmt.Errorf("decrypt: archive too short (%d bytes)", len(archive))
	}

	// Validate magic bytes.
	if !IsEncrypted(archive) {
		return nil, fmt.Errorf("decrypt: invalid magic bytes")
	}

	// Extract components.
	off := len(magicBytes)
	salt := archive[off : off+saltLen]
	off += saltLen
	nonce := archive[off : off+nonceLen]
	off += nonceLen
	ciphertext := archive[off:]

	// Derive key.
	key := deriveKey(password, salt)

	// Create AES cipher.
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	// Create GCM.
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create gcm: %w", err)
	}

	// Decrypt. GCM.Open returns an error if the authentication tag is invalid.
	plaintext, err := aesgcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt: wrong password or corrupted archive: %w", err)
	}

	return plaintext, nil
}

// IsEncrypted reports whether data starts with the magic bytes prefix.
func IsEncrypted(data []byte) bool {
	return bytes.HasPrefix(data, []byte(magicBytes))
}

// deriveKey derives a 32-byte key from the password and salt using Argon2id
// with 64MB RAM, 3 iterations, and 4-way parallelism.
func deriveKey(password string, salt []byte) []byte {
	return argon2.IDKey([]byte(password), salt, iterations, memoryKiB, parallelism, keyLen)
}
