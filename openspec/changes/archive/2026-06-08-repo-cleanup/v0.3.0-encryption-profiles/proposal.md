# Proposal: v0.3.0 — Encryption at Rest & Machine Profiles

## Intent

Cloud backups are currently plaintext tarballs in GitHub Gists. Anyone with gist access reads your full AI config. Users with multiple machines (work laptop, home PC) also lack isolated backup namespaces — one gist holds everything, making it unclear which machine produced what.

v0.3.0 solves both: AES-256-GCM encryption for push/pull, and named profiles to scope backups per machine.

## Scope

### In Scope
- `internal/crypto` package — encrypt/decrypt with AES-256-GCM + Argon2id KDF
- Manifest `Encryption` field (algorithm, KDF params, salt, nonce) — manifest stays plaintext
- `bak push`/`bak pull` encryption wiring in cmd layer (NOT in Provider interface)
- `bak profile create|list|show|delete` commands
- `--profile` flag on `bak backup`, `bak push`, `bak pull`
- Config migration v0.2.0 → v0.3.0 (add `profiles` map, bump `schema_version`)
- Backward compat: old `bak pull` detects encrypted archives and errors gracefully

### Out of Scope
- Key derivation from hardware tokens or OS keychains
- Password rotation or re-encryption of existing backups
- Profile-level provider isolation (profiles share providers; encryption is per-profile)
- Merge restore mode

## Capabilities

### New Capabilities
- `encryption-at-rest`: AES-256-GCM encrypt/decrypt, Argon2id KDF, password prompt, manifest encryption metadata
- `machine-profiles`: named profile CRUD, profile-scoped backups, `--profile` flag routing, config migration v0.3.0

### Modified Capabilities
- `cloud-sync`: push/pull must encrypt before upload and decrypt after download when profile has encryption enabled
- `manifest`: add optional `Encryption` struct to manifest schema
- `backup-engine`: accept profile context to scope backup output directory

## Approach

**Encryption:** New `internal/crypto` package. `Encrypt(plaintext, password) → (ciphertext, salt, nonce, error)`. `Decrypt(ciphertext, password, salt, nonce) → (plaintext, error)`. Argon2id: 64MB RAM, 3 iterations, 4 parallelism, 32-byte salt. Encryption happens in `cmd/push.go` (after tar.gz, before provider.Push) and `cmd/pull.go` (after provider.Pull, before extract). Provider interface unchanged.

**Password input:** Interactive stdin prompt (primary). `BAK_ENCRYPTION_PASSWORD` env var (CI/scripting). No password file in v0.3.0.

**Forgot password:** Data is unrecoverable. `bak pull` MUST print clear error: "wrong password or corrupted archive — encryption is unrecoverable without the correct password." No password reset, no recovery key.

**Profiles:** `ProfileConfig` struct in config: `{adapters, categories, preset, provider, encryption}`. Profile creation validates provider exists and token is configured. Encryption is per-profile (each profile can independently enable/disable encryption with its own password).

**Config migration:** Additive. `migrateV020()` adds empty `profiles` map, bumps `schema_version` to `"0.3.0"`, writes `config.json.v020.bak`. Existing providers preserved.

**Backward compat:** Encrypted archives include a magic byte prefix (`BAK_ENC\x01`) before the ciphertext. Old `bak pull` (v0.2.0) attempting to extract an encrypted archive will fail at tar.gz parse with a clear error. New `bak pull` checks for magic bytes first; if absent, processes as plaintext (backward compat with unencrypted v0.2.0 archives).

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/crypto/` | New | AES-256-GCM encrypt/decrypt, Argon2id KDF |
| `internal/manifest/manifest.go` | Modified | Add `Encryption` struct field |
| `internal/config/config.go` | Modified | Add `ProfileConfig`, `Profiles` map, migration |
| `cmd/push.go` | Modified | Encrypt archive before push |
| `cmd/pull.go` | Modified | Decrypt archive after pull |
| `cmd/profile.go` | New | Profile CRUD commands |
| `cmd/backup.go` | Modified | Accept `--profile` flag |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Wrong password corrupts restore | Med | Magic bytes + GCM auth tag detect wrong password before any file write |
| Argon2id too slow on low-end machines | Low | 64MB/3 iter is ~200ms on modern hardware; document minimum specs |
| Config migration corrupts existing config | Low | Backup `.v020.bak` before migration; additive-only changes |
| Encrypted archives break `bak list` | Low | Manifest stays plaintext; only archive payload is encrypted |

## Rollback Plan

1. Config: restore `config.json.v020.bak` → `config.json`
2. Encrypted cloud backups: cannot decrypt without password; user must re-push unencrypted from v0.2.0 binary
3. Binary: `go install github.com/danielxxomg/bak-cli@v0.2.0` reverts CLI

## Dependencies

- `golang.org/x/crypto/argon2` (already in go.mod)
- `crypto/aes`, `crypto/cipher` (stdlib)

## Success Criteria

- [ ] `bak push` with encrypted profile produces archive that `bak pull` decrypts correctly (round-trip)
- [ ] Wrong password returns clear error, writes zero files
- [ ] `bak pull` on v0.2.0 binary with encrypted archive shows "upgrade required" error
- [ ] `bak profile create work-laptop` validates provider and persists config
- [ ] Config migration v0.2.0→v0.3.0 preserves all existing provider settings
- [ ] `go test ./...` passes with >80% coverage on new code
