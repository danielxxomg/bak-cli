# Tasks: v0.3.0 — Encryption at Rest & Machine Profiles

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~1400–1600 (new + modified) |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | PR 1 (Phases 1–2) → PR 2 (Phases 3–4) → PR 3 (Phases 5–7) |
| Delivery strategy | ask-on-risk |
| Chain strategy | stacked-to-main |

Decision needed before apply: Yes
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: High

### Suggested Work Units

| Unit | Goal | Likely PR | Notes |
|------|------|-----------|-------|
| 1 | Crypto package + manifest encryption struct | PR 1 | Foundation; independently testable; ~420 lines |
| 2 | Push/pull encryption wiring + config migration v0.2→v0.3 | PR 2 | End-to-end encrypt flow; ~420 lines; depends on PR 1 |
| 3 | Profile CRUD + engine awareness + integration tests + docs | PR 3 | Profiles + final wiring; ~720 lines; depends on PR 2 |

## Phase 1: Crypto Package (Foundation)

- [x] 1.1 Create `internal/crypto/crypto.go` with `Encrypt(plaintext, password) → ([]byte, error)` — generates 32-byte random salt, derives key via Argon2id (64MB/3iter/4parallel), generates 12-byte nonce, encrypts with AES-256-GCM, returns `BAK_ENC\x01 + salt + nonce + ciphertext`
- [x] 1.2 Add `Decrypt(archive, password) → ([]byte, error)` to `internal/crypto/crypto.go` — validates magic bytes, extracts salt+nonce, derives key, decrypts with AES-256-GCM, returns plaintext or auth-failure error
- [x] 1.3 Add `IsEncrypted(data) → bool` and `DeriveKey(password, salt) → []byte` to `internal/crypto/crypto.go`
- [x] 1.4 Create `internal/crypto/crypto_test.go` — table-driven tests: round-trip encrypt/decrypt, wrong password returns auth error, distinct salts produce distinct ciphertext, `IsEncrypted` detects magic bytes, `IsEncrypted` returns false for plaintext, empty input errors
- [x] 1.5 Verify: `go test ./internal/crypto/...` passes with >80% coverage

## Phase 2: Manifest Encryption Struct

- [x] 2.1 Add `Encryption` struct to `internal/manifest/manifest.go` with fields: Algorithm, KDF, Salt (hex), Nonce (hex), Iterations, MemoryKB, Parallelism — add as `omitempty` pointer field on `Manifest`
- [x] 2.2 Add `SetEncryption(algorithm, kdf string, salt, nonce []byte, iter, memKB, parallel int)` helper method on `Manifest`
- [x] 2.3 Add test in `internal/manifest/manifest_test.go` — verify encrypted manifest serializes Encryption struct, plaintext manifest omits it, round-trip JSON marshal/unmarshal preserves fields
- [x] 2.4 Verify: `go test ./internal/manifest/...` passes

## Phase 3: Push/Pull Encryption Wiring

- [x] 3.1 Create `internal/crypto/password.go` with `GetPassword(prompt string) → (string, error)` — checks `BAK_ENCRYPTION_PASSWORD` env var first, falls back to interactive stdin prompt, errors if no terminal and no env var
- [x] 3.2 Create `internal/crypto/password_test.go` — test env var precedence, test error on non-terminal without env var (mock stdin)
- [ ] 3.3 Modify `cmd/push.go` — add `--profile` flag, resolve profile from config, after `cloud.TarGzDirectory()` check `profile.Encryption.Enabled`, call `crypto.Encrypt()` + `crypto.GetPassword()`, update manifest with `Encryption` struct, then call `provider.Push()` with encrypted data
- [ ] 3.4 Modify `cmd/pull.go` — add `--profile` flag, after `provider.Pull()` check `crypto.IsEncrypted()`, if encrypted call `crypto.GetPassword()` + `crypto.Decrypt()`, on auth failure print "wrong password or corrupted archive — encryption is unrecoverable without the correct password" and write zero files, if not encrypted proceed as plaintext (backward compat)
- [ ] 3.5 Add tests in `cmd/push_test.go` and `cmd/pull_test.go` — mock provider, verify encrypted push produces `BAK_ENC\x01` prefix, verify pull decrypts correctly, verify wrong password error message, verify plaintext backward compat
- [ ] 3.6 Verify: `go test ./cmd/... ./internal/crypto/...` passes

## Phase 4: Config Migration v0.2.0 → v0.3.0

- [ ] 4.1 Add `ProfileConfig` struct and `EncryptionConfig` struct to `internal/config/config.go` — `ProfileConfig{Adapters, Categories, Preset, Provider, Encryption}`, `EncryptionConfig{Enabled bool}`, add `Profiles map[string]ProfileConfig` to `Config`
- [ ] 4.2 Add `isV020(cfg) → bool` detection function — returns true if `schema_version == "0.2.0"` and `Profiles` map is nil
- [ ] 4.3 Add `migrateV020(cfg, originalData) → error` — writes `config.json.v020.bak`, adds empty `Profiles` map, bumps `schema_version` to `"0.3.0"`, preserves all existing providers, saves
- [ ] 4.4 Wire `migrateV020` into `LoadPath()` — after v0.1.0 check, add v0.2.0 detection and migration
- [ ] 4.5 Add tests in `internal/config/config_test.go` — v0.2.0 config migrates to v0.3.0, `.v020.bak` created, providers preserved, schema_version bumped, idempotent (running again does not re-migrate)
- [ ] 4.6 Verify: `go test ./internal/config/...` passes

## Phase 5: Profile CRUD Commands

- [ ] 5.1 Create `cmd/profile.go` — add `profileCmd` parent command with `Use: "profile"`, register under `rootCmd`
- [ ] 5.2 Add `profileCreateCmd` — `bak profile create <name> --provider <name> [--preset <p>] [--adapters a,b] [--categories c,d] [--encrypt]` — validates provider exists in config and token is set, creates `ProfileConfig`, persists to config
- [ ] 5.3 Add `profileListCmd` — `bak profile list` — displays table of profiles with name, provider, preset, encryption status
- [ ] 5.4 Add `profileShowCmd` — `bak profile show <name>` — displays full profile details: adapters, categories, preset, provider, encryption enabled/disabled
- [ ] 5.5 Add `profileDeleteCmd` — `bak profile delete <name>` — removes profile from config, errors if profile does not exist
- [ ] 5.6 Create `cmd/profile_test.go` — test create with valid/invalid provider, create with missing token, list with multiple profiles, show existing/missing, delete existing/missing
- [ ] 5.7 Verify: `go test ./cmd/...` passes, manual test: `bak profile create test --provider github-gist`

## Phase 6: Backup Engine Profile Awareness

- [ ] 6.1 Modify `cmd/backup.go` — add `--profile` flag, when set load config and resolve profile, pass profile's preset/categories/adapters to `backup.Engine` instead of CLI flags
- [ ] 6.2 Modify `internal/backup/engine.go` — add `CustomCategories []string` field to `Engine` struct, when set use it instead of resolving from preset; add adapter filtering by profile's adapter list
- [ ] 6.3 Add test in `cmd/backup_test.go` — verify `--profile` flag resolves profile settings, verify engine uses profile's preset/categories
- [ ] 6.4 Verify: `go test ./cmd/... ./internal/backup/...` passes

## Phase 7: Integration Testing + Documentation

- [ ] 7.1 Create end-to-end integration test — create backup, push with encrypted profile, pull with same profile, verify decrypted archive matches original, verify manifest has Encryption struct
- [ ] 7.2 Create backward-compat integration test — push unencrypted archive (v0.2.0 style), pull with v0.3.0 binary, verify extraction works without encryption prompt
- [ ] 7.3 Create wrong-password integration test — push encrypted, pull with wrong password, verify error message and zero files restored
- [ ] 7.4 Update `README.md` — document `bak profile` commands, `--profile` flag, encryption workflow, `BAK_ENCRYPTION_PASSWORD` env var, backward compat notes
- [ ] 7.5 Update `CHANGELOG.md` — add v0.3.0 entry with encryption and profiles features
- [ ] 7.6 Verify: `go test ./...` passes with >80% coverage on new code, `go vet ./...` clean, `golangci-lint` clean
