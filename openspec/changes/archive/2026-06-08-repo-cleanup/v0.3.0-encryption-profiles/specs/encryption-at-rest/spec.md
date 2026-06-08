# encryption-at-rest Specification

## Purpose
Defines AES-256-GCM encryption and Argon2id key derivation for backup archives.

## Requirements

### Requirement: AES-256-GCM Encrypt/Decrypt
The system MUST encrypt tar.gz archives with AES-256-GCM before upload and decrypt after download when profile encryption is enabled.

#### Scenario: Push encrypts
- GIVEN a profile with encryption enabled
- WHEN `bak push` runs
- THEN the archive is encrypted before provider.Push

#### Scenario: Pull decrypts
- GIVEN an encrypted archive and correct password
- WHEN `bak pull` runs
- THEN the archive is decrypted before extraction

### Requirement: Argon2id KDF
The system MUST derive the encryption key using Argon2id with 64MB RAM, 3 iterations, 4 parallelism, and a 32-byte random salt.

#### Scenario: Distinct salts
- GIVEN two pushes with the same password
- WHEN archives are encrypted
- THEN each archive uses a unique salt

### Requirement: Password Input
The system MUST accept passwords via interactive stdin prompt and MAY use `BAK_ENCRYPTION_PASSWORD`.

#### Scenario: Interactive prompt
- GIVEN no env var is set
- WHEN `bak push` runs on an encrypted profile
- THEN the system prompts for password on stdin

### Requirement: Wrong Password
The system MUST detect wrong passwords via the GCM authentication tag and MUST NOT write any files.

#### Scenario: Bad password
- GIVEN an encrypted archive
- WHEN `bak pull` runs with an incorrect password
- THEN error "wrong password or corrupted archive" prints
- AND zero files are restored

### Requirement: Magic Bytes
The system MUST prefix encrypted archives with `BAK_ENC\x01`.

#### Scenario: Plaintext backward compat
- GIVEN a v0.2.0 unencrypted archive
- WHEN `bak pull` runs on v0.3.0
- THEN the archive extracts normally without magic bytes

#### Scenario: Old binary on encrypted archive
- GIVEN an encrypted archive
- WHEN v0.2.0 `bak pull` attempts extraction
- THEN a clear "upgrade required" error is shown
