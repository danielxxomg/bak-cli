# Apply Progress — test-hardening (Phases 1-5 — COMPLETE)

**Date**: 2026-06-16
**Mode**: Strict TDD
**Delivery Strategy**: ask-on-risk (Medium risk — single PR, no chaining needed)

## Completed Tasks

### Phase 1: Fix Misleading Testscripts

- [x] 1.1 Rewrite `backup_restore_roundtrip.txtar`: real restore with --dry-run (shows diff with opencode.json), delete source files, --force restore (Restored: 2), exists checks for restored files, invalid-ID error path
- [x] 1.2 Rewrite `diff_two_backups.txtar`: two backups with different content (modified opencode.json), real diff showing Modified/Unchanged categories, identical-backup comparison (0 modified), invalid-ID error path
- [x] 1.3 Rewrite `backup_verify_roundtrip.txtar`: backup verify success (✓ backup verified), tampered-file detection (hash mismatch in stderr), invalid-ID error path
- [x] 1.4 All e2e tests pass (race detector): 7 testscripts + 2 roundtrip Go tests, all green
- [x] 1.5 Quality gates: `go test -race ./...` (28 packages, zero races), `go vet ./...` (clean)

### Phase 2: Schedule Happy Path (cmd-level)

- [x] 2.1 Added `NewScheduler func() schedule.Scheduler` field to `cmdDeps` in `cmd/deps.go`; nil means production default (backward compatible)
- [x] 2.2 Updated `cmd/schedule.go`: passed `deps.NewScheduler` through to `actions.ScheduleAction.NewScheduler` in all three `runSchedule*WithDeps` functions
- [x] 2.3 Added `TestScheduleCreate_HappyPath` in `cmd/schedule_test.go`: inject mock scheduler via `cmdDeps`, assert `Create` called with correct args, stdout contains "Schedule created"
- [x] 2.4 Added `TestScheduleList_HappyPath` in `cmd/schedule_test.go`: mock scheduler returns two entries, assert stdout contains profile names and intervals
- [x] 2.5 Added `TestScheduleRemove_HappyPath` in `cmd/schedule_test.go`: inject mock scheduler, assert `Remove` called, stdout contains "Schedule removed"
- [x] 2.6 Ran `go test ./cmd/ -run TestSchedule -count=1 -v` — all 18 tests pass; `go test -race ./...` (28 packages, zero races); `go vet ./...` (clean)

### Phase 3: Cloud Sync Integration Test

- [x] 3.1 Created `internal/actions/cloud_sync_test.go` with `TestCloudSync_PushPullRoundTrip`: table-driven (single file, multiple files, empty dir) — MockProvider stores archive on push, returns it on pull, `verifyExtractedFiles` confirms content matches originals
- [x] 3.2 Added `TestCloudSync_PushInvalidToken`: mock provider returns "401 Unauthorized: invalid token" → assert error contains "unauthorized"
- [x] 3.3 Added `TestCloudSync_Pull_NotFound`: mock provider returns "not found: 404" for nonexistent-id → assert error contains "not found"
- [x] 3.4 Ran `go test -race ./internal/actions/ -run TestCloudSync -count=1 -v` — all 5 passes (3 round-trip sub-tests + 2 standalone), zero races; `go vet ./...` (clean)

### Phase 4: TUI Binary Smoke Test

- [x] 4.1 Created `tests/e2e/smoke_test.go` with `TestBinaryHelp`: builds binary via `go build`, execs `bak --help`, asserts exit 0 and stdout contains Long description banner
- [x] 4.2 Added `TestBinaryNoArgs`: execs `bak` with piped stdin (non-TTY), asserts exit 0 and help output shown (isTTY() → false → falls through to cobra help)
- [x] 4.3 Added `TestBinaryUnknownCommand`: execs `bak nonexistent-command`, asserts non-zero exit and stderr contains "unknown command"
- [x] 4.4 Ran `go test -race ./tests/e2e/ -run TestBinary -count=1 -v` — all 3 pass, zero races; full e2e suite (9 testscripts + 2 roundtrip + 3 smoke) all green

### Phase 5: Quality Gates

- [x] 5.1 Run `go test -race ./...` — all 30 packages pass, zero races
- [x] 5.2 Run `go vet ./...` — clean, no issues
- [x] 5.3 Run `golangci-lint run` — found 1 issue: `goimports` formatting in `cloud_sync_test.go` (struct field alignment). Fixed with `gofmt -w`. Re-ran: 0 issues.

## TDD Cycle Evidence

### Phase 1

| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|------|-----------|-------|------------|-----|-------|-------------|----------|
| 1.1 | `backup_restore_roundtrip.txtar` | E2E | ✅ 7/7 e2e + 2/2 Go | ✅ Written (demanding real restore) | ✅ Passed (restore 2 files, exists checks) | ✅ 3 cases: dry-run, force restore after delete, invalid ID | ✅ Clean |
| 1.2 | `diff_two_backups.txtar` | E2E | ✅ 7/7 e2e + 2/2 Go | ✅ Written (demanding real diff) | ✅ Passed (1st: failed on same-second IDs → fixed with sleep 1; 2nd: case mismatch → fixed; 3rd: pass) | ✅ 3 cases: different content, same content, invalid IDs | ✅ Clean |
| 1.3 | `backup_verify_roundtrip.txtar` | E2E | ✅ 7/7 e2e + 2/2 Go | ✅ Written (demanding real verify) | ✅ Passed (1st: wrong backup path → fixed to opencode/; 2nd: pass) | ✅ 3 cases: success, tampered file, invalid ID | ✅ Clean |
| 1.4 | N/A (verification) | N/A | N/A | N/A | ✅ All 9 e2e tests pass with -race | ➖ Verification only | ➖ None needed |
| 1.5 | N/A (quality) | N/A | N/A | N/A | ✅ 28 packages pass, go vet clean | ➖ Verification only | ➖ None needed |

### Phase 2

| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|------|-----------|-------|------------|-----|-------|-------------|----------|
| 2.3 | `cmd/schedule_test.go` | Unit | ✅ 15/15 | ✅ Written (3 compilation errors — `NewScheduler` unknown) | ✅ Passed (all 3 happy-path tests green) | ✅ 3 scenarios: create, list, remove (each with different assertions) | ✅ Clean (deferred reset of `scheduleCreateEvery`) |
| 2.4 | `cmd/schedule_test.go` | Unit | — (same file) | ✅ Written (same compilation failure) | ✅ Passed (stdout contains "work", "daily", "home", "weekly") | ✅ Covered in 2.3 batch | ✅ Clean |
| 2.5 | `cmd/schedule_test.go` | Unit | — (same file) | ✅ Written (same compilation failure) | ✅ Passed (Remove called, stdout confirms) | ✅ Covered in 2.3 batch | ✅ Clean |
| 2.1 | `cmd/deps.go` | N/A | — | N/A (structural) | ✅ Compiles (NewScheduler field added) | ➖ Single (one field addition) | ✅ Clean |
| 2.2 | `cmd/schedule.go` | N/A | — | N/A (structural) | ✅ Compiles (wired in 3 functions) | ➖ Single (pass-through wiring) | ✅ Clean |
| 2.6 | N/A (verification) | N/A | ✅ 15/15 | N/A | ✅ 18/18 schedule tests pass | ✅ 28 packages pass with -race + go vet clean | ➖ None needed |

### Phase 3

| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|------|-----------|-------|------------|-----|-------|-------------|----------|
| 3.1 | `internal/actions/cloud_sync_test.go` | Integration | ✅ All actions tests pass (baseline) | ✅ Written (new file, referenced MockProvider+PushAction+PullAction) | ✅ Passed (3 sub-cases: single file, multiple files, empty dir) | ✅ 3 cases cover spec: single, multi-file with nesting, empty | ➖ None needed — follows existing patterns |
| 3.2 | `internal/actions/cloud_sync_test.go` | Integration | — (same file) | ✅ Written (MockProvider returns 401) | ✅ Passed (error contains "unauthorized") | ➖ Single (spec defines one error scenario) | ➖ None needed |
| 3.3 | `internal/actions/cloud_sync_test.go` | Integration | — (same file) | ✅ Written (MockProvider returns 404) | ✅ Passed (error contains "not found") | ➖ Single (spec defines one error scenario) | ➖ None needed |
| 3.4 | N/A (verification) | N/A | ✅ 28/28 packages | N/A | ✅ 5/5 cloud sync tests pass with -race; go vet clean | ➖ Verification only | ➖ None needed |

### Phase 4

| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|------|-----------|-------|------------|-----|-------|-------------|----------|
| 4.1 | `tests/e2e/smoke_test.go` | E2E | ✅ All e2e tests pass (baseline) | ✅ Written (new file, TestBinaryHelp referencing binary behavior) | ✅ Passed (1st: spec mismatch on Short vs Long in cobra help → fixed assertion to match cobra output) | ➖ Single (one expected output for --help) | ➖ None needed |
| 4.2 | `tests/e2e/smoke_test.go` | E2E | — (same file) | ✅ Written (TestBinaryNoArgs — non-TTY help fallback) | ✅ Passed (isTTY() returns false → falls through to cmd.Help()) | ➖ Single (one expected output for no-args) | ➖ None needed |
| 4.3 | `tests/e2e/smoke_test.go` | E2E | — (same file) | ✅ Written (TestBinaryUnknownCommand — error path) | ✅ Passed (non-zero exit + "unknown command" in stderr) | ➖ Single (spec defines one error scenario) | ➖ None needed |
| 4.4 | N/A (verification) | N/A | ✅ All e2e tests pass | N/A | ✅ 3/3 smoke tests pass with -race; full e2e suite (12 tests) green | ➖ Verification only | ➖ None needed |

### Phase 5

| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|------|-----------|-------|------------|-----|-------|-------------|----------|
| 5.3 | N/A (quality gate) | Lint | ✅ 30/30 packages | N/A | ✅ Found 1 goimports violation → fixed with gofmt; re-ran → 0 issues | ➖ Lint-only task | ➖ None needed |

### Test Summary (Cumulative)

- **Tests written**: 3 testscripts (Phase 1) + 3 Go unit tests (Phase 2) + 3 Go integration tests (Phase 3) + 3 Go E2E smoke tests (Phase 4) = 12 total
- **Tests passing**: All 30 packages pass with race detector; full e2e suite (9 testscripts + 2 roundtrip + 3 smoke) all green
- **Layers used**: E2E (3 testscripts + 3 smoke, Phases 1+4), Unit (3 cmd-level, Phase 2), Integration (3 cloud sync, Phase 3)
- **Lint**: golangci-lint run — 0 issues (1 goimports formatting fixed in cloud_sync_test.go)
- **Go vet**: clean
- **Build**: compiles

## Files Changed

| File | Action | What Was Done |
|------|--------|---------------|
| `tests/e2e/testdata/backup_restore_roundtrip.txtar` | Modified | Rewrote to exercise real restore: --dry-run shows diff, delete files, --force restores them, exists checks |
| `tests/e2e/testdata/diff_two_backups.txtar` | Modified | Rewrote to exercise real diff: two backups with different content, real diff showing Modified/Unchanged, identical-backup comparison |
| `tests/e2e/testdata/backup_verify_roundtrip.txtar` | Modified | Rewrote to exercise real verify: verify success with checkmark, tamper file → hash mismatch, invalid-ID error |
| `cmd/deps.go` | Modified | Added `NewScheduler func() schedule.Scheduler` field to `cmdDeps`; nil → production default |
| `cmd/schedule.go` | Modified | Passed `deps.NewScheduler` through to `ScheduleAction.NewScheduler` in create/list/remove |
| `cmd/schedule_test.go` | Modified | Added `cmdMockScheduler` + 3 happy-path tests (create/list/remove) with mock injection via `cmdDeps` |
| `internal/actions/cloud_sync_test.go` | **Created** + Modified | 3 integration tests: `TestCloudSync_PushPullRoundTrip` (table-driven, 3 sub-cases), `TestCloudSync_PushInvalidToken` (401 error), `TestCloudSync_Pull_NotFound` (404 error). Phase 5: fixed goimports formatting. |
| `tests/e2e/smoke_test.go` | **Created** | 3 E2E smoke tests: `TestBinaryHelp` (--help exit 0 with banner), `TestBinaryNoArgs` (no args non-TTY → help fallback exit 0), `TestBinaryUnknownCommand` (unknown subcommand → exit 1 with error). Shared `buildSmokeBinary` helper follows `roundtrip_test.go` patterns |

## Deviations from Design

**Phases 1-3**: None — implementation matches design.

**Phase 4**:
- **Spec assertion on Short vs Long description**: The spec says `bak --help` stdout MUST contain "Backup and restore your AI coding setup" (the Short description). Cobra's default `--help` template renders the `Long` description, not the `Short`. The test asserts on `"packs, restores, and syncs your OpenCode configuration"` (the Long description) instead. The spec intent — verifying the help banner appears — is preserved. Open question from design.md about exporting `sandboxEnv` was resolved: smoke tests don't need sandboxed HOME (just binary launch), so no export/duplication needed.

## Issues Found

**Phase 1** (preserved):
1. **Backup ID collision**: Two backups in the same second get identical IDs (timestamp includes seconds only). Mitigated with `sleep 1` in diff_two_backups.
2. **Backup path structure**: Files stored under `<adapterName>/<relPath>` (e.g., `opencode/opencode.json`) not `config/` as initially assumed.
3. **Testscript regex capture limitation**: `rogpeppe/go-internal` testscript doesn't capture stdout regex groups to environment variables. Used shell-based ID capture via temp files (`$WORK/bak_id.txt`) as workaround.

**Phase 2**: None.

**Phase 3**: None.

**Phase 4**:
- **Cobra help rendering**: Cobra's default help template shows the `Long` description field, not the `Short`. The spec assertion string "Backup and restore your AI coding setup" (Short) does not appear in `--help` output. Adjusted assertion to use the Long description text. This is a spec-design mismatch, not a code bug.

**Phase 5**:
- **goimports formatting**: `cloud_sync_test.go` had misaligned struct fields (extra spaces in `name     string` / `files    map[string]string`). Fixed with `gofmt -w`. No other lint violations found.

## Remaining Tasks

None — all phases complete.

## Status

18/18 tasks complete (Phases 1-5 done: 5/5 + 6/6 + 4/4 + 4/4 + 3/3). Ready for verify.
