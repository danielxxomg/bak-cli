# Design: v0.3.0 — Encryption at Rest & Machine Profiles

## Technical Approach

Add AES-256-GCM encryption to backup archives with Argon2id key derivation, scoped per machine profile. Encryption happens in the cmd layer (push/pull) — Provider interface unchanged. Profiles isolate backup namespaces with independent encryption settings. Config migrates additively from v0.2.0.

## Architecture Decisions

### Decision: Encryption Layer Location

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Encrypt in Provider interface | Breaks backward compat, forces all providers to handle crypto | ❌ Rejected |
| Encrypt in cmd/push.go & cmd/pull.go | Keeps Provider simple, encryption is opt-in per profile | ✅ **Chosen** |

**Rationale**: Provider interface must remain stable. Encryption is a user choice per profile, not a provider responsibility. Cmd layer already handles archive packaging — natural place for encrypt/decrypt.

### Decision: Key Derivation Function

| Option | Tradeoff | Decision |
|--------|----------|----------|
| PBKDF2 | Fast, widely supported, but weaker against GPU attacks | ❌ Rejected |
| scrypt | Good, but less modern than Argon2 | ❌ Rejected |
| Argon2id | Winner of Password Hashing Competition, memory-hard, resistant to GPU/ASIC | ✅ **Chosen** |

**Rationale**: Argon2id with 64MB RAM, 3 iterations, 4 parallelism provides strong protection against brute-force. ~200ms on modern hardware is acceptable for backup operations. Already available via `golang.org/x/crypto/argon2`.

### Decision: Magic Bytes Format

| Option | Tradeoff | Decision |
|--------|----------|----------|
| No magic bytes, detect via manifest | Requires parsing manifest first, slower | ❌ Rejected |
| Custom magic bytes `BAK_ENC\x01` | 7-byte prefix, clear versioning, instant detection | ✅ **Chosen** |

**Rationale**: Magic bytes allow instant encrypted archive detection without parsing. Old v0.2.0 binaries fail at tar.gz parse with clear error. New v0.3.0 checks magic bytes first, falls back to plaintext for backward compat.

### Decision: Password Input Strategy

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Password file only | Insecure (file on disk), bad UX | ❌ Rejected |
| Interactive prompt only | Bad for CI/scripting | ❌ Rejected |
| Stdin prompt + `BAK_ENCRYPTION_PASSWORD` env var | Secure (no disk), works in CI | ✅ **Chosen** |

**Rationale**: Interactive prompt for humans, env var for automation. No password file in v0.3.0 (out of scope). Env var takes precedence when set.

### Decision: Profile Scoping

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Profiles as separate config files | Complex, hard to manage | ❌ Rejected |
| Profiles as map in single config | Simple, atomic updates, easy migration | ✅ **Chosen** |

**Rationale**: Single config with `profiles` map keeps config management simple. Each profile has its own adapters, categories, preset, provider, and encryption settings. Migration is additive — existing providers preserved.

## Data Flow

### Encrypted Push Flow

```
User runs: bak push --profile work-laptop
         ↓
cmd/push.go: Load config, resolve profile
         ↓
backup.BakDir() → find backup ID
         ↓
cloud.TarGzDirectory() → archiveData (plaintext tar.gz)
         ↓
profile.Encryption.Enabled? → Yes
         ↓
crypto.Encrypt(archiveData, password)
  ├─ Generate 32-byte random salt
  ├─ Argon2id(password, salt, 64MB, 3 iter, 4 parallel) → 32-byte key
  ├─ Generate 12-byte random nonce
  ├─ AES-256-GCM encrypt → ciphertext + auth tag
  └─ Prepend magic bytes: BAK_ENC\x01 + salt + nonce + ciphertext
         ↓
provider.Push(encryptedArchive, meta)
         ↓
Update manifest with Encryption struct (algorithm, KDF, salt, nonce)
```

### Decrypted Pull Flow

```
User runs: bak pull --profile work-laptop
         ↓
cmd/pull.go: Load config, resolve profile, get remote ID
         ↓
provider.Pull(remoteID) → archiveData (may be encrypted or plaintext)
         ↓
Check magic bytes: archiveData[:7] == "BAK_ENC\x01"?
         ↓
    ┌────┴────┐
    │         │
   Yes       No
    │         │
    ↓         ↓
Extract salt + nonce    Plaintext archive
    ↓                   (v0.2.0 backward compat)
Prompt for password     ↓
(or use env var)        Extract directly
    ↓
crypto.Decrypt(ciphertext, password, salt, nonce)
  ├─ Argon2id(password, salt, ...) → key
  ├─ AES-256-GCM decrypt → plaintext (or auth failure)
  └─ Return plaintext tar.gz
         ↓
cloud.UntarGz(plaintextArchive, backupPath)
```

### Profile-Scoped Backup Flow

```
User runs: bak backup --profile work-laptop
         ↓
cmd/backup.go: Load config, resolve profile
         ↓
Profile provides:
  - Adapters: [opencode, cursor]
  - Categories: [config, skills]
  - Preset: full
  - Provider: github-gist
  - Encryption: {enabled: true}
         ↓
backup.Engine{
  HomeDir: homeDir,
  BakDir: bakDir,
  Registry: reg,
  Preset: profile.Preset,
  AdapterFilter: "",  // use profile's adapter list
  CustomCategories: profile.Categories,
  BakVersion: Version,
  Verbose: verbose,
}
         ↓
engine.Run() → backup with profile's settings
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/crypto/crypto.go` | Create | AES-256-GCM encrypt/decrypt, Argon2id KDF, magic bytes handling |
| `internal/crypto/crypto_test.go` | Create | Unit tests for encrypt/decrypt round-trip, wrong password, magic bytes |
| `internal/crypto/password.go` | Create | Password input: stdin prompt + env var fallback |
| `internal/crypto/password_test.go` | Create | Tests for password input logic |
| `internal/manifest/manifest.go` | Modify | Add `Encryption` struct field to Manifest |
| `internal/config/config.go` | Modify | Add `ProfileConfig` struct, `Profiles` map, `migrateV020()` function |
| `internal/config/config_test.go` | Modify | Add tests for v0.2.0→v0.3.0 migration |
| `cmd/push.go` | Modify | Add `--profile` flag, encrypt archive before push when profile has encryption |
| `cmd/pull.go` | Modify | Add `--profile` flag, decrypt archive after pull when magic bytes detected |
| `cmd/backup.go` | Modify | Add `--profile` flag, scope backup to profile's adapters/categories/preset |
| `cmd/profile.go` | Create | Profile CRUD commands: create, list, show, delete |
| `cmd/profile_test.go` | Create | Tests for profile commands |

## Interfaces / Contracts

### Crypto Package

```go
package crypto

// Encrypt encrypts plaintext with AES-256-GCM using Argon2id key derivation.
// Returns: magic bytes + salt (32 bytes) + nonce (12 bytes) + ciphertext.
func Encrypt(plaintext []byte, password string) ([]byte, error)

// Decrypt decrypts ciphertext encrypted by Encrypt.
// Input format: magic bytes + salt + nonce + ciphertext.
// Returns error on wrong password (GCM auth tag failure).
func Decrypt(archive []byte, password string) ([]byte, error)

// IsEncrypted checks if data starts with magic bytes.
func IsEncrypted(data []byte) bool

// DeriveKey derives a 32-byte key from password using Argon2id.
// Params: 64MB RAM, 3 iterations, 4 parallelism.
func DeriveKey(password string, salt []byte) []byte
```

### Magic Bytes Format

```
Offset  Length  Content
0       7       "BAK_ENC\x01" (magic bytes)
7       32      salt (Argon2id salt)
39      12      nonce (AES-GCM nonce)
51      ...     ciphertext (AES-GCM encrypted data + auth tag)
```

### Manifest Encryption Struct

```go
// Encryption holds encryption metadata for an encrypted backup.
type Encryption struct {
    Algorithm string `json:"algorithm"`          // "AES-256-GCM"
    KDF       string `json:"kdf"`                // "Argon2id"
    Salt      string `json:"salt"`               // hex-encoded salt
    Nonce     string `json:"nonce"`              // hex-encoded nonce
    Iterations int   `json:"iterations"`         // Argon2id iterations (3)
    MemoryKB  int    `json:"memory_kb"`          // Argon2id memory (65536)
    Parallelism int  `json:"parallelism"`        // Argon2id parallelism (4)
}
```

### ProfileConfig Struct

```go
// ProfileConfig holds settings for a named machine profile.
type ProfileConfig struct {
    Adapters   []string           `json:"adapters,omitempty"`   // adapter names
    Categories []string           `json:"categories,omitempty"` // backup categories
    Preset     string             `json:"preset,omitempty"`     // quick, full, skills
    Provider   string             `json:"provider,omitempty"`   // provider name
    Encryption EncryptionConfig   `json:"encryption,omitempty"` // encryption settings
}

// EncryptionConfig holds per-profile encryption settings.
type EncryptionConfig struct {
    Enabled bool `json:"enabled"` // true = encrypt backups for this profile
}
```

### Config Migration

```go
// migrateV020 transforms a v0.2.0 config into v0.3.0 format.
// Adds empty profiles map, bumps schema_version to "0.3.0",
// writes config.json.v020.bak before overwriting.
func migrateV020(cfg *Config, original []byte) error
```

### Password Input

```go
// GetPassword returns the encryption password.
// Priority: BAK_ENCRYPTION_PASSWORD env var, then interactive stdin prompt.
// Returns error if stdin is not a terminal and env var is not set.
func GetPassword(prompt string) (string, error)
```

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | `internal/crypto` encrypt/decrypt round-trip | Table-driven tests with known passwords, verify ciphertext differs with same password (unique salts) |
| Unit | Wrong password detection | Encrypt with "correct", decrypt with "wrong", verify GCM auth failure error |
| Unit | Magic bytes detection | Test `IsEncrypted()` with encrypted and plaintext data |
| Unit | Argon2id key derivation | Verify deterministic output for same password+salt, verify distinct salts produce distinct keys |
| Unit | Password input | Mock stdin, test env var precedence, test error when no terminal and no env var |
| Unit | Config migration v0.2.0→v0.3.0 | Load v0.2.0 config, verify profiles map added, schema_version bumped, .v020.bak created, providers preserved |
| Unit | Profile CRUD | Create/list/show/delete profiles, verify validation (provider exists, token configured) |
| Integration | Push/pull round-trip with encryption | Create backup, push with encrypted profile, pull with same profile, verify decryption and extraction |
| Integration | Backward compat: pull plaintext archive | Push unencrypted archive (v0.2.0 style), pull with v0.3.0, verify extraction works |
| Integration | Profile-scoped backup | Create profile with specific adapters/categories, run backup, verify only profile's scope is backed up |

## Migration / Rollout

### Config Migration (Automatic)

1. User runs any `bak` command with v0.3.0 binary
2. `config.LoadPath()` detects v0.2.0 (schema_version == "0.2.0", no profiles map)
3. `migrateV020()` executes:
   - Writes `config.json.v020.bak` (backup of original)
   - Adds empty `profiles` map
   - Bumps `schema_version` to "0.3.0"
   - Preserves all existing providers
   - Saves migrated config
4. User continues with v0.3.0 features

### Rollback Plan

1. **Config rollback**: Restore `config.json.v020.bak` → `config.json`
2. **Encrypted backups**: Cannot decrypt without password; user must re-push unencrypted from v0.2.0 binary
3. **Binary rollback**: `go install github.com/danielxxomg/bak-cli@v0.2.0` reverts CLI

### Backward Compatibility

- **v0.3.0 pulling v0.2.0 archives**: Magic bytes absent → treat as plaintext, extract normally
- **v0.2.0 pulling v0.3.0 encrypted archives**: tar.gz parse fails on magic bytes → clear error "upgrade required"
- **v0.2.0 pulling v0.3.0 unencrypted archives**: Works normally (no magic bytes, plaintext)

## Sequence Diagrams

### Encrypt Push Sequence

```
User            cmd/push.go         crypto          provider
 │                  │                  │                │
 │ bak push         │                  │                │
 │ --profile work   │                  │                │
 ├─────────────────>│                  │                │
 │                  │ Load config      │                │
 │                  │ Resolve profile  │                │
 │                  │                  │                │
 │                  │ Tar.gz backup    │                │
 │                  │─────────────────>│                │
 │                  │  plaintext       │                │
 │                  │<─────────────────│                │
 │                  │                  │                │
 │                  │ Encryption on?   │                │
 │                  │─────────────────>│                │
 │                  │  yes             │                │
 │                  │<─────────────────│                │
 │                  │                  │                │
 │ Password?        │                  │                │
 │<─────────────────│                  │                │
 │ mypassword       │                  │                │
 ├─────────────────>│                  │                │
 │                  │                  │                │
 │                  │ Encrypt(plaintext, password)      │
 │                  │─────────────────>│                │
 │                  │  salt+nonce+     │                │
 │                  │  ciphertext      │                │
 │                  │<─────────────────│                │
 │                  │                  │                │
 │                  │ Push(encrypted, meta)             │
 │                  │─────────────────────────────────>│
 │                  │                  │                │ id
 │                  │<─────────────────────────────────│
 │                  │                  │                │
 │ ✅ Pushed        │                  │                │
 │<─────────────────│                  │                │
```

### Decrypt Pull Sequence

```
User            cmd/pull.go         crypto          provider
 │                  │                  │                │
 │ bak pull         │                  │                │
 │ --profile work   │                  │                │
 ├─────────────────>│                  │                │
 │                  │ Load config      │                │
 │                  │ Resolve profile  │                │
 │                  │                  │                │
 │                  │ Pull(remoteID)   │                │
 │                  │─────────────────────────────────>│
 │                  │  archiveData     │                │
 │                  │<─────────────────────────────────│
 │                  │                  │                │
 │                  │ IsEncrypted?     │                │
 │                  │─────────────────>│                │
 │                  │  yes             │                │
 │                  │<─────────────────│                │
 │                  │                  │                │
 │ Password?        │                  │                │
 │<─────────────────│                  │                │
 │ mypassword       │                  │                │
 ├─────────────────>│                  │                │
 │                  │                  │                │
 │                  │ Decrypt(archive, password)        │
 │                  │─────────────────>│                │
 │                  │  plaintext       │                │
 │                  │<─────────────────│                │
 │                  │                  │                │
 │                  │ Untar.gz(plaintext)               │
 │                  │ Extract files    │                │
 │                  │                  │                │
 │ ✅ Pulled        │                  │                │
 │<─────────────────│                  │                │
```

### Profile-Scoped Backup Sequence

```
User            cmd/backup.go       backup.Engine     adapters
 │                  │                    │                │
 │ bak backup       │                    │                │
 │ --profile work   │                    │                │
 ├─────────────────>│                    │                │
 │                  │ Load config        │                │
 │                  │ Resolve profile    │                │
 │                  │                    │                │
 │                  │ Engine{            │                │
 │                  │   Preset: profile.Preset,          │
 │                  │   CustomCategories: profile.Categories,
 │                  │   AdapterFilter: "",               │
 │                  │ }                  │                │
 │                  │───────────────────>│                │
 │                  │                    │                │
 │                  │                    │ Resolve preset │
 │                  │                    │ (or use custom)│
 │                  │                    │────────────────>
 │                  │                    │                │
 │                  │                    │ DetectAll()    │
 │                  │                    │────────────────>
 │                  │                    │ [opencode,     │
 │                  │                    │  cursor, ...]  │
 │                  │                    │<────────────────
 │                  │                    │                │
 │                  │                    │ Filter by      │
 │                  │                    │ profile.Adapters
 │                  │                    │                │
 │                  │                    │ For each adapter:
 │                  │                    │   ListItems(categories)
 │                  │                    │   Backup()     │
 │                  │                    │────────────────>
 │                  │                    │                │
 │                  │                    │ Save manifest  │
 │                  │                    │                │
 │                  │ Result{ID, Files}  │                │
 │                  │<───────────────────│                │
 │                  │                    │                │
 │ ✅ Backup        │                    │                │
 │<─────────────────│                    │                │
```

## Open Questions

- [ ] **None** — all technical decisions resolved in proposal and specs.
