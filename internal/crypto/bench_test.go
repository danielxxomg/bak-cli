package crypto

import (
	"fmt"
	"testing"
)

// BenchmarkEncrypt measures AES-256-GCM encryption performance with
// varying payload sizes.
func BenchmarkEncrypt(b *testing.B) {
	sizes := []int{64, 1024, 65536} // 64B, 1KB, 64KB

	for _, size := range sizes {
		b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
			plaintext := make([]byte, size)
			// Fill with non-zero data to avoid compression artifacts.
			for i := range plaintext {
				plaintext[i] = byte(i % 256)
			}
			password := "benchmark-password-12345"

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := Encrypt(plaintext, password)
				if err != nil {
					b.Fatalf("Encrypt: %v", err)
				}
			}
		})
	}
}

// BenchmarkDecrypt measures AES-256-GCM decryption performance with
// varying payload sizes.
func BenchmarkDecrypt(b *testing.B) {
	sizes := []int{64, 1024, 65536}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
			plaintext := make([]byte, size)
			for i := range plaintext {
				plaintext[i] = byte(i % 256)
			}
			password := "benchmark-password-12345"

			// Pre-encrypt the data.
			ciphertext, err := Encrypt(plaintext, password)
			if err != nil {
				b.Fatalf("pre-encrypt: %v", err)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				decrypted, err := Decrypt(ciphertext, password)
				if err != nil {
					b.Fatalf("Decrypt: %v", err)
				}
				if len(decrypted) != size {
					b.Fatalf("decrypted size = %d, want %d", len(decrypted), size)
				}
			}
		})
	}
}

// BenchmarkEncryptDecrypt_Roundtrip measures full encrypt+decrypt cycle.
func BenchmarkEncryptDecrypt_Roundtrip(b *testing.B) {
	plaintext := make([]byte, 4096)
	for i := range plaintext {
		plaintext[i] = byte(i % 256)
	}
	password := "benchmark-password"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		encrypted, err := Encrypt(plaintext, password)
		if err != nil {
			b.Fatalf("Encrypt: %v", err)
		}
		decrypted, err := Decrypt(encrypted, password)
		if err != nil {
			b.Fatalf("Decrypt: %v", err)
		}
		_ = decrypted
	}
}
