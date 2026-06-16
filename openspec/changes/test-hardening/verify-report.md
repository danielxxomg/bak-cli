# Verification Report — test-hardening

**Change**: Test Hardening — Fix Misleading Testscripts and Close Coverage Gaps  
**Version**: N/A (SDD delta)  
**Mode**: Strict TDD  
**Date**: 2026-06-16  
**Verifier**: sdd-verify sub-agent

---

## Completeness

| Metric | Value |
|--------|-------|
| Tasks total | 21 (as listed in `tasks.md`) |
| Tasks complete | 21 |
| Tasks incomplete | 0 |

All tasks in Phases 1–5 are checked complete in `tasks.md` and `apply-progress.md`. No unchecked implementation tasks remain.

---

## Build & Tests Execution

**Build**: ✅ Passed

```text
$ go build .
# compiles cleanly
```

**Tests**: ✅ All passed / ❌ 0 failed / ⚠️ 0 skipped

```text
$ go test -race -count=1 ./...
?       github.com/danielxxomg/bak-cli        [no test files]
ok      github.com/danielxxomg/bak-cli/cmd                    20.753s
ok      github.com/danielxxomg/bak-cli/internal/actions       2.213s
ok      github.com/danielxxomg/bak-cli/internal/adapters      1.037s
ok      github.com/danielxxomg/bak-cli/internal/adapters/claudecode   1.023s
ok      github.com/danielxxomg/bak-cli/internal/adapters/codex        1.020s
ok      github.com/danielxxomg/bak-cli/internal/adapters/cursor       1.015s
ok      github.com/danielxxomg/bak-cli/internal/adapters/kilocode     1.019s
ok      github.com/danielxxomg/bak-cli/internal/adapters/kiro         1.020s
ok      github.com/danielxxomg/bak-cli/internal/adapters/opencode     1.023s
ok      github.com/danielxxomg/bak-cli/internal/adapters/pidev        1.021s
ok      github.com/danielxxomg/bak-cli/internal/adapters/register     1.016s
ok      github.com/danielxxomg/bak-cli/internal/adapters/windsurf     1.014s
ok      github.com/danielxxomg/bak-cli/internal/backup                1.044s
ok      github.com/danielxxomg/bak-cli/internal/cloud                 1.080s
ok      github.com/danielxxomg/bak-cli/internal/config                1.018s
?       github.com/danielxxomg/bak-cli/internal/config/testutil       [no test files]
ok      github.com/danielxxomg/bak-cli/internal/crypto                3.798s
ok      github.com/danielxxomg/bak-cli/internal/diff                  1.012s
ok      github.com/danielxxomg/bak-cli/internal/git                   1.100s
ok      github.com/danielxxomg/bak-cli/internal/manifest              1.016s
ok      github.com/danielxxomg/bak-cli/internal/paths                 1.017s
ok      github.com/danielxxomg/bak-cli/internal/presets               1.015s
ok      github.com/danielxxomg/bak-cli/internal/restore               1.050s
ok      github.com/danielxxomg/bak-cli/internal/schedule              1.033s
ok      github.com/danielxxomg/bak-cli/internal/tui                   1.033s
ok      github.com/danielxxomg/bak-cli/internal/tui/components        1.015s
ok      github.com/danielxxomg/bak-cli/internal/tui/screens           1.104s
ok      github.com/danielxxomg/bak-cli/internal/tui/styles            1.011s
ok      github.com/danielxxomg/bak-cli/tests/e2e                      5.093s
```

**Static analysis**: ✅ Clean

```text
$ go vet ./...
# no output (clean)

$ golangci-lint run
0 issues.
```

**Coverage**:

| File | Line % | Notes |
|------|--------|-------|
| `cmd/deps.go` | 100% | `depsFromCmd` fully covered |
| `cmd/schedule.go` | 100% | all `runSchedule*` functions fully covered |
| `internal/actions/push.go` | 82.5% | above 80% threshold |
| `internal/actions/pull.go` | 80.0% | at threshold |
| `internal/actions/cloud_sync_test.go` | N/A | test file |
| `tests/e2e/smoke_test.go` | N/A | test file (no statements) |

The `cmd` package as a whole reports 55.8% coverage, but that is driven by pre-existing uncovered command files; the files changed by this PR are at 100%.

---

## Spec Compliance Matrix

| Capability | Scenario | Test | Result |
|------------|----------|------|--------|
| testscript-restore-roundtrip | restore with dry-run shows expected files | `tests/e2e/testdata/backup_restore_roundtrip.txtar` | ✅ COMPLIANT |
| testscript-restore-roundtrip | restore with force actually restores files | `tests/e2e/testdata/backup_restore_roundtrip.txtar` | ✅ COMPLIANT |
| testscript-restore-roundtrip | restore with invalid ID fails gracefully | `tests/e2e/testdata/backup_restore_roundtrip.txtar` | ✅ COMPLIANT |
| testscript-diff-two-backups | diff between two backups shows differences | `tests/e2e/testdata/diff_two_backups.txtar` | ✅ COMPLIANT |
| testscript-diff-two-backups | diff with identical backups shows no differences | `tests/e2e/testdata/diff_two_backups.txtar` | ✅ COMPLIANT |
| testscript-diff-two-backups | diff with invalid IDs fails gracefully | `tests/e2e/testdata/diff_two_backups.txtar` | ✅ COMPLIANT |
| testscript-verify-roundtrip | verify confirms backup integrity | `tests/e2e/testdata/backup_verify_roundtrip.txtar` | ✅ COMPLIANT |
| testscript-verify-roundtrip | verify with tampered file detects corruption | `tests/e2e/testdata/backup_verify_roundtrip.txtar` | ✅ COMPLIANT |
| testscript-verify-roundtrip | verify with invalid ID fails gracefully | `tests/e2e/testdata/backup_verify_roundtrip.txtar` | ✅ COMPLIANT |
| cloud-sync-integration | action-level push creates gist and pull retrieves content | `internal/actions/cloud_sync_test.go > TestCloudSync_PushPullRoundTrip` | ✅ COMPLIANT |
| cloud-sync-integration | push with invalid token returns authentication error | `internal/actions/cloud_sync_test.go > TestCloudSync_PushInvalidToken` | ✅ COMPLIANT |
| cloud-sync-integration | pull for non-existent gist returns not found error | `internal/actions/cloud_sync_test.go > TestCloudSync_Pull_NotFound` | ✅ COMPLIANT |
| schedule-happy-path | schedule create succeeds with valid profile and interval (cmd level) | `cmd/schedule_test.go > TestScheduleCreate_HappyPath` | ⚠️ PARTIAL |
| schedule-happy-path | schedule list shows active schedules (cmd level) | `cmd/schedule_test.go > TestScheduleList_HappyPath` | ✅ COMPLIANT |
| schedule-happy-path | schedule remove succeeds (cmd level) | `cmd/schedule_test.go > TestScheduleRemove_HappyPath` | ⚠️ PARTIAL |
| tui-launch-smoke | bak binary with --help exits cleanly | `tests/e2e/smoke_test.go > TestBinaryHelp` | ⚠️ PARTIAL |
| tui-launch-smoke | bak binary with no args in non-TTY shows help | `tests/e2e/smoke_test.go > TestBinaryNoArgs` | ✅ COMPLIANT |
| tui-launch-smoke | bak binary with unknown subcommand fails gracefully | `tests/e2e/smoke_test.go > TestBinaryUnknownCommand` | ✅ COMPLIANT |

**Compliance summary**: 15/18 scenarios fully compliant, 3/18 partial.

### Partial scenarios explained

- **schedule create / remove (cmd level)**: Tests verify mock scheduler invocation and stdout confirmation, but do **not** assert that the profile's `Schedule` config is updated (`Enabled: true, Interval: "daily"` on create; `nil` on remove). The mutation is implemented in `internal/actions/schedule.go` and covered indirectly by `internal/actions/schedule_test.go`, but the cmd-level tests omit this assertion.
- **bak --help exits cleanly**: The spec asserts the stdout contains the Short description `"Backup and restore your AI coding setup"`. Cobra's default `--help` template renders the Long description instead, so the test asserts on the Long text (`"packs, restores, and syncs your OpenCode configuration"`). The intent — help banner appears and exit code is 0 — is preserved, but the literal spec wording is not matched.

---

## Correctness (Static Evidence)

| Requirement | Status | Notes |
|-------------|--------|-------|
| Rewritten testscripts exercise real restore/diff/verify | ✅ Implemented | `backup_restore_roundtrip.txtar`, `diff_two_backups.txtar`, `backup_verify_roundtrip.txtar` all call real commands |
| Cloud sync push→pull round-trip | ✅ Implemented | `cloud_sync_test.go` uses `MockProvider` to capture and return archive bytes |
| TUI binary smoke tests | ✅ Implemented | `smoke_test.go` builds binary and exercises `--help`, no-args, and unknown-command paths |
| Schedule happy path with mock injection | ✅ Implemented | `cmd/deps.go` exposes `NewScheduler`; `cmd/schedule.go` passes it through to `ScheduleAction` |
| Backward-compatible `cmdDeps` extension | ✅ Implemented | `NewScheduler` is optional; nil falls through to `schedule.NewScheduler()` |
| Error wrapping with `%w` / context | ✅ Implemented | Existing patterns preserved; new test code uses `fmt.Errorf` |
| Table-driven tests | ✅ Implemented | `TestCloudSync_PushPullRoundTrip` uses table-driven subtests |
| No `panic` in new code | ✅ Implemented | All errors returned, no panics |
| Cross-platform path handling in tests | ✅ Implemented | `filepath.Join`, `t.TempDir()`, OS-aware binary extension in smoke tests |

---

## Coherence (Design)

| Decision | Followed? | Notes |
|----------|-----------|-------|
| Rewrite misleading testscripts in-place | ✅ Yes | Filenames now match behavior |
| Cloud sync test in `internal/actions/cloud_sync_test.go` | ✅ Yes | New file, does not bloat `push_test.go` |
| TUI smoke tests in `tests/e2e/smoke_test.go` | ✅ Yes | Separate from `roundtrip_test.go` |
| Schedule happy path at cmd level with mock injection | ✅ Yes | `cmdDeps.NewScheduler` wired through all three schedule commands |
| Reuse existing mock infrastructure | ✅ Yes | `MockProvider`, `MockProviderFactory`, hand-rolled `cmdMockScheduler` |
| Optional `NewScheduler` preserves production behavior | ✅ Yes | nil falls through to `schedule.NewScheduler()` |

**Design deviations**:

1. **Cloud sync mechanism**: The spec/proposal mention using `setupMockGistAPI` (HTTP server) for the action-level test. The implementation uses `MockProvider`/`MockProviderFactory` instead, which is consistent with `design.md` and still exercises the full push→pull action flow. Behavior is preserved.
2. **`tui_launch.txtar` not created**: `design.md` listed `tests/e2e/testdata/tui_launch.txtar` as a create item, but the implementation covered the same no-args/--help scenarios in `smoke_test.go` (more reliable in CI/non-TTY environments). No spec scenario is lost.

---

## TDD Compliance (Strict TDD Mode)

| Check | Result | Details |
|-------|--------|---------|
| TDD Evidence reported | ✅ | `apply-progress.md` contains TDD Cycle Evidence tables for Phases 1–5 |
| All tasks have tests | ✅ | 6/6 test files verified to exist |
| RED confirmed (tests exist) | ✅ | All reported test files exist in the codebase |
| GREEN confirmed (tests pass) | ✅ | All reported tests pass on fresh execution |
| Triangulation adequate | ✅ | Multi-case triangulation present for testscript scenarios and `TestCloudSync_PushPullRoundTrip`; single-case items match spec scope |
| Safety Net for modified files | ✅ | All modified files report baseline safety-net status in apply-progress |

**TDD Compliance**: 6/6 checks passed

### Test files verified

| File | Exists | Passes |
|------|--------|--------|
| `tests/e2e/testdata/backup_restore_roundtrip.txtar` | ✅ | ✅ |
| `tests/e2e/testdata/diff_two_backups.txtar` | ✅ | ✅ |
| `tests/e2e/testdata/backup_verify_roundtrip.txtar` | ✅ | ✅ |
| `cmd/schedule_test.go` | ✅ | ✅ |
| `internal/actions/cloud_sync_test.go` | ✅ | ✅ |
| `tests/e2e/smoke_test.go` | ✅ | ✅ |

---

## Test Layer Distribution

| Layer | Tests | Files | Tools |
|-------|-------|-------|-------|
| Unit | 3 (schedule happy path) | 1 (`cmd/schedule_test.go`) | Go testing stdlib |
| Integration | 5 (cloud sync: 3 round-trip subtests + 2 error cases) | 1 (`internal/actions/cloud_sync_test.go`) | Go testing stdlib + hand-rolled mocks |
| E2E | 12 (7 testscripts + 2 roundtrip Go tests + 3 smoke tests) | 4 (`tests/e2e/roundtrip_test.go`, `tests/e2e/smoke_test.go`, 3 txtar files) | `testscript`, `exec.Command` |
| **Total** | **20** | **6** | |

Note: `cmd/schedule_test.go` contains additional pre-existing schedule tests; the count above reflects tests added/extended by this change.

---

## Changed File Coverage

| File | Line % | Branch % | Uncovered Lines | Rating |
|------|--------|----------|-----------------|--------|
| `cmd/deps.go` | 100% | N/A | — | ✅ Excellent |
| `cmd/schedule.go` | 100% | N/A | — | ✅ Excellent |
| `internal/actions/push.go` | 82.5% | N/A | error branches inside `Run` | ⚠️ Acceptable |
| `internal/actions/pull.go` | 80.0% | N/A | error branches inside `Run` | ⚠️ Acceptable |
| `tests/e2e/smoke_test.go` | N/A | N/A | — | ➖ Test file |
| `internal/actions/cloud_sync_test.go` | N/A | N/A | — | ➖ Test file |

**Average changed production-file coverage**: 90.6% (4 files).

---

## Assertion Quality

| File | Line | Assertion | Issue | Severity |
|------|------|-----------|-------|----------|
| `cmd/schedule_test.go` | 282–290 | `len(sched.createCalls) == 1`, `profile == "work"`, `interval == "daily"` | Mock call-count assertion verifies wiring; idiomatic for hand-rolled Go mocks in this codebase | SUGGESTION |
| `cmd/schedule_test.go` | 360–364 | `len(sched.removeCalls) == 1`, `removeCalls[0] == "work"` | Mock call-count assertion verifies wiring | SUGGESTION |
| `internal/actions/cloud_sync_test.go` | 110–112 | `storedArchive == ""` | Verifies Push was actually invoked | SUGGESTION |

**Assertion quality**: 0 CRITICAL, 0 WARNING, 3 SUGGESTIONS.

The mock call-count assertions are appropriate here because they verify the cmd→action→scheduler wiring without invoking real OS schedulers, which is the explicit design choice for CI-compatible schedule tests.

---

## Issues Found

**CRITICAL**: None

**WARNING**:

1. **Partial spec compliance — schedule config mutation not asserted at cmd level**  
   `TestScheduleCreate_HappyPath` and `TestScheduleRemove_HappyPath` do not verify that the profile's `Schedule` config is updated (`Enabled: true, Interval: "daily"` on create; `nil` on remove). The behavior is implemented in `internal/actions/schedule.go`, but the cmd-level scenarios in the spec include these assertions.

2. **Partial spec compliance — TUI `--help` smoke test asserts Long description, not Short**  
   The spec says `bak --help` stdout MUST contain `"Backup and restore your AI coding setup"` (Short description). Cobra renders the Long description by default, so the test asserts on that text instead. The help banner is verified, but the literal spec wording is not.

3. **Design deviation — cloud sync integration uses `MockProvider` instead of `setupMockGistAPI` HTTP server**  
   The spec/proposal reference `setupMockGistAPI`; the implementation uses the hand-rolled `MockProvider`/`MockProviderFactory` as chosen in `design.md`. The action-level push→pull behavior is fully exercised; no behavior is broken.

**SUGGESTION**:

1. Consider strengthening `TestScheduleCreate_HappyPath` and `TestScheduleRemove_HappyPath` to assert the in-memory `cfg.Profiles["work"].Schedule` state after the action runs, matching the spec's "Then" clauses exactly.
2. Consider updating the spec text for the TUI `--help` scenario to reference the Long description that Cobra actually renders, eliminating the wording mismatch.
3. The `cmd` package overall coverage is 55.8%; while the files changed by this PR are at 100%, future work could extend cmd-level coverage for other commands.

---

## Verdict

**PASS WITH WARNINGS**

All 21 tasks are complete, all tests pass with the race detector, `go vet` and `golangci-lint` are clean, and every spec scenario has at least a passing covering test. The warnings are limited to:

- Two cmd-level schedule tests that omit the profile `Schedule` config mutation assertion (behavior implemented, just not asserted).
- A TUI smoke test that verifies the Long help description instead of the Short description stated in the spec (Cobra rendering quirk).
- A design-consistent deviation in cloud sync integration test infrastructure (`MockProvider` vs. HTTP server).

No critical issues, no failing tests, no regressions.
