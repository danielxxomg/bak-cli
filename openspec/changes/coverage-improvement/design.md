# Design: Coverage Improvement (cmd/ + actions/)

## Technical Approach

Add quality tests that exercise real logic in cmd/ (46.6% → 70-75%) and actions/ (82.9% → 88-90%). Strategy: **unit-first with selective E2E**. Test cmd/ thin wrappers via existing `cmdDeps` pattern, inject mocks into `*WithDeps` functions, test bubbletea model `Update()`/`View()` as pure functions, cover actions/ error paths with filesystem mocks, and add 2 E2E testscripts for export/undo roundtrips.

## Architecture Decisions

### Decision: isTTY Injection Strategy

**Choice**: Extract `isTTY` to package-level variable `var isTTY = func() bool { ... }` for test override.

**Alternatives considered**: 
- Pass isTTY via cmdDeps — rejected: breaks existing pattern, isTTY is a system capability not a dependency
- Skip TUI guard tests — rejected: leaves error paths untested

**Rationale**: Minimal refactor (1 line change in wizard.go), follows existing `var execCommand` pattern in AGENTS.md, allows tests to override without changing function signatures.

### Decision: Provider Factory Testing

**Choice**: Defer runPush/runPull unit tests to E2E; test actions.PushAction/PullAction directly with mock factories.

**Alternatives considered**:
- Refactor cmdDeps to accept Factory — rejected: cmd/ should not know about ProviderFactory (architecture boundary violation)
- Mock HTTP layer — rejected: too complex for unit tests, E2E covers this better

**Rationale**: cmd/ wrappers are thin delegation (1-2 lines). Testing actions/ directly with injected mocks provides better coverage with less coupling. E2E tests verify the full wiring.

### Decision: Bubbletea Model Testing

**Choice**: Test `Update()`/`View()` as pure functions — feed `tea.KeyMsg`, assert model state changes and view output.

**Alternatives considered**:
- Test via `tea.Program` — rejected: AGENTS.md forbids `Program.Run()` in unit tests
- Skip model tests — rejected: models contain business logic (selection, navigation)

**Rationale**: Models are pure functions (input msg → output model). Easy to test, high value. Existing wizard_test.go establishes pattern; extend to pickModel.

## Data Flow

```
Test Setup                    cmd/ Wrapper                 actions/ Action
────────────                  ─────────────                ───────────────
setupTestDeps(t)  ──→  runLoginWithDeps(cmd, args, deps)  ──→  LoginAction{
  mock ConfigLoader          deps.ConfigLoader()                 ConfigLoader: deps.ConfigLoader
  bytes.Buffer stdout        deps.Stdout                         Stdout: deps.Stdout
  bytes.Buffer stdin         actions.LoginAction{...}            Stdin: deps.Stdin
                           action.Run(provider, stdout)        }
                                                               action.Run(loginProvider, out)

E2E testscript              bak binary                   actions/ + real FS
──────────────              ─────────                    ──────────────────
exec bak export <id>  ──→  runExportWithDeps()    ──→   actions.RunExport()
stdout 'Exported'            deps.Stdout                  os.Stat, CreateTarGz
exists output.tar.gz         actions.RunExport(...)       writes tar.gz
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `cmd/wizard.go` | Modify | Extract `isTTY` to package-level var for test injection |
| `cmd/login_test.go` | Create | Test `runLoginWithDeps` (config error, non-interactive path, TUI guard) |
| `cmd/pick_test.go` | Modify | Add `pickModel.Update()`/`View()` tests (navigation, selection, quit) |
| `cmd/push_test.go` | Modify | Add `runPushWithDeps` delegation test (verify action wired correctly) |
| `cmd/pull_test.go` | Modify | Add `runPullWithDeps` delegation test |
| `cmd/backup_test.go` | Modify | Add `runBackupWithDeps` delegation test (verify action invocation) |
| `internal/actions/export_test.go` | Create | Test `RunExport` (invalid ID, missing backup, create error), `CreateTarGz` (gzip close error) |
| `internal/actions/pick_backup_test.go` | Create | Test `PickBackupAction.Run` (Picker error, not-confirmed, empty selection) |
| `internal/actions/restore_test.go` | Modify | Add `RestoreAction.Run` with stdin="n\n" cancel path |
| `tests/e2e/testdata/export_roundtrip.txtar` | Create | backup → export → verify tar.gz exists and contains manifest |
| `tests/e2e/testdata/undo_after_restore.txtar` | Create | backup → restore --force → undo → verify git revert commit |

## Interfaces / Contracts

### isTTY Variable (cmd/wizard.go)

```go
// Before:
func isTTY() bool {
    return isatty.IsTerminal(os.Stdin.Fd())
}

// After:
var isTTY = func() bool {
    return isatty.IsTerminal(os.Stdin.Fd())
}
```

Tests override: `defer func(orig func() bool) { isTTY = orig }(isTTY); isTTY = func() bool { return false }`

### Mock Picker (actions/pick_backup_test.go)

```go
mockPicker := func(categories []CategoryItem) (PickResult, error) {
    return PickResult{
        Selected:  []string{"skills", "config"},
        Confirmed: true,
    }, nil
}

action := &PickBackupAction{
    Stdout:  io.Discard,
    Picker:  mockPicker,
    BakDir:  func() (string, error) { return t.TempDir(), nil },
    HomeDir: func() (string, error) { return t.TempDir(), nil },
}
```

### writeFailingFS for Export Tests (actions/export_test.go)

```go
type writeFailingFS struct {
    *OSFileSystem
    home string
}

func (w *writeFailingFS) Create(name string) (*os.File, error) {
    return nil, errors.New("permission denied")
}
```

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit (cmd/) | Delegation wrappers (`runBackup`, `runLogin`, `runPick`, `runPush`, `runPull`) | Call `runXWithDeps` with `setupTestDeps`, verify action invoked (mock ConfigLoader, check stdout) |
| Unit (cmd/) | TUI guards (non-TTY error) | Override `isTTY` → false, call `runLoginInteractiveWithDeps`/`runPickWithDeps`, assert error contains "TTY" |
| Unit (cmd/) | Bubbletea models (`pickModel`, `wizardModel`) | Feed `tea.KeyMsg` to `Update()`, assert cursor/selection state; call `View()`, assert output contains expected strings |
| Unit (actions/) | Error paths (`RunExport`, `CreateTarGz`, `saveManifest`) | Inject `writeFailingFS`, assert error contains context ("create output file", "save manifest") |
| Unit (actions/) | Mock flows (`PickBackupAction.Run`, `RestoreAction.Run`) | Inject mock Picker (error/not-confirmed/empty), stdin="n\n" for restore cancel |
| Unit (actions/) | Pure functions (`FormatSizeBytes`, `IsValidBackupID`) | Table-driven tests (already partially done in cmd/list_test.go, extend to actions/) |
| Integration | cmd/ → actions/ wiring | Existing tests cover this; no new integration tests needed |
| E2E | Export roundtrip | testscript: `exec bak backup --preset quick` → `exec bak export <id> --output out.tar.gz` → `exists out.tar.gz` |
| E2E | Undo after restore | testscript: backup → restore --force → undo → `exec git log` (verify revert commit) |

## Migration / Rollout

No migration required. All changes are test additions. The only production code change is extracting `isTTY` to a variable (1 line, backward-compatible).

**Rollback**: `git revert` of test commits. If `isTTY` refactor causes issues, revert that commit separately.

## Open Questions

- [ ] **Resolved**: `isTTY` extraction is minimal (1 line) and follows existing patterns. Proceed with variable extraction.
- [ ] **Deferred**: `runPushWithDeps`/`runPullWithDeps` wire `RealProviderFactory` directly. Testing these at cmd/ level adds little value; E2E covers the wiring. Unit tests will focus on actions.PushAction/PullAction with mock factories.
- [ ] **Coverage tooling**: Proposal mentions `formatSize` shows 0% but is tested. Verify with `go test -coverprofile` after adding tests — may be a tooling artifact.

## Review Workload Forecast

**Decision needed before apply**: No  
**Chained PRs recommended**: No  
**400-line budget risk**: Low

**Rationale**: ~23 new tests, most are 10-30 lines each. Total additions ~500-600 lines (tests + minimal refactor). Single PR is acceptable — all changes are test-only except 1-line `isTTY` extraction. Reviewer can verify tests by running `go test ./...` and checking coverage report.
