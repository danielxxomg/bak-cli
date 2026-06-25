# Design: CI Hardening v2

## Technical Approach

Config/infra hardening — no Go architecture changes. Adds deterministic CI
teeth (lint, coverage, vulncheck, pre-commit) while keeping GGA advisory (REQ-CI-004).
Three corrections verified live.

## Architecture Decisions

| # | Decision | Choice | Rationale |
|---|----------|--------|-----------|
| A | Refactoring linters | Enable `maintidx`(20) + `dupl`(80); **3 `nolint:maintidx`** on SEVERE fns | **Correction**: explore claimed 0 violations; live run shows 3 (`BackupAction.Run` MI15, `Engine.Run` MI17, `Model.Update` MI13) — same fns deferred to qa-refactor-analysis. Ratchet: linter active for new code, 3 known-debt marked. |
| B | Pre-commit | Versioned `.githooks/pre-commit` + `task setup` → `core.hooksPath` | Zero new deps. Hook: `golangci-lint && go vet && go build && gga run`, fail-fast. |
| C | Lint in CI | `golangci/golangci-lint-action@v8`, `version: v2.12.2`, `install-mode: binary` | Pinned prebuilt (no compile), free cache, `verify: true` validates config. Replaces `go install @latest`. |
| D | Coverage gate | `task cover:pkg` awk-parses `go test -cover`, fail if any `internal/` pkg <80% | No new dep. Excludes `cmd/`, root, testutil. tui/screens bumped via 3 cheap tests on 0% fns. |
| E | govulncheck | Remove `|| true`; **block on ANY reachable vuln**; pin v1.4.0 | **Correction**: v1.4.0 has NO `-severity` flag. Reachability IS the filter (low FP). gosec stays `|| true` (advisory). |
| F | GGA install | `git clone --branch v2.8.1 + ./install.sh` | **Correction**: GGA is pure Bash (97.4% Shell), not Go. `go install` impossible. Keep `continue-on-error` (REQ-CI-004). |
| G | DRY consolidation | 3 extractions (see Contracts) | Eliminates 3 dupl pairs; TDD: test helper first (RED), extract (GREEN) |
| H | Deferred linters | gocognit/funlen/nestif + maintidx-SEVERE → qa-refactor-analysis | 44 violations incl. 3 architectural SEVERE need real extraction |

## Data Flow

```
Local:  git commit → .githooks/pre-commit → golangci-lint → go vet → go build → gga run (fail-fast)
CI:     push/PR → ci.yml
              ├─ lint     → action@v8 (pinned v2.12.2, binary, cache)
              ├─ cover    → task cover (75% floor) + task cover:pkg (80% per-internal) ← NEW
              ├─ security → govulncheck (BLOCKS) + gosec (advisory)
              └─ gga.yml  → clone v2.8.1 + install.sh → gga run (advisory, continue-on-error)
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `.golangci.yml` | Modify | Enable `maintidx`(20) + `dupl`(80); add both to `_test.go` exclusion |
| `.githooks/pre-commit` | Create | `set -e; golangci-lint run && go vet ./... && go build ./... && gga run` |
| `Taskfile.yml` | Modify | Add `setup` (core.hooksPath) + `cover:pkg` (80% gate); `security`: drop `|| true` on govulncheck, pin versions |
| `.github/workflows/ci.yml` | Modify | Lint: action@v8; security: pin govulncheck@v1.4.0, gosec@v2.x; cover: add `task cover:pkg` |
| `.github/workflows/gga.yml` | Modify | Replace brew with `git clone --branch v2.8.1 + ./install.sh` |
| `internal/cloud/httputil.go` | Modify | Add `pullContentFromAPI` |
| `internal/cloud/{gitea,github_repo}.go` | Modify | `Pull` delegates to `pullContentFromAPI` |
| `cmd/excludes.go` | Create | `loadExcludes()` extracted from backup.go + root.go |
| `cmd/{backup,root}.go` | Modify | `ExcludesLoader: loadExcludes` |
| `internal/tui/model.go` | Modify | Add `mapBackupInfo` + `listBackupsForScreens`; `initDashboard`/`initRestore` delegate |
| `CONTRIBUTING.md` | Modify | Document `task setup` one-time step |

## Interfaces / Contracts

**A. `.golangci.yml`** (v2 schema, verified):
```yaml
linters:
  enable: [maintidx, dupl]          # appended
  settings: { maintidx: { min-complexity: 20 }, dupl: { threshold: 80 } }
  exclusions:
    rules:
      - path: '(.+_test\.go)'
        linters: [errcheck, gosec, maintidx, dupl]   # extend existing
```
3 `nolint`: `//nolint:maintidx // SEVERE — deferred to qa-refactor-analysis` on `actions/backup.go:58`, `backup/engine.go:62`, `tui/model.go:127`.

**G1. `pullContentFromAPI`** (`cloud/httputil.go`) — Pull NOT touched by 2026-06-17 archive (that did Push/types):
```go
func pullContentFromAPI(client *http.Client, token, url, accept string, wrap func(string, ...any) error) ([]byte, error)
```
Gitea passes `p.errf`+`"application/json"`; GitHub a closure+`"application/vnd.github+json"`. Validation stays in each `Pull`.

**G2. `loadExcludes`** (`cmd/excludes.go`): `func loadExcludes() (adapters.ScanOptions, error)` — `config.Load → paths.ConfigDir → config.LoadExcludes`.

**G3. `mapBackupInfo` + `listBackupsForScreens`** (`tui/model.go`):
```go
func mapBackupInfo([]tui.BackupInfo) []screens.BackupInfo
func (m Model) listBackupsForScreens() ([]screens.BackupInfo, error)  // nil-check + ListBackups + mapBackupInfo
```
`initDashboard`/`initRestore` delegate to `m.listBackupsForScreens`.

## Testing Strategy

| Layer | What | Approach |
|-------|------|----------|
| Unit | `pullContentFromAPI` | Table-driven, `httptest.Server` (RED before extract) |
| Unit | `loadExcludes` | DI config home via `setConfigHome(t, ...)` |
| Unit | `mapBackupInfo`, `listBackupsForScreens` | Pure-fn table tests; nil-deps branch |
| Unit | tui/screens 0% fns | Cover `progress.Running`, `wizard.renderCheckboxList/renderConfirmSummary` → ~83% |
| Integration | `golangci-lint run` | 0 reported violations (dupl fixed, maintidx nolint-suppressed) |
| E2E | `.githooks/pre-commit` | Stage, commit, assert hook runs |

## Migration / Rollout

No migration. Additive config + isolated refactors. Rollback: `git revert`. One-time `task setup` per clone; `--no-verify` hatch in AGENTS.md #41.

## Open Questions

- [ ] GGA `install.sh` PATH on ubuntu-latest — verify in apply; fallback: `echo "$HOME/bin" >> "$GITHUB_PATH"`
- [ ] gosec exact pin — confirm latest v2.x at apply time
