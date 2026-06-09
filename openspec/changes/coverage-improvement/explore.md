# Exploration: Detailed Coverage Analysis for cmd/ and actions/

## Current State

- **cmd/ total coverage**: 46.6% — most uncovered code is thin cobra `runX` → `runXWithDeps` delegation wrappers and `init()` flag registration.
- **actions/ total coverage**: 82.9% — already above 80% threshold, but specific functions have gaps in error paths and edge cases.
- **E2E**: 5 testscript files + 1 Go roundtrip test covering backup, restore, verify, diff, profile, schedule.

---

## cmd/ Detailed Analysis

| Function | File:Line | Current % | Category | Effort | Recommendation |
|----------|-----------|-----------|----------|--------|----------------|
| `init` | backup.go:38 | 77.8% | boilerplate | — | skip (flag registration tested by TestBackupCmd_FlagsAfterInit) |
| `runBackup` | backup.go:51 | 0.0% | boilerplate | low | test (1-line delegation, add `runBackup(cmd, nil)` test) |
| `init` | export.go:30 | 0.0% | boilerplate | — | skip (flag tested by TestCmdFlagRegistration) |
| `init` | list.go:33 | 66.7% | boilerplate | — | skip (flag tested) |
| `runListWithDeps` | list.go:43 | 33.3% | unit | low | test (local path via `runListLocal` not exercised) |
| `init` | login.go:43 | 0.0% | boilerplate | — | skip |
| `runLogin` | login.go:51 | 0.0% | boilerplate | low | test (1-line delegation) |
| `runLoginWithDeps` | login.go:55 | 0.0% | unit | med | test (non-interactive path with mock config + token validator) |
| `runLoginInteractiveWithDeps` | login.go:77 | 0.0% | TUI | med | test (inject isTTY + Wizard fn, test non-TTY error + cancelled flow) |
| `init` | pick.go:30 | 100% | — | — | skip |
| `runPick` | pick.go:131 | 0.0% | boilerplate | low | test (1-line delegation) |
| `runPickWithDeps` | pick.go:136 | 0.0% | TUI | med | test (inject isTTY + Picker fn, test non-TTY error + confirmed flow) |
| `init` | pull.go:36 | 25.0% | boilerplate | — | skip |
| `runPull` | pull.go:44 | 0.0% | boilerplate | low | test (1-line delegation) |
| `runPullWithDeps` | pull.go:51 | 0.0% | unit | med | test (wire mock Factory to test arg parsing + error path) |
| `init` | push.go:36 | 33.3% | boilerplate | — | skip |
| `runPush` | push.go:44 | 0.0% | boilerplate | low | test (1-line delegation) |
| `runPushWithDeps` | push.go:49 | 0.0% | unit | med | test (wire mock Factory to test arg parsing + error path) |
| `init` | restore.go:35 | 100% | — | — | skip |
| `runRestoreWithDeps` | restore.go:50 | 0.0% | unit | low | test (valid ID + missing backup → error from ResolveBackup) |
| `Execute` | root.go:26 | 0.0% | os.Exit | — | skip (tested indirectly via rootCmd.Execute in TestExecute_*) |
| `formatSize` | root.go:40 | 0.0% | pure fn | low | test (already tested by TestFormatSize_FullRange — coverage tool mismatch) |
| `init` | undo.go:28 | 0.0% | boilerplate | — | skip |
| `runUndoWithDeps` | undo.go:36 | 0.0% | unit | low | test (inject mock IsRepo returning false → error) |

### cmd/ Summary

- **0% functions that are testable thin wrappers**: `runBackup`, `runLogin`, `runPick`, `runPull`, `runPush` — all 1-line delegations. Testing these requires just calling them with a cobra.Command and mock deps.
- **0% functions with real logic**: `runLoginWithDeps`, `runPickWithDeps`, `runPullWithDeps`, `runPushWithDeps`, `runRestoreWithDeps`, `runUndoWithDeps` — all follow the *WithDeps pattern and can be tested with injected mocks.
- **TUI wrappers**: `runLoginInteractiveWithDeps`, `runPickWithDeps` — can test the non-TTY error path and inject mock Wizard/Picker functions.
- **Skip**: All `init()` functions (tested indirectly by flag/structure tests), `Execute()` (os.Exit path).

---

## actions/ Detailed Analysis

| Function | File:Line | Current % | Category | Effort | Recommendation |
|----------|-----------|-----------|----------|--------|----------------|
| `Run` (BackupAction) | backup.go:49 | 85.0% | error-path | med | partial (test adapter registration error path) |
| `saveManifest` | backup.go:232 | 85.7% | error-path | low | test (inject writeFailingFS) |
| `scanBackupForSecrets` | backup.go:246 | 64.7% | unit | med | test (add test with files containing/missing secrets, walk error path) |
| `RunExport` | export.go:17 | 57.7% | error-path | low | test (create output file error, cleanup on CreateTarGz error) |
| `CreateTarGz` | export.go:69 | 66.7% | error-path | med | test (gzip close error, unreadable file during walk) |
| `FormatSizeBytes` | list_local.go:102 | 50.0% | pure fn | low | test (table-driven: B, KB, MB, GB ranges — only KB tested) |
| `Run` (PickBackupAction) | pick_backup.go:83 | 62.8% | mock | med | test (Picker returns error, empty selection, BakDir/HomeDir errors) |
| `ProfileCreateInteractive` | profile.go:215 | 0.0% | unit | low | test (not TUI — accepts ProfileCreateFromWizard struct + io.Writer) |
| `defaultConfigLoad` | mock_impl.go:10 | 0.0% | boilerplate | — | skip (fallback only, overridden in production) |
| `UserHomeDir` | os_impl.go:17 | 0.0% | OS wrapper | — | skip (trivial os.UserHomeDir wrapper) |
| `MkdirAll` | os_impl.go:49 | 66.7% | OS wrapper | low | partial (error path hard to trigger without permission tricks) |
| `CopyFile` | os_impl.go:56 | 80.0% | OS wrapper | — | skip (at threshold) |
| `RemoveAll` | os_impl.go:80 | 66.7% | OS wrapper | — | skip |
| `WalkDir` | os_impl.go:87 | 66.7% | OS wrapper | — | skip |
| `WriteFile` | os_impl.go:94 | 66.7% | OS wrapper | low | partial (error path with read-only dir) |
| `Load` (RealConfigLoader) | os_impl.go:113 | 0.0% | OS wrapper | — | skip (delegates to configLoad variable) |
| `Run` (RestoreAction) | restore.go:48 | 68.9% | error-path | med | partial (test confirmation prompt "n" → cancel, validation fail + --force) |
| `sched` | schedule.go:28 | 66.7% | unit | low | test (nil NewScheduler → default path) |

### actions/ Summary

- **Quick wins (< 30 min each)**: `FormatSizeBytes` (pure function table test), `ProfileCreateInteractive` (accepts struct, not TUI), `saveManifest` (inject failing FS), `sched` (test nil fallback).
- **Medium effort**: `scanBackupForSecrets` (needs fixture files with/without patterns), `Run` PickBackupAction (inject mock Picker/BakDir/HomeDir), `Run` RestoreAction (confirmation prompt test).
- **Skip**: `defaultConfigLoad` (unused fallback), `UserHomeDir` (trivial OS wrapper), `Load` (delegates to variable).

---

## Bubbletea Models

### pickModel (`cmd/pick.go`)
- **State**: `items []categoryItem`, `cursor int`, `quitting bool`, `confirmed bool`
- **Messages**: `tea.KeyMsg` — q/esc/ctrl+c (quit), up/k (cursor up), down/j (cursor down), space (toggle), enter (confirm)
- **View()**: Renders title, cursor list with [ ]/[x] checkboxes, help text
- **Tested**: ✅ Init, Update (quit, cursor up/down, toggle, confirm), View, Selected — all well-covered

### wizardModel (`cmd/wizard.go`)
- **State**: `step wizardStep`, `mode string`, `providers/presets []string`, `adapterItems/categoryItems []toggleItem`, cursors, `selectedProvider/selectedPreset string`
- **Messages**: `tea.KeyMsg` — ctrl+c/esc (quit), enter (advance step), up/k/down/j (navigate), space (toggle adapters/categories)
- **Steps**: stepProvider → stepPreset → stepAdapters → stepCategories → stepConfirm
- **View()**: Renders step-specific title, list, and confirm summary
- **Tested**: ✅ Init, step transitions, Ctrl+C/Esc exit, View (title + quitting empty), provider cursor navigation

---

## E2E Coverage Already Provided

| Script | Commands Tested |
|--------|----------------|
| `backup_restore_roundtrip.txtar` | `bak backup --preset quick`, `bak restore --help`, `bak restore --dry-run <bad-id>` |
| `backup_verify_roundtrip.txtar` | `bak backup --preset quick/full`, `bak list` |
| `diff_two_backups.txtar` | `bak backup`, `bak diff --help`, `bak diff <bad-ids>` |
| `profile_create_list.txtar` | `bak profile create/show/list/delete`, duplicate detection |
| `schedule_create_list.txtar` | `bak schedule --help`, `bak schedule create` (missing flag, invalid interval, missing profile), `bak schedule list` |
| `roundtrip_test.go` | Full backup→restore→checksum verification (quick + skills presets) |

### E2E Gaps
- `bak export` — not tested
- `bak undo` — not tested
- `bak login` — not tested (requires token)
- `bak pick` — not tested (requires TTY)
- `bak push` / `bak pull` — not tested (requires token)
- `bak verify` — not tested as standalone command

---

## Test Plan

### cmd/ New Tests (~8 tests)

| # | Test | Target | Pattern |
|---|------|--------|---------|
| 1 | `TestRunBackup_Delegation` | `runBackup` | Call runBackup with mock cmd, verify it reaches runBackupWithDeps |
| 2 | `TestRunLogin_Delegation` | `runLogin` | Same pattern |
| 3 | `TestRunPick_Delegation` | `runPick` | Same pattern |
| 4 | `TestRunPull_Delegation` | `runPull` | Same pattern |
| 5 | `TestRunPush_Delegation` | `runPush` | Same pattern |
| 6 | `TestRunLoginWithDeps_NonInteractive` | `runLoginWithDeps` | Mock ConfigLoader + TokenValidator, test happy path |
| 7 | `TestRunLoginInteractiveWithDeps_NoTTY` | `runLoginInteractiveWithDeps` | Override isTTY → false, expect error |
| 8 | `TestRunPickWithDeps_NoTTY` | `runPickWithDeps` | Override isTTY → false, expect error |
| 9 | `TestRunUndoWithDeps_NotRepo` | `runUndoWithDeps` | Inject IsRepo=false, expect error |
| 10 | `TestRunRestoreWithDeps_InvalidID` | `runRestoreWithDeps` | Pass invalid backup ID, expect format error |
| 11 | `TestRunListWithDeps_Local` | `runListWithDeps` | Set provider="", exercise local path with mock BakDir |

### actions/ New Tests (~10 tests)

| # | Test | Target | Pattern |
|---|------|--------|---------|
| 1 | `TestFormatSizeBytes_AllRanges` | `FormatSizeBytes` | Table-driven: 0 B, 500 B, 1 KB, 1.5 MB, 2.5 GB |
| 2 | `TestProfileCreateInteractive_Cancelled` | `ProfileCreateInteractive` | Confirmed=false → "cancelled" message |
| 3 | `TestProfileCreateInteractive_HappyPath` | `ProfileCreateInteractive` | Confirmed=true → profile saved |
| 4 | `TestScanBackupForSecrets_WithFixtures` | `scanBackupForSecrets` | Create files with/without secret patterns |
| 5 | `TestSaveManifest_WriteError` | `saveManifest` | Inject writeFailingFS |
| 6 | `TestPickBackupAction_PickerError` | `PickBackupAction.Run` | Mock Picker returns error |
| 7 | `TestPickBackupAction_NotConfirmed` | `PickBackupAction.Run` | Mock Picker returns Confirmed=false |
| 8 | `TestPickBackupAction_EmptySelection` | `PickBackupAction.Run` | Mock Picker returns empty Selected |
| 9 | `TestRestoreAction_ConfirmCancel` | `RestoreAction.Run` | Stdin="n\n", verify cancelled |
| 10 | `TestSched_DefaultFallback` | `ScheduleAction.sched` | nil NewScheduler → real scheduler |

### E2E New Tests (~2 scripts)

| # | Script | Commands |
|---|--------|----------|
| 1 | `export_roundtrip.txtar` | `bak backup` → `bak export <id>` → verify tar.gz exists |
| 2 | `undo_after_restore.txtar` | `bak backup` → `bak restore --force` → `bak undo` → verify revert |

### Estimated Total Effort

| Area | Tests | Effort |
|------|-------|--------|
| cmd/ unit | 11 | Medium (2-3h) |
| actions/ unit | 10 | Medium (2-3h) |
| E2E scripts | 2 | Low (1h) |
| **Total** | **23** | **5-7h** |

### Expected Coverage After Implementation

- **cmd/**: 46.6% → ~70-75% (remaining gaps: init functions, Execute/os.Exit, TUI Program.Run paths)
- **actions/**: 82.9% → ~88-90% (remaining gaps: OS wrappers error paths, RealConfigLoader)

---

## Approaches

### 1. Unit-First (Recommended)
Add unit tests for all *WithDeps functions in cmd/ and gap-fill actions/ error paths. E2E scripts for export and undo.
- **Pros**: Fast feedback, no binary builds, uses existing test patterns (setupTestDeps, MockFileSystem)
- **Cons**: Won't cover os.Exit or real TUI execution
- **Effort**: Medium

### 2. E2E-Heavy
Add testscript scripts for every untested command (export, undo, login --help, etc.)
- **Pros**: Exercises full binary, catches wiring issues
- **Cons**: Slow (binary build per test), can't test error injection easily, login/push/pull need tokens
- **Effort**: High

### 3. TUI teatest Tests
Add teatest-based tests for pickModel and wizardModel interactive flows.
- **Pros**: Full TUI coverage
- **Cons**: Already well-covered by direct Update() tests, teatest is heavier, AGENTS.md prefers Update() testing
- **Effort**: High

---

## Recommendation

**Approach 1 (Unit-First)** with selective E2E (approach 2) for export and undo. This follows existing patterns, is fastest, and gets the most coverage per effort. TUI models are already well-tested via direct Update()/View() calls per AGENTS.md rules.

---

## Risks

- **`runLoginWithDeps`** requires a mock `TokenValidator` and `ConfigSaver` — the `cloud.ValidateToken` function makes HTTP calls. Need to inject a mock validator.
- **`runPickWithDeps` and `runLoginInteractiveWithDeps`** use `isTTY()` which reads `os.Stdin.Fd()` — not injectable without refactoring. Can only test the non-TTY error path.
- **`runPushWithDeps` and `runPullWithDeps`** wire `&actions.RealProviderFactory{}` directly — need to refactor to inject a mock factory for unit testing, OR test at the E2E level.
- **`formatSize`** in root.go shows 0% but IS tested by `TestFormatSize_FullRange` — this may be a coverage profile artifact from how `go test ./cmd/...` collects coverage.

---

## Ready for Proposal

**Yes.** The exploration identifies 23 concrete tests across cmd/, actions/, and E2E with clear effort estimates. The orchestrator should tell the user:

> "Coverage analysis complete. cmd/ is at 46.6% (mostly untested delegation wrappers), actions/ at 82.9% (gap-fill needed). Plan: 11 cmd/ unit tests, 10 actions/ unit tests, 2 E2E scripts. Estimated 5-7h effort, targeting ~70-75% cmd/ and ~88-90% actions/. Ready to proceed with proposal?"
