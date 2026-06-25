# Tasks: CI Hardening v2

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~490 |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | PR 1 config → PR 2 coverage → PR 3 DRY |
| Delivery strategy | ask-always |
| Chain strategy | stacked-to-main |

Decision needed before apply: Resolved (chained PRs, stacked-to-main)
Chained PRs recommended: Yes
Chain strategy: stacked-to-main (PR1 config → PR2 coverage → PR3 DRY)
400-line budget risk: High

## Phase 1: Linter Config (Item A)

- [x] 1.1 [CONFIG] Enable `maintidx`(20) + `dupl`(80) in `.golangci.yml`; extend `_test.go` exclusion. **Dep**: none. **~10 lines**. (REQ-RL-001/002/003)
- [x] 1.2 [CONFIG] Add `//nolint:maintidx` (SEVERE, deferred to qa-refactor-analysis) on `actions/backup.go:58`, `backup/engine.go:62`, `tui/model.go:127`. **Dep**: 1.1. **~3 lines**. (REQ-RL-001)

## Phase 2: Pre-Commit Hook (Item B)

- [x] 2.1 [CONFIG] Create `.githooks/pre-commit` — `set -euo pipefail`; runs golangci-lint, go vet, go build, gga run. **Dep**: none. **~18 lines**. (REQ-PCH-001/002/003)
- [x] 2.2 [CONFIG] Add `setup` task to `Taskfile.yml` → `git config core.hooksPath .githooks`. **Dep**: 2.1. **~6 lines**. (REQ-PCH-001)

## Phase 3: CI Lint Pin (Item C)

- [x] 3.1 [CONFIG] `ci.yml`: replace `go install @latest` with `golangci-lint-action@v8`, `version: v2.12.2`, `install-mode: binary`. **Dep**: none. **~20 lines**. (REQ-CI-006)

## Phase 4: Coverage Gate (Item D)

- [x] 4.1 [RED] Tests for `progress.Running`, `wizard.renderCheckboxList`, `wizard.renderConfirmSummary`. **Dep**: none. **~55 lines**. (REQ-CI-005)
- [x] 4.2 [GREEN] Verify coverage 80.0% → ~83%. No code changes. **Dep**: 4.1. **~0 lines**. (REQ-CI-005)
- [x] 4.3 [CONFIG] `cover:pkg` task in `Taskfile.yml` — per-pkg ≥80% for `internal/`, excludes `cmd/`. **Dep**: 4.1. **~25 lines**. (REQ-CI-005)
- [x] 4.4 [CONFIG] Add `task cover:pkg` to coverage job in `ci.yml`. **Dep**: 4.3. **~5 lines**. (REQ-CI-005)

## Phase 5: govulncheck Blocking (Item E)

- [x] 5.1 [CONFIG] `Taskfile.yml`: remove `|| true` from govulncheck (blocking), pin v1.4.0; keep gosec `|| true`. **Dep**: none. **~5+3 lines**. (REQ-CI-007)
- [x] 5.2 [CONFIG] `ci.yml`: pin `govulncheck@v1.4.0`, `gosec@v2.x` (replace `@latest`). **Dep**: 5.1. **~5+5 lines**. (REQ-CI-007)

## Phase 6: GGA Install (Item F)

- [x] 6.1 [CONFIG] `gga.yml`: replace `brew install` with `git clone --branch v2.8.1 + ./install.sh`; keep `continue-on-error: true`. **Dep**: none. **~8+4 lines**. (REQ-CI-008)

## Phase 7: DRY Consolidation (Item G)

- [x] 7.1 [RED] Test `pullContentFromAPI` (httptest table: success, 4xx, decode err). `cloud/httputil_test.go`. **~45 lines**.
- [x] 7.2 [GREEN] Implement `pullContentFromAPI` in `cloud/httputil.go`. **Dep**: 7.1. **~28 lines**.
- [x] 7.3 [REFACTOR] `GiteaProvider.Pull` + `GitHubRepoProvider.Pull` → delegate to helper. **Dep**: 7.2. **~45 lines**.
- [x] 7.4 [RED] Test `loadExcludes` via `setConfigHome`. `cmd/excludes_test.go`. **~30 lines**.
- [x] 7.5 [GREEN] Create `cmd/excludes.go` with `loadExcludes()`. **Dep**: 7.4. **~18 lines**.
- [x] 7.6 [REFACTOR] Replace inline closures in `cmd/backup.go` + `cmd/root.go`. **Dep**: 7.5. **~36 lines**.
- [x] 7.7 [RED] Test `mapBackupInfo` + `listBackupsForScreens` (nil-deps, error). `tui/model_test.go`. **~45 lines**.
- [x] 7.8 [GREEN] Implement both helpers in `tui/model.go`. **Dep**: 7.7. **~22 lines**.
- [x] 7.9 [REFACTOR] `initDashboard` + `initRestore` → `listBackupsForScreens`. **Dep**: 7.8. **~34 lines**.

## Phase 8: Documentation (Item H)

- [x] 8.1 [CONFIG] `CONTRIBUTING.md`: document `task setup`, note deferred linters. **Dep**: 2.2. **~12 lines**. (REQ-RL-003)
