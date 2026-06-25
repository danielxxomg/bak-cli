# Tasks: qa-refactor-analysis

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~870 (additions + deletions) |
| 400-line budget risk | High |
| 800-line budget risk | Medium |
| Chained PRs recommended | Yes |
| Suggested split | PR 1 (Phases 1-4, Tier 1) → PR 2 (Phases 5-7, Tier 2) → PR 3 (Phases 8-11, Tier 3 + linter) |
| Delivery strategy | ask-always |
| Chain strategy | pending (user decision required) |

Decision needed before apply: Yes
Chained PRs recommended: Yes
Chain strategy: pending
400-line budget risk: High

### Suggested Work Units

| Unit | Goal | Likely PR | Notes |
|------|------|-----------|-------|
| 1 | Engine consolidation + Cloud List + tui.Screen + RenderTooSmall | PR 1 | base: main; ~430 lines; tests included |
| 2 | Model.Update submodel map + resolveBackupID + hostname/loadConfig | PR 2 | base: PR 1 branch; ~310 lines; depends on PR 1 |
| 3 | cmd/ extractions + linter enable + nolint cleanup | PR 3 | base: PR 2 branch; ~210 lines; depends on PR 2 |

## Phase 1: Engine Consolidation (T1.1) — CRITICAL

- [x] **1.1 [RED]** Write test: manifest.Items excludes secret files (10 files, 2 secrets → len==8). Currently FAILS on CLI path.
  - **Files:** `internal/actions/backup_test.go`
  - **Deps:** —
  - **Lines:** +35
  - **Spec:** REQ-BE-001 §"Manifest Items count excludes secrets"
  - **Accept:** Test asserts `len(manifest.Items)==8`; secret files absent from manifest

- [x] **1.2 [GREEN]** Create `internal/backup/workflow.go`: `FS` interface, `osFS`, `Context` struct, `Run(Context)(*Result,error)` — the 8 canonical phases. Adopt Engine's `secretRelPaths` skip-map + BackupAction's `RemoveAll`.
  - **Files:** `internal/backup/workflow.go` (new)
  - **Deps:** 1.1
  - **Lines:** +180
  - **Spec:** REQ-BE-001 §"Consolidated backup engine", §"Backup with secrets produces manifest without secret entries"
  - **Accept:** `go test ./internal/backup/...` green; 1.1 test now passes

- [x] **1.3 [REFACTOR]** `BackupAction.Run` delegates to `backup.Run`; build `Context{FS:a.FS,...}`; print report from `Result`. Remove `//nolint:maintidx`.
  - **Files:** `internal/actions/backup.go`
  - **Deps:** 1.2
  - **Lines:** -80/+20
  - **Spec:** REQ-BE-001 §"CLI and TUI paths use same implementation"
  - **Accept:** `go test ./internal/actions/...` green; BackupAction.Run <50 lines

- [x] **1.4 [REFACTOR]** `Engine.Run` delegates to `backup.Run`; build `Context{FS:e.FS or osFS{},...}`. Remove `//nolint:maintidx`.
  - **Files:** `internal/backup/engine.go`
  - **Deps:** 1.2
  - **Lines:** -70/+15
  - **Spec:** REQ-BE-001 §"CLI and TUI paths use same implementation"
  - **Accept:** `go test ./internal/backup/...` green; Engine.Run <30 lines

- [x] **1.5 [VERIFY]** Integration: CLI and TUI paths produce byte-identical manifests over same fixture.
  - **Files:** `internal/backup/workflow_test.go` or `internal/actions/backup_test.go`
  - **Deps:** 1.3, 1.4
  - **Lines:** +40
  - **Spec:** REQ-BE-001 §"CLI and TUI paths use same implementation"
  - **Accept:** Test runs both paths, asserts identical Items/checksums/ordering

- [x] **1.6 [GREEN]** Preserve exclusion pipeline: test `config.LoadExcludes` called, `SetScanOptions` called, `.bakignore` applied.
  - **Files:** `internal/backup/workflow_test.go`
  - **Deps:** 1.2
  - **Lines:** +30
  - **Spec:** REQ-BE-001 §"Consolidated engine preserves exclusion pipeline"
  - **Accept:** Mock adapters verify SetScanOptions called; excludes loaded

## Phase 2: Cloud List() Dedup (T1.2)

- [x] **2.1 [RED]** Test `listContentsDir` shared helper: parameterized URL/accept/prefix, 404→empty, error prefix propagation.
  - **Files:** `internal/cloud/httputil_test.go`
  - **Deps:** —
  - **Lines:** +50
  - **Spec:** REQ-CP-001 §"shared logic parameterized", §"HTTP error propagated"
  - **Accept:** httptest server validates URL/headers/prefix per provider

- [x] **2.2 [GREEN]** Implement `listContentsDir(client, url, token, accept, errPrefix, urlBuilder)` in `internal/cloud/httputil.go`.
  - **Files:** `internal/cloud/httputil.go`
  - **Deps:** 2.1
  - **Lines:** +40
  - **Spec:** REQ-CP-001 §"Cloud List() consolidation"
  - **Accept:** 2.1 tests pass

- [x] **2.3 [REFACTOR]** `GiteaProvider.List` and `GitHubRepoProvider.List` delegate to `listContentsDir`.
  - **Files:** `internal/cloud/gitea.go`, `internal/cloud/github_repo.go`
  - **Deps:** 2.2
  - **Lines:** -50/+10
  - **Spec:** REQ-CP-001 §"GiteaProvider.List returns correct items", §"GitHubRepoProvider.List returns correct items"
  - **Accept:** Existing gitea/github_repo tests green; each List() <20 lines

## Phase 3: tui.Screen Unexport (T1.3)

- [x] **3.1 [REFACTOR]** Rename `type Screen int` → `type screen int`. Verify zero `cmd/` references (confirmed). Constants stay exported if needed.
  - **Files:** `internal/tui/model.go` (or `types.go`/`dispatch.go`)
  - **Deps:** —
  - **Lines:** ~10
  - **Spec:** REQ-TD-002 §"grep confirms no cmd/ references", §"tui package compiles"
  - **Accept:** `go build ./...` green; `grep -r 'tui\.Screen' cmd/` returns 0 matches

## Phase 4: styles.RenderTooSmall (T1.4)

- [x] **4.1 [RED]** Test `RenderTooSmall(width, height int) string` returns "Terminal too small (WxH)".
  - **Files:** `internal/tui/styles/styles_test.go`
  - **Deps:** —
  - **Lines:** +15
  - **Spec:** REQ-TD-003 §"RenderTooSmall produces correct message"
  - **Accept:** `RenderTooSmall(15, 5)` contains "Terminal too small (15x5)"

- [x] **4.2 [GREEN]** Implement `RenderTooSmall` in `internal/tui/styles/styles.go`.
  - **Files:** `internal/tui/styles/styles.go`
  - **Deps:** 4.1
  - **Lines:** +10
  - **Spec:** REQ-TD-003 §"RenderTooSmall produces correct message"
  - **Accept:** 4.1 test passes

- [x] **4.3 [REFACTOR]** Replace inline "Terminal too small" in `model.go:523`, `dashboard.go:150`, `health.go:127` with `styles.RenderTooSmall`.
  - **Files:** `internal/tui/model.go`, `internal/tui/screens/dashboard.go`, `internal/tui/screens/health.go`
  - **Deps:** 4.2
  - **Lines:** -15/+5
  - **Spec:** REQ-TD-003 §"all too-small guards use shared helper"
  - **Accept:** `grep "Terminal too small" internal/tui/` matches only `styles/`

## Phase 5: Model.Update Submodel Map (T2.1) — HIGH RISK

- [x] **5.1 [RED]** Test `subModel` interface + `forwardTo` dispatch: key routes to correct sub-model, unknown screen returns false, lazy-init populates map.
  - **Files:** `internal/tui/model_test.go`
  - **Deps:** —
  - **Lines:** +50
  - **Spec:** REQ-TD-001 §"key event routed", §"unknown screen does not panic", §"lazy-init populates"
  - **Accept:** Tests verify routing, no-panic on unknown, map populated after screenChange

- [x] **5.2 [GREEN]** Implement `subModel` interface, `m.subs map[screen]subModel`, `forwardTo` helper in `internal/tui/model.go`.
  - **Files:** `internal/tui/model.go`
  - **Deps:** 5.1
  - **Lines:** +40
  - **Spec:** REQ-TD-001 §"Model.Update submodel dispatch"
  - **Accept:** 5.1 tests pass; compiles

- [x] **5.3 [REFACTOR]** `Model.Update` and `handleKey` use map dispatch via `forwardTo`; remove 21 type-assert-reassign blocks. `View` extracts `renderScreen`.
  - **Files:** `internal/tui/model.go`
  - **Deps:** 5.2
  - **Lines:** -120/+30
  - **Spec:** REQ-TD-001 §"WindowSizeMsg routed", §"ProgressStepMsg handled directly"
  - **Accept:** Existing tui tests green; Update <80 lines; handleKey <40 lines

- [x] **5.4 [GREEN]** Remove `//nolint:maintidx` from `model.go:127`. Verify gocognit <35.
  - **Files:** `internal/tui/model.go`
  - **Deps:** 5.3
  - **Lines:** -1
  - **Spec:** REQ-RL-002 §"golangci-lint reports 0 maintidx violations"
  - **Accept:** `gocognit` on model.go Update <35; no maintidx nolint

## Phase 6: resolveBackupID Consolidation (T2.2)

- [x] **6.1 [RED]** Test `LatestBackupID(backupsDir)` and `ListBackupIDs(backupsDir)`: latest, empty→error, descending sort.
  - **Files:** `internal/backup/resolve_test.go`
  - **Deps:** —
  - **Lines:** +35
  - **Spec:** REQ-CH-001 §"resolves by latest", §"not-found returns error"
  - **Accept:** Table-driven: latest returned, empty→error, sorted desc

- [x] **6.2 [GREEN]** Implement `LatestBackupID` + `ListBackupIDs` in `internal/backup/resolve.go`.
  - **Files:** `internal/backup/resolve.go`
  - **Deps:** 6.1
  - **Lines:** +25
  - **Spec:** REQ-CH-001 §"resolveBackupID canonical"
  - **Accept:** 6.1 tests pass

- [x] **6.3 [REFACTOR]** Replace 3 inline resolutions: `push.go:201`, `pick_backup.go:33`, `cleanup.go:60` → use `backup.LatestBackupID`/`ListBackupIDs`.
  - **Files:** `internal/actions/push.go`, `internal/actions/pick_backup.go`, `internal/actions/cleanup.go`
  - **Deps:** 6.2
  - **Lines:** -30/+10
  - **Spec:** REQ-CH-001 §"all call sites use canonical function"
  - **Accept:** `grep` for inline sort+backupID logic in actions/ returns 0; tests green

## Phase 7: hostname + loadConfig Helpers (T2.3)

- [x] **7.1 [RED]** Test `backup.ResolveHostname`: injected fn returns, nil→os.Hostname, error→"unknown"+verbose warn.
  - **Files:** `internal/backup/hostname_test.go`
  - **Deps:** —
  - **Lines:** +30
  - **Spec:** REQ-CH-002 §"hostname returns correct value", §"falls back", §"defaults to unknown"
  - **Accept:** 3 scenarios covered; verbose warning captured via bytes.Buffer

- [x] **7.2 [GREEN]** Implement `ResolveHostname` in `internal/backup/hostname.go`.
  - **Files:** `internal/backup/hostname.go` (new)
  - **Deps:** 7.1
  - **Lines:** +20
  - **Spec:** REQ-CH-002 §"hostname helper consolidated"
  - **Accept:** 7.1 tests pass

- [x] **7.3 [REFACTOR]** Replace 3 hostname duplications: `backup.go:137`, `push.go:119`, `engine.go:127` → `backup.ResolveHostname`.
  - **Files:** `internal/actions/backup.go`, `internal/actions/push.go`, `internal/backup/engine.go`
  - **Deps:** 7.2
  - **Lines:** -25/+5
  - **Spec:** REQ-CH-002 §"hostname helper consolidated"
  - **Accept:** No inline hostname resolution in actions/; engine.go uses helper

- [x] **7.4 [RED]** Test `loadConfigOr`: injected loader returns, nil→config.Load, error propagated.
  - **Files:** `internal/actions/config_test.go` or `internal/actions/pull_test.go`
  - **Deps:** —
  - **Lines:** +25
  - **Spec:** REQ-CH-003 §"loadConfig returns correct config", §"falls back", §"error handling"
  - **Accept:** 3 scenarios covered

- [x] **7.5 [GREEN]** Implement `loadConfigOr` method in `internal/actions/` (shared via embedded struct or interface).
  - **Files:** `internal/actions/interfaces.go` or new `internal/actions/config_helpers.go`
  - **Deps:** 7.4
  - **Lines:** +15
  - **Spec:** REQ-CH-003 §"loadConfig helper consolidated"
  - **Accept:** 7.4 tests pass

- [x] **7.6 [REFACTOR]** Replace 2 loadConfig duplications: `pull.go:71-83`, `push.go:178-185` → `loadConfigOr`.
  - **Files:** `internal/actions/pull.go`, `internal/actions/push.go`
  - **Deps:** 7.5
  - **Lines:** -20/+4
  - **Spec:** REQ-CH-003 §"loadConfig helper consolidated"
  - **Accept:** No inline ConfigLoader nil-check in pull/push; tests green

## Phase 8: cmd/profile Wizard Dedup (T3.1)

- [x] **8.1 [RED]** Test extracted `launchWizard` helper.
  - **Files:** `cmd/profile_test.go` or `cmd/wizard_test.go`
  - **Deps:** —
  - **Lines:** +25
  - **Spec:** (Tier 3 quick win, no explicit REQ — supports REQ-RL-001 funlen reduction)
  - **Accept:** Helper returns correct wizard model + action

- [x] **8.2 [GREEN]** Implement `launchWizard(cfg, name, providers)` in `cmd/profile.go` or `cmd/wizard.go`.
  - **Files:** `cmd/profile.go`
  - **Deps:** 8.1
  - **Lines:** +30
  - **Spec:** (funlen 49 + nestif 6 reduction)
  - **Accept:** `runProfileCreateInteractiveWithDeps` <40 stmts

- [x] **8.3 [REFACTOR]** Replace 2 inline wizard-launch blocks (name=="" and name!="") with `launchWizard` call.
  - **Files:** `cmd/profile.go`
  - **Deps:** 8.2
  - **Lines:** -45/+5
  - **Spec:** (funlen/nestif reduction)
  - **Accept:** `runProfileCreateInteractiveWithDeps` funlen <49 stmts; nestif <6

## Phase 9: diff/cleanup/list Extract-Methods (T3.2)

- [x] **9.1 [RED]** Test extracted helpers: `printDiffGroups`, `printDiffSummary`, `printDryRunPlan`, `formatBackupRow`.
  - **Files:** `internal/actions/diff_backups_test.go`, `internal/actions/cleanup_test.go`, `internal/actions/list_local_test.go`
  - **Deps:** —
  - **Lines:** +40
  - **Spec:** (Tier 3 quick wins, supports REQ-RL-001)
  - **Accept:** Each helper produces correct output for table-driven inputs

- [x] **9.2 [GREEN]** Implement extracted helpers in respective files.
  - **Files:** `internal/actions/diff_backups.go`, `internal/actions/cleanup.go`, `internal/actions/list_local.go`
  - **Deps:** 9.1
  - **Lines:** +40
  - **Spec:** (funlen reduction)
  - **Accept:** 9.1 tests pass; parent functions funlen <45 stmts

- [x] **9.3 [REFACTOR]** Replace inline logic with helper calls; dedup "No backups found" branches.
  - **Files:** `internal/actions/diff_backups.go`, `internal/actions/cleanup.go`, `internal/actions/list_local.go`
  - **Deps:** 9.2
  - **Lines:** -30/+10
  - **Spec:** (funlen reduction)
  - **Accept:** All 3 functions funlen <45 stmts; existing tests green

## Phase 10: cmd/backup applyProfileOverrides (T3.3)

- [x] **10.1 [RED]** Test `applyProfileOverrides(deps, cfg)` returns preset, cats, adapters.
  - **Files:** `cmd/backup_test.go`
  - **Deps:** —
  - **Lines:** +20
  - **Spec:** (Tier 3, supports REQ-RL-001 funlen/nestif reduction)
  - **Accept:** Profile overrides applied correctly; no-profile returns defaults

- [x] **10.2 [GREEN]** Implement `applyProfileOverrides` in `cmd/backup.go`.
  - **Files:** `cmd/backup.go`
  - **Deps:** 10.1
  - **Lines:** +20
  - **Spec:** (funlen 76 + nestif 8 reduction)
  - **Accept:** 10.1 tests pass

- [x] **10.3 [REFACTOR]** Replace inline profile-override block in `runBackupWithDeps` with `applyProfileOverrides` call.
  - **Files:** `cmd/backup.go`
  - **Deps:** 10.2
  - **Lines:** -20/+5
  - **Spec:** (funlen/nestif reduction)
  - **Accept:** `runBackupWithDeps` funlen <76; nestif <8

## Phase 11: Linter Enable (CONFIG)

- [x] **11.1 [CONFIG]** Enable gocognit(35), funlen(80/50), nestif(6) in `.golangci.yml`. Add test exclusion for all 3 linters.
  - **Files:** `.golangci.yml`
  - **Deps:** All Phase 1-10 refactors complete
  - **Lines:** +15
  - **Spec:** REQ-RL-001 §"gocognit enabled at threshold 35", §"funlen enabled", §"nestif enabled", §"test files exempt"
  - **Accept:** Config valid; `golangci-lint run ./...` exits 0

- [x] **11.2 [CONFIG]** Add `//nolint:gocognit // inherent: tar/gzip walk is fixed algorithm` to `tarGZDir`, `untarGzDir`. Add `//nolint:funlen // static key-bindings table` to `RenderShortcuts`.
  - **Files:** `internal/cloud/pack.go`, `internal/tui/screens/shortcuts.go`
  - **Deps:** 11.1
  - **Lines:** +6
  - **Spec:** REQ-RL-003 §"tarGZDir has nolint with reason", §"no new nolint beyond Tier 4"
  - **Accept:** Each nolint has reason comment; only 3 Tier-4 nolints exist

- [x] **11.3 [VERIFY]** `golangci-lint run ./...` produces 0 violations. `grep '//nolint:gocognit\|//nolint:funlen\|//nolint:nestif'` returns only Tier-4 items.
  - **Files:** — (verification only)
  - **Deps:** 11.1, 11.2
  - **Lines:** 0
  - **Spec:** REQ-RL-001 §"zero violations on refactored non-test code", REQ-RL-003 §"no new nolint beyond Tier 4"
  - **Accept:** Exit code 0; only Tier-4 nolint annotations with reasons
  - **Deviation from plan:** Acceptance originally said "exactly 3" nolints, but `tarGzDir` legitimately exceeds BOTH the gocognit (35) and funlen (80) budgets for the same inherent tar/gzip walk reason, so it carries two Tier-4 nolints. Actual count is 4 — all Tier-4, all with reasons, no non-Tier-4 nolints. The "exactly 3" prediction predated funlen actually being enabled (PR3 forgot to enable it). Verified: `golangci-lint run ./...` → 0 issues with funlen enabled.

---

## Implementation Order

**Phase 1 first** — the engine consolidation is the centerpiece with highest risk and highest value. It removes ~150 duplicated lines and 2 of 3 SEVERE functions. All other phases benefit from a stable foundation.

**Phases 2-4** are independent Tier-1 items that can proceed in parallel after Phase 1.

**Phase 5** (Model.Update) is the second highest risk — core TUI routing. Do after Tier-1 is stable.

**Phases 6-7** are low-risk helper consolidations.

**Phases 8-10** are Tier-3 quick wins — do after engine consolidation to avoid churn.

**Phase 11** MUST be last — linters enabled only after all violations resolved.

## Risks

| Risk | Severity | Mitigation |
|------|----------|------------|
| Engine consolidation breaks CLI or TUI path | High | TDD red/green on both test suites; 1.5 byte-identical integration test |
| Model.Update submodel map breaks TUI routing | High | Strong existing coverage (tui 80%+); 5.1 tests for unknown-screen no-panic |
| Cloud List() abstraction obscures intent | Medium | Parameterize URL/header/prefix; existing gitea/github_repo tests as safety net |
| resolveHostname in backup/ (spec correction) | Low | Documented; import-direction constraint makes this the only valid location |
