# Proposal: Coverage Improvement (cmd/ + actions/)

## Intent

cmd/ sits at 46.6% coverage — mostly untested cobra delegation wrappers and `*WithDeps` functions. actions/ is at 82.9% with gaps in error paths and pure functions. This change adds **quality tests that exercise real logic**, not coverage padding. Target: cmd/ → 70-75%, actions/ → 88-90%.

## Scope

### In Scope
- 11 unit tests for cmd/ (delegation wrappers + `*WithDeps` with injected mocks)
- 10 unit tests for actions/ (error paths, pure functions, mock injection)
- 2 E2E testscript tests (`export`, `undo` roundtrips)
- Bubbletea model `Update()`/`View()` tests where gaps exist (per AGENTS.md rules)

### Out of Scope
- Testing `bubbletea.Program.Run()` — AGENTS.md forbids it; test `Update()`/`View()` instead
- Testing `os.Exit` paths directly — E2E covers entry points
- Changing production code (except minimal refactoring for mock injection where needed)
- Login/push/pull E2E (require tokens, not feasible in CI without secrets)
- OS wrapper error paths in `os_impl.go` (low value, high effort)

## Capabilities

### New Capabilities
None — this is a testing-only change. No new user-facing behavior.

### Modified Capabilities
None — spec-level requirements are unchanged. Coverage improvements are implementation quality, not behavioral changes.

## Approach

**Unit-first with selective E2E** (recommended by exploration):

1. **cmd/ thin wrappers** — test `runBackup`, `runLogin`, `runPick`, `runPull`, `runPush` delegation (call with mock cmd, verify reach `*WithDeps`)
2. **cmd/ `*WithDeps` functions** — inject mock deps (`ConfigLoader`, `TokenValidator`, `IsRepo`, `BakDir`) to test non-interactive paths and error handling
3. **cmd/ TUI guards** — override `isTTY` → false for `runLoginInteractiveWithDeps` and `runPickWithDeps`, verify non-TTY error
4. **actions/ error paths** — `saveManifest` (writeFailingFS), `scanBackupForSecrets` (fixture files), `RunExport` (create error), `CreateTarGz` (gzip close error)
5. **actions/ pure functions** — `FormatSizeBytes` table-driven (B, KB, MB, GB), `ProfileCreateInteractive` (struct input, not TUI)
6. **actions/ mock flows** — `PickBackupAction.Run` (Picker error/not-confirmed/empty), `RestoreAction.Run` (stdin="n\n" cancel)
7. **E2E scripts** — `export_roundtrip.txtar` (backup → export → verify tar.gz), `undo_after_restore.txtar` (backup → restore --force → undo → verify revert)

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `cmd/*_test.go` | New | 11 test functions covering delegation + WithDeps + TUI guards |
| `internal/actions/*_test.go` | New | 10 test functions covering error paths + pure functions + mock flows |
| `tests/e2e/*.txtar` | New | 2 testscript files for export and undo |
| `cmd/login.go` | Modified | May need `isTTY` variable extraction for injection (minimal refactor) |
| `cmd/pick.go` | Modified | Same `isTTY` extraction if not already done |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| `isTTY()` not injectable without refactoring | Medium | Extract to package-level `var isTTY = ...` variable, override in tests |
| `runPushWithDeps`/`runPullWithDeps` wire `RealProviderFactory` directly | Medium | Refactor to accept factory via deps struct, or defer to E2E |
| `runLoginWithDeps` needs mock `TokenValidator` (HTTP calls) | Low | Existing `cloud.MockTokenValidator` or inject via deps |
| Coverage tool mismatch (`formatSize` shows 0% but is tested) | Low | Verify with `go test -coverprofile` after adding tests; may resolve itself |

## Rollback Plan

All changes are test-only additions. Rollback = `git revert` of the test commit(s). No production code changes to worry about. If minimal refactoring is needed (isTTY extraction, factory injection), those are small isolated changes that revert cleanly.

## Dependencies

- Existing test infrastructure: `setupTestDeps()`, `MockFileSystem`, `MockConfigLoader`
- Existing E2E infrastructure: testscript framework in `tests/e2e/`
- AGENTS.md testing rules (already updated with bubbletea + os.Exit guidance)

## Success Criteria

- [ ] cmd/ coverage ≥ 70% (from 46.6%)
- [ ] actions/ coverage ≥ 88% (from 82.9%)
- [ ] All 23 new tests pass on 3-OS CI matrix
- [ ] No production code behavior changes
- [ ] No `Program.Run()` or `os.Exit` direct tests
- [ ] `go test ./...` completes in < 60s
