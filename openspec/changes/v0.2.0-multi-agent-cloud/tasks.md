# Tasks: v0.2.0 — Multi-Agent Adapters & Cloud Backend Abstraction

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 1800–2200 |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | PR 1 → PR 2 → PR 3 → PR 4 |
| Delivery strategy | ask-on-risk |
| Chain strategy | feature-branch-chain |

Decision needed before apply: Yes
Chained PRs recommended: Yes
Chain strategy: feature-branch-chain
400-line budget risk: High

### Suggested Work Units

| Unit | Goal | Likely PR | Notes |
|------|------|-----------|-------|
| 1 | Provider interface + GitHubGistProvider + config migration | PR 1 | base: feature/v0.2.0; tests included |
| 2 | New providers (GitHubRepo, Gitea, Codeberg, Rclone) | PR 2 | base: PR 1 branch; depends on provider.go |
| 3 | 7 agent adapters + RegisterAll + backup cmd wiring | PR 3 | base: PR 2 branch; each adapter isolated |
| 4 | CLI integration (--provider flag, push/pull/login/list) | PR 4 | base: PR 3 branch; end-to-end verification |

## Phase 1: Cloud Provider Interface + Registry (Foundation)

- [x] 1.1 Create `internal/cloud/provider.go`: `Provider` interface (Name/Push/Pull/List), `PushMeta`, `BackupMeta` structs, `ProviderRegistry` with Register/Get/SetDefault/NewProviderRegistry
- [x] 1.2 Create `internal/cloud/provider_test.go`: table-driven tests for registry — register/get, unknown provider error, default provider fallback
- [x] 1.3 Create `internal/cloud/github_gist.go`: `GitHubGistProvider` struct wrapping existing `gist.go` funcs (CreateGist/UpdateGist/GetGist) into Provider interface; token resolution via env → config
- [x] 1.4 Create `internal/cloud/github_gist_test.go`: test Push/Pull/List with `httptest.Server` mocking GitHub Gist API responses

## Phase 2: Config Migration (v0.1.0 → v0.2.0)

- [x] 2.1 Modify `internal/config/config.go`: add `SchemaVersion` field, `Providers map[string]ProviderConfig` nested struct; keep old fields as compat shim
- [x] 2.2 Add `isV010()` detection in `LoadPath()`: checks for `github_token` + `gist_id` at root AND no `schema_version`; write `config.json.v010.bak` before migrating
- [x] 2.3 Add migration transform: `github_token` → `providers.github.token`, `gist_id` → `providers.github.gist_id`, set `schema_version: "0.2.0"`
- [x] 2.4 Update `Get()`/`Set()` to support new nested keys (`providers.github.token`, `providers.codeberg.token`, etc.)
- [x] 2.5 Create `internal/config/migration_test.go`: table-driven tests — v0.1.0 detected + migrated, v0.2.0 skipped, `.v010.bak` created, idempotent re-load

## Phase 3: New Provider Implementations

- [x] 3.1 Create `internal/cloud/gitea.go`: `GiteaProvider` with configurable `BaseURL`; implements Push/Pull/List via Gitea API (file content endpoints); token from env `GITEA_TOKEN` or config. Includes `CodebergProvider` wrapper with fixed BaseURL.
- [x] 3.2 Create `internal/cloud/gitea_test.go`: test with `httptest.Server` simulating Gitea API. Covers Gitea + Codeberg providers (20 tests).
- [x] 3.3 Create `internal/cloud/codeberg.go`: `CodebergProvider` embedding `GiteaProvider` with fixed `BaseURL = "https://codeberg.org"`; token from `CODEBERG_TOKEN` — implemented as type in gitea.go.
- [x] 3.4 Create `internal/cloud/codeberg_test.go`: verify delegation to GiteaProvider with correct base URL — included in gitea_test.go.
- [x] 3.5 Create `internal/cloud/github_repo.go`: `GitHubRepoProvider` using GitHub Contents API; token from `GITHUB_TOKEN`; config key `repo` (owner/name)
- [x] 3.6 Create `internal/cloud/github_repo_test.go`: test Push/Pull/List with `httptest.Server` mocking Contents API (17 tests)
- [x] 3.7 Create `internal/cloud/rclone.go`: `RcloneProvider` shelling out via `os/exec` with swappable `execCommand`; config key `remote`; Push = `rclone copyto`, Pull = `rclone cat`, List = `rclone lsf`
- [x] 3.8 Create `internal/cloud/rclone_test.go`: test remote validation, missing binary error, command construction with mock exec via batch/shell scripts and env-var-controlled failure mode (16 tests)

## Phase 4: Multi-Agent Adapters

- [x] 4.1 Create `internal/adapters/claudecode/adapter.go` + `adapter_test.go`: Name="claude-code", configPath=`.claude/`, categories: config (settings.json, CLAUDE.md), skills, commands
- [x] 4.2 Create `internal/adapters/cursor/adapter.go` + `adapter_test.go`: Name="cursor", configPath=`.cursor/`, categories: config (settings), extensions
- [x] 4.3 Create `internal/adapters/codex/adapter.go` + `adapter_test.go`: Name="codex", configPath=`.codex/`, categories: config, instructions
- [x] 4.4 Create `internal/adapters/windsurf/adapter.go` + `adapter_test.go`: Name="windsurf", configPath=`.codeium/windsurf/`, categories: config, rules
- [x] 4.5 Create `internal/adapters/kiro/adapter.go` + `adapter_test.go`: Name="kiro", configPath=`.kiro/`, categories: config, hooks
- [x] 4.6 Create `internal/adapters/kilocode/adapter.go` + `adapter_test.go`: Name="kilocode", configPath=`.kilocode/`, categories: config, rules
- [x] 4.7 Create `internal/adapters/pidev/adapter.go` + `adapter_test.go`: Name="pidev", configPath=`.pi/`, categories: config, agents
- [x] 4.8 Create `internal/adapters/register/register.go` + `register_test.go`: `RegisterAll()` function registering all 8 adapters in priority order (Claude Code → Cursor → Codex → Windsurf → Kiro → KiloCode → pi.dev → OpenCode). Lives in separate package to avoid circular imports.
- [x] 4.9 Modify `cmd/backup.go`: replace manual opencode registration with `register.All(reg)`

## Phase 5: CLI Integration

- [ ] 5.1 Modify `cmd/push.go`: add `--provider` flag (default "github-gist"); build `ProviderRegistry`, resolve provider, call `provider.Push()` instead of direct Gist calls
- [ ] 5.2 Modify `cmd/pull.go`: add `--provider` flag; build `ProviderRegistry`, call `provider.Pull()` instead of direct Gist calls
- [ ] 5.3 Modify `cmd/list.go`: add `--provider` flag; call `provider.List()` to display backups from selected backend
- [ ] 5.4 Modify `cmd/login.go`: keep GitHub-only for now; add `--provider` flag stub that errors for non-GitHub providers with message to use `bak config set`
- [ ] 5.5 Update `internal/cloud/auth.go`: add `ResolveProviderToken(provider, cfg)` supporting per-provider env vars and config keys

## Phase 6: Testing + Verification

- [ ] 6.1 Run `go test ./...` — verify all existing + new tests pass
- [ ] 6.2 Run `go test -cover ./internal/cloud/ ./internal/config/ ./internal/adapters/...` — verify >80% coverage on new code
- [ ] 6.3 Manual E2E: `bak backup` discovers multiple agents, `bak push --provider github-gist` round-trips, `bak pull` restores
- [ ] 6.4 Verify config migration: create v0.1.0 config.json, run `bak push`, confirm auto-migration + `.v010.bak` created

## Phase 7: Documentation + Cleanup

- [ ] 7.1 Update `README.md`: document `--provider` flag, new supported agents, config migration
- [ ] 7.2 Add godoc comments on all new exported types (Provider, ProviderRegistry, PushMeta, BackupMeta, each provider struct)
- [ ] 7.3 Update CLI `--help` text for push/pull/list/login to mention provider support
