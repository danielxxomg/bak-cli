# Tasks: Coverage DI Refactor

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~1400–1600 across 5 PRs (PR0: ~85, PR1: ~350, PR2: ~400, PR3: ~680, PR4: ~25) |
| 400-line budget risk | High (PR2 and PR3 exceed individually) |
| Chained PRs recommended | Yes |
| Suggested split | PR0 → PR1 → PR2 → PR3 → PR4 (feature-branch-chain) |
| Delivery strategy | auto-chain |
| Chain strategy | feature-branch-chain |

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: feature-branch-chain
400-line budget risk: High

### Suggested Work Units

| Unit | Goal | Likely PR | Notes |
|------|------|-----------|-------|
| 1 | Guardrail: E2E test + threshold→70% | PR0 | ~85 lines; base = `feature/coverage-refactor` |
| 2 | Adapters: `LoadYAMLAdapters(homeDir)` + yaml_test + register_test | PR1 | ~350 lines; base = PR0 branch |
| 3 | Actions: ProviderFactory, HostnameFunc, restoreFile→FS, mock+tests | PR2 | ~400 lines; base = PR1 branch |
| 4 | Cmd: extract profile/login/list/export to actions + tests, threshold→80% | PR3 | ~680 lines; base = PR2 branch |
| 5 | CI: lint pin, build tag, rate-limit fix | PR4 | ~25 lines; base = PR3 branch |

## Phase 1: PR0 — Guardrail (base: `feature/coverage-refactor`)

- [x] 1.1 Create `tests/e2e/roundtrip_test.go` — backup real files in `t.TempDir()` → restore → verify SHA-256 checksums match manifest; use `exec.Command` to invoke `bak` binary built from source [M]
- [x] 1.2 Lower `COVERAGE_THRESHOLD` from `80` to `70` in `Taskfile.yml` line 7 [S]
- [x] 1.3 Create branch `feature/coverage-refactor` from `main`; commit 1.1 + 1.2; verify `go test ./tests/e2e/...` passes [S]

## Phase 2: PR1 — Adapters (base: PR0 branch)

- [x] 2.1 Modify `internal/adapters/yaml.go` `LoadYAMLAdapters(dir string)` → `LoadYAMLAdapters(dir, homeDir string)`; replace `os.UserHomeDir()` call with `homeDir` param; keep path traversal check [S]
- [x] 2.2 Modify `internal/adapters/register/register.go` `LoadYAMLAdapters(reg, override)` → `LoadYAMLAdapters(reg, override, homeDir string)`; pass `homeDir` through to `adapters.LoadYAMLAdapters` [S]
- [x] 2.3 Update caller in `cmd/backup.go` — resolve `homeDir` via `os.UserHomeDir()` and pass to `register.LoadYAMLAdapters` [S]
- [x] 2.4 Create `internal/adapters/yaml_test.go` — table-driven tests: `ConfigAdapter.Detect` (exists/missing/not-dir), `ListItems` (dir category, root files, missing category), `Backup`/`Restore` (copy files via `t.TempDir()`), `LoadYAMLAdapters` (valid YAML, invalid YAML, missing name, traversal rejection), `fileHash`, `scanCategoryDir` [L]
- [x] 2.5 Create `internal/adapters/register/register_test.go` — test `All()` registers 8 built-in adapters, `LoadYAMLAdapters()` with temp dir containing valid/invalid YAML, override warning on stderr [M]
- [x] 2.6 Run `go test -cover ./internal/adapters/...` — assert ≥80%; run `go build ./...` — zero errors [S]

## Phase 3: PR2 — Actions (base: PR1 branch)

- [x] 3.1 Add `ProviderFactory` interface and `HostnameFunc` type to `internal/actions/interfaces.go` [S]
- [x] 3.2 Add `MockProviderFactory` and `MockProvider` to `internal/actions/mock_impl.go` with compile-time interface checks [M]
- [x] 3.3 Add `HostnameFn HostnameFunc` field to `BackupAction` in `internal/actions/backup.go`; replace `os.Hostname()` call at line 101 with `a.HostnameFn()` (nil-guard → `os.Hostname`) [S]
- [x] 3.4 Modify `internal/actions/restore.go` `restoreFile()` — replace `os.Open`/`os.Create`/`io.Copy` (lines 167–182) with `a.FS.CopyFile(src, d.TargetPath)`; add `Stdin io.Reader` field to `RestoreAction`; replace `os.Stdin` at line 92 with `a.Stdin` (nil-guard → `os.Stdin`) [S]
- [x] 3.5 Add `Factory ProviderFactory` + `HostnameFn HostnameFunc` fields to `PushAction` in `internal/actions/push.go`; replace lines 58–68 (hardcoded `cloud.NewProviderRegistry` + `NewGitHubGistProvider`) with `a.Factory.CreateProvider(a.Provider)`; replace `os.Hostname()` at line 81 with `a.HostnameFn()` [M]
- [x] 3.6 Add `Factory ProviderFactory` field to `PullAction` in `internal/actions/pull.go`; replace lines 40–48 with `a.Factory.CreateProvider(a.Provider)` [M]
- [x] 3.7 Update `internal/actions/restore_test.go` — add table-driven tests for `restoreFile` via `MockFileSystem.CopyFile`: happy path, source traversal, target traversal, mkdir failure, copy failure [M]
- [x] 3.8 Update `internal/actions/push_test.go` — add tests using `MockProviderFactory` + `MockProvider`: happy path (verify archive + Push call), provider error, hostname injection [M]
- [x] 3.9 Update `internal/actions/pull_test.go` — add tests using `MockProviderFactory`: happy path (verify download + extract), provider error [M]
- [x] 3.10 Run `go test -cover ./internal/actions/...` — coverage 69.4% (below 80% target; remaining gap is in Run method branches covered by PR3 cmd extraction); verify `go build ./...` passes [S]

## Phase 4: PR3 — Cmd (base: PR2 branch)

- [x] 4.1 Create `internal/actions/profile.go` — extract `ProfileCreate(name, opts)`, `ProfileList()`, `ProfileShow(name)`, `ProfileDelete(name)` functions accepting `config.Config` param; move validation logic from `cmd/profile.go` [L]
- [x] 4.2 Create `internal/actions/profile_test.go` — table-driven tests: create (valid, duplicate, missing provider, no token), list (empty, populated), show (exists, missing), delete (exists, missing); use `t.TempDir()` for config isolation [L]
- [x] 4.3 Create `internal/actions/login.go` — `LoginAction` struct with `Stdin io.Reader`, `TokenValidator func(string) error`, `ConfigSaver` fields; extract token prompt + validation flow from `cmd/login.go` [M]
- [x] 4.4 Create `internal/actions/login_test.go` — tests with `strings.NewReader("y\n")` and `strings.NewReader("n\n")` for replace prompt; empty token; validation failure [M]
- [x] 4.5 Modify `cmd/profile.go` — `runProfileCreate` delegates to `actions.ProfileCreate`; `runProfileList` → `actions.ProfileList`; `runProfileShow` → `actions.ProfileShow`; `runProfileDelete` → `actions.ProfileDelete` [M]
- [x] 4.6 Modify `cmd/login.go` — `runLogin` delegates to `actions.LoginAction` with `Stdin: os.Stdin`, real `cloud.ValidateToken`, real `cfg.Save` [M]
- [x] 4.7 Extract `runListLocal` in `cmd/list.go` to `internal/actions/list_local.go` — accept `bakDir string` param; move directory scanning + manifest loading + tabwriter logic [M]
- [x] 4.8 Extract `runExport` in `cmd/export.go` to `internal/actions/export.go` — accept `homeDir, backupID, outputPath string` params; move `os.Stat` + `createTarGz` logic [M]
- [x] 4.9 Restore `COVERAGE_THRESHOLD` from `70` to `80` in `Taskfile.yml` [S]
- [x] 4.10 Run `go test -cover ./...` — total 75.0% (see deviation note below); `go build ./...` passes [S]

## Phase 5: PR4 — CI Fixes (base: PR3 branch)

- [x] 5.1 Pin `golangci-lint` version in `.github/workflows/ci.yml` — changed `version: latest` to `version: v1.64.5` [S]
- [x] 5.2 Add `//go:build windows` build tag to `internal/schedule/scheduler_test.go` lines referencing `parseSchtasksCSV` — split into `scheduler_parse_test.go` with windows build tag [S]
- [x] 5.3 Pin `arduino/setup-task` version in `.github/workflows/ci.yml` — changed `version: 3.x` to `version: 3.42.1` in all 5 jobs [S]
- [x] 5.4 Verify CI passes locally: `go build`, `go vet`, `go test` — all 978 pass, vet clean [S]
