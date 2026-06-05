# Proposal: v0.2.0 — Multi-Agent Adapters & Cloud Backend Abstraction

## Intent

bak-cli v0.1.0 only backs up OpenCode configs to GitHub Gists. Users of Claude Code, Cursor, Codex, Windsurf, Kiro, pi.dev, and KiloCode cannot use bak. Cloud sync is hardcoded to Gist — no GitLab, Codeberg, or self-hosted options. This change makes bak agent-agnostic and backend-agnostic.

## Scope

### In Scope
- Adapter implementations for 7 agents: Claude Code, Cursor, Codex, Windsurf, Kiro, pi.dev, KiloCode
- Provider interface abstracting cloud push/pull from Gist-specific code
- Provider implementations: GitHub Repo, Codeberg, rclone, Gitea/Forgejo
- Config migration: v0.1.0 flat `{github_token, gist_id}` → v0.2.0 multi-backend format
- Backward compatibility: read and restore v0.1.0 backups seamlessly

### Out of Scope
- Encryption at rest (deferred to v0.3.0)
- Merge-mode restore (overwrite-only stays)
- GUI / TUI provider selection wizard
- Session/auth state backup

## Capabilities

### New Capabilities
- `multi-agent-adapters`: Adapter implementations for Claude Code, Cursor, Codex, Windsurf, Kiro, pi.dev, KiloCode — each following the existing `Adapter` interface
- `cloud-provider-interface`: `Provider` interface with Push/Pull/List operations, decoupling cloud sync from GitHub Gist
- `config-migration`: Auto-migration from v0.1.0 flat config to v0.2.0 multi-backend config on first load

### Modified Capabilities
- `cloud-sync`: Requirements change from "GitHub Gist only" to "pluggable provider backend" — existing Gist becomes one provider among many
- `agent-adapters`: Registry now auto-discovers multiple agents, not just OpenCode

## Approach

**Adapters**: Each agent gets a sub-package under `internal/adapters/` following the `opencode/` pattern. Detection uses known config directory paths per OS. Priority order: Claude Code → Cursor → Codex → Windsurf → Kiro → KiloCode → pi.dev (by market share).

**Provider interface**: Define `Provider` interface in `internal/cloud/provider.go` with `Push(archive, meta)`, `Pull(id) → archive`, `List() → []BackupMeta`. Refactor existing Gist code into `GitHubGistProvider`. Add `GitHubRepoProvider`, `CodebergProvider`, `RcloneProvider`, `GiteaProvider`.

**Config migration**: On `config.Load()`, detect v0.1.0 schema (presence of `github_token` + `gist_id` at root). Auto-migrate to v0.2.0 nested structure: `providers.github.token`, `providers.github.gist_id`. Write migrated config back with `schema_version: "0.2.0"` marker. Non-breaking — old keys still readable via compat shim.

**Backward compat**: Manifest v0.1.0 already uses `map[string]AdapterManifest` — no schema bump needed. Restore engine detects adapter name in manifest and routes to correct adapter. v0.1.0 backups contain only `"opencode"` key — works as-is.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/adapters/claudecode/` | New | Claude Code adapter |
| `internal/adapters/cursor/` | New | Cursor adapter |
| `internal/adapters/codex/` | New | Codex adapter |
| `internal/adapters/windsurf/` | New | Windsurf adapter |
| `internal/adapters/kiro/` | New | Kiro adapter |
| `internal/adapters/kilocode/` | New | KiloCode adapter |
| `internal/adapters/pidev/` | New | pi.dev adapter |
| `internal/cloud/provider.go` | New | Provider interface definition |
| `internal/cloud/github_gist.go` | Modified | Refactor existing gist.go → GitHubGistProvider |
| `internal/cloud/github_repo.go` | New | GitHub Repo provider |
| `internal/cloud/codeberg.go` | New | Codeberg provider |
| `internal/cloud/rclone.go` | New | rclone provider |
| `internal/cloud/gitea.go` | New | Gitea/Forgejo provider |
| `internal/config/config.go` | Modified | Multi-backend schema + v0.1.0 migration |
| `cmd/push.go` | Modified | Use Provider interface instead of direct Gist calls |
| `cmd/pull.go` | Modified | Use Provider interface instead of direct Gist calls |
| `cmd/login.go` | Modified | Multi-provider auth flow |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Agent config paths vary by OS/version | Medium | Table-driven path detection per adapter with fallback scan |
| Config migration corrupts existing setups | Low | Write backup of old config before migration; validate round-trip |
| Provider API differences (Codeberg ≠ GitHub) | Medium | Common denominator interface; provider-specific tests with mocks |
| rclone provider requires external binary | High | Detect rclone in PATH; graceful error with install instructions |
| Large backup archives exceed Gist file size limit | Low | Document 10MB Gist limit; recommend GitHub Repo provider for large configs |

## Rollback Plan

- Config migration preserves original file as `config.json.v010.bak` — restore manually if needed
- Adapter registration is additive — removing new adapter packages reverts to v0.1.0 behavior
- Provider interface wraps existing Gist code — default provider stays GitHub Gist if no config change
- Git tag `v0.1.0` remains available; `go install` can pin to it

## Dependencies

- rclone binary (optional, for rclone provider)
- No new Go module dependencies — all providers use net/http or exec

## Success Criteria

- [ ] `bak backup` discovers and backs up all installed agents (not just OpenCode)
- [ ] `bak push --provider github-repo` pushes to a GitHub repo (not just Gist)
- [ ] `bak pull --provider codeberg` pulls from Codeberg
- [ ] v0.1.0 config auto-migrates to v0.2.0 format without user intervention
- [ ] v0.1.0 backups restore correctly with v0.2.0 binary
- [ ] `go test ./...` passes with >80% coverage on new code
- [ ] All adapters tested on Windows, macOS, Linux path patterns
