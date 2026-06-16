# Apply Progress — tui-wiring-gaps

**Phase**: 1 + 2 + 3 + 4 + **5** (Toast Wiring + Search → Dashboard Wiring + Menu Items 1 & 4 + ScreenWizard Removal + Action Dispatch)
**Mode**: Strict TDD
**Date**: 2026-06-15

## Completed Tasks

### Phase 1: Toast Wiring
- [x] 1.1 Write test: `actionResultMsg{err: nil}` → `toast.Show()` with success message
- [x] 1.2 Write test: `actionResultMsg{err: error}` → `toast.Show()` with error text
- [x] 1.3 Define `actionResultMsg` struct in `internal/tui/model.go`
- [x] 1.4 Add `case actionResultMsg:` handler in `Model.Update()`
- [x] 1.5 Quality gates: `go test -race ./...`, `go vet ./...` (golangci-lint not installed)

### Phase 2: Search → Dashboard Wiring
- [x] 2.1 Write test: `SetFilter("conf")` returns only matching rows (case-insensitive)
- [x] 2.2 Write test: `SetFilter("")` restores all original rows
- [x] 2.3 Write test: `SetFilter("xyz")` with no matches shows empty table rows
- [x] 2.4 Add `allRows []table.Row` field to `DashboardModel`; populate in `NewDashboardModel`
- [x] 2.5 Implement `SetFilter(query string)` on `DashboardModel`
- [x] 2.6 Write test: search query change triggers dashboard filter
- [x] 2.7 Wire search forwarding in `Model.Update()` for `ScreenDashboard`
- [x] 2.8 Quality gates: `go test -race ./...`, `go vet ./...`

### Phase 3: Menu Items 1 & 4
- [x] 3.1 Write test: cursor=1 + enter → toast shows "coming soon" message
- [x] 3.2 Write test: cursor=4 + enter → toast shows "coming soon" message
- [x] 3.3 Add `case 1:` and `case 4:` in `handleMenuEnter()` in `internal/tui/model.go`
- [x] 3.4 Quality gates: `go test -race ./...`, `go vet ./...`

### Phase 4: ScreenWizard Removal
- [x] 4.1 Write test: `TestScreenIotaValues` verifies Screen enum values match expected sequence after removal (ScreenMenu=0 through ScreenHealth=6, no ScreenWizard)
- [x] 4.2 Remove `ScreenWizard` from iota enum in `internal/tui/model.go`; remove 3 tests referencing it; verify zero code references
- [x] 4.3 Quality gates: `go test -race ./...`, `go vet ./...`

### Phase 5: Action Dispatch
- [x] 5.1 Write test in `internal/tui/dispatch_test.go`: table-driven test for `RouteSelection(sel MenuSelection, deps Deps) error` — 5 cases (cursor 0 Backup, cursor 1 Restore no-op, cursor 6 Quit no-op, empty selection no-op, error propagation)
- [x] 5.2 Implement `RouteSelection()` as a pure function in `internal/tui/dispatch.go` — checks empty selection, routes cursor 0 to `deps.RunBackup`, returns error if action fails
- [x] 5.3 Update `cmd/tty.go` `defaultRunTUI`: capture `finalModel` from `p.Run()`, type-assert to `tui.Model`, call `tui.RouteSelection(model.Selection(), deps)`
- [x] 5.4 Quality gates: `go test -race ./...` (28/28 pkgs), `go vet ./...` (clean), `golangci-lint run` (0 issues)

## TDD Cycle Evidence

| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|------|-----------|-------|------------|-----|-------|-------------|----------|
| 1.1 | `internal/tui/model_test.go` | Unit | ✅ 28/28 pkgs | ✅ Written (compile fail) | ✅ Passed | ✅ 2 paths (nil/error) | ➖ None needed |
| 1.2 | `internal/tui/model_test.go` | Unit | ✅ 28/28 pkgs | ✅ Written (compile fail) | ✅ Passed | — (paired with 1.1) | ➖ None needed |
| 1.3 | `internal/tui/model.go` | — | ✅ 28/28 pkgs | N/A (struct def) | ✅ Compiles | — | ➖ None needed |
| 1.4 | `internal/tui/model.go` | — | ✅ 28/28 pkgs | N/A (handler) | ✅ Compiles | — | ➖ None needed |
| 1.5 | — | Quality | — | — | ✅ 28/28 pkgs | — | — |
| 2.1 | `internal/tui/screens/dashboard_test.go` | Unit | ✅ 28/28 pkgs | ✅ Written (compile fail: SetFilter undefined) | ✅ Passed | ✅ 3 filter tests | ➖ None needed |
| 2.2 | `internal/tui/screens/dashboard_test.go` | Unit | ✅ 28/28 pkgs | ✅ Written (compile fail: SetFilter undefined) | ✅ Passed | — (paired with 2.1) | ➖ None needed |
| 2.3 | `internal/tui/screens/dashboard_test.go` | Unit | ✅ 28/28 pkgs | ✅ Written (compile fail: SetFilter undefined) | ✅ Passed | — (paired with 2.1) | ➖ None needed |
| 2.4 | `internal/tui/screens/dashboard.go` | — | ✅ 28/28 pkgs | N/A (field add) | ✅ Compiles | — | ➖ None needed |
| 2.5 | `internal/tui/screens/dashboard.go` | — | ✅ 28/28 pkgs | N/A (method impl) | ✅ Compiles | — | ➖ None needed |
| 2.6 | `internal/tui/model_test.go` | Unit | ✅ 28/28 pkgs | ✅ Written (test fails: Query()="", no filter) | ✅ Passed | ✅ 2 paths (filter + Esc restore) | ➖ None needed |
| 2.7 | `internal/tui/model.go` | — | ✅ 28/28 pkgs | N/A (wiring) | ✅ Compiles | — | ➖ None needed |
| 2.8 | — | Quality | — | — | ✅ 28/28 pkgs | — | — |
| 3.1 | `internal/tui/model_test.go` | Unit | ✅ 64/64 tests (tui pkg) | ✅ Written (toast empty after enter) | ✅ Passed | ✅ 2 menu items | ➖ None needed |
| 3.2 | `internal/tui/model_test.go` | Unit | ✅ 64/64 tests (tui pkg) | ✅ Written (toast empty after enter) | ✅ Passed | — (paired with 3.1) | ➖ None needed |
| 3.3 | `internal/tui/model.go` | — | ✅ 64/64 tests (tui pkg) | N/A (case branches) | ✅ Compiles | — | ➖ None needed |
| 3.4 | — | Quality | — | — | ✅ 19/19 pkgs | — | — |
| 4.1 | `internal/tui/model_test.go` | Unit | ✅ 4/4 tui pkgs | ✅ Written (ScreenSettings=4 want 3) | ✅ Passed | ➖ Skipped: purely structural (iota enum value verification) | ➖ None needed |
| 4.2 | `internal/tui/model.go` | — | ✅ 4/4 tui pkgs | N/A (constant removal) | ✅ Compiles + no code refs | — | ➖ None needed |
| 4.3 | — | Quality | — | — | ✅ 28/28 pkgs | — | — |
| 5.1 | `internal/tui/dispatch_test.go` | Unit | ✅ 28/28 pkgs | ✅ Written (compile fail: undefined: RouteSelection) | ✅ Passed (5/5 subtests) | ✅ 5 paths (c0/c1/c6/empty/err) | ➖ None needed |
| 5.2 | `internal/tui/dispatch.go` | — | ✅ 28/28 pkgs | N/A (pure function) | ✅ Compiled + all tests green | — | ✅ switch→if for gocritic singleCaseSwitch |
| 5.3 | `cmd/tty.go` | — | ✅ 28/28 pkgs | N/A (wiring) | ✅ Compiled + all cmd tests green | — | ➖ None needed |
| 5.4 | — | Quality | — | — | ✅ 28/28 pkgs + 0 lint issues | — | — |

### RED Details (Phase 4)

#### Task 4.1: Screen iota values shifted by ScreenWizard
New `TestScreenIotaValues` asserted the expected sequence after ScreenWizard removal (0,1,2,3,4,5,6). With ScreenWizard still present at position 3, the subsequent constants were all shifted by +1:
```
--- FAIL: TestScreenIotaValues (0.00s)
    --- FAIL: TestScreenIotaValues/ScreenSettings (0.00s)
        model_test.go:1322: ScreenSettings = 4, want 3
    --- FAIL: TestScreenIotaValues/ScreenCloud (0.00s)
        model_test.go:1322: ScreenCloud = 5, want 4
    --- FAIL: TestScreenIotaValues/ScreenShortcuts (0.00s)
        model_test.go:1322: ScreenShortcuts = 6, want 5
    --- FAIL: TestScreenIotaValues/ScreenHealth (0.00s)
        model_test.go:1322: ScreenHealth = 7, want 6
```

### GREEN Details (Phase 4)

After removing `ScreenWizard` from the iota enum and deleting 3 wizard-referencing tests:
```
=== RUN   TestScreenIotaValues
--- PASS: TestScreenIotaValues (0.00s)
    --- PASS: TestScreenIotaValues/ScreenMenu (0.00s)
    --- PASS: TestScreenIotaValues/ScreenDashboard (0.00s)
    --- PASS: TestScreenIotaValues/ScreenProgress (0.00s)
    --- PASS: TestScreenIotaValues/ScreenSettings (0.00s)
    --- PASS: TestScreenIotaValues/ScreenCloud (0.00s)
    --- PASS: TestScreenIotaValues/ScreenShortcuts (0.00s)
    --- PASS: TestScreenIotaValues/ScreenHealth (0.00s)
```

Full tui suite: 4/4 packages pass. Zero code references to `ScreenWizard` (only 1 comment describing the removal).

### TRIANGULATE (Phase 4)

Triangulation skipped: purely structural task (iota enum value verification). The enum has exactly ONE possible valid sequence after removal. Remaining test `TestModel_View_UnknownScreen` using `Screen(99)` provides orthogonal coverage for out-of-range screen handling.

### REFACTOR (Phase 4)

No refactoring needed. The removal is minimal:
- 2 lines removed from `internal/tui/model.go` (constant + comment)
- 3 tests removed from `internal/tui/model_test.go` (TestModel_View_Wizard, TestModel_Update_ScreenChange_Wizard, TestModel_Update_Wizard_Key_Noop)
- 1 new test added (TestScreenIotaValues — 7 sub-cases, table-driven)
- 2 comment lines updated (section headers no longer reference ScreenWizard)

### RED Details (Phase 5)

#### Task 5.1: RouteSelection function does not exist
New `TestRouteSelection` in `internal/tui/dispatch_test.go` referenced `RouteSelection(sel, deps)` which had no implementation:
```
internal/tui/dispatch_test.go:67:11: undefined: RouteSelection
FAIL	github.com/danielxxomg/bak-cli/internal/tui [build failed]
```
5 test cases defined: cursor 0 (calls RunBackup), cursor 1 (no-op), cursor 6 (no-op), empty selection (no-op), error propagation.

### GREEN Details (Phase 5)

After implementing `RouteSelection()` in `internal/tui/dispatch.go`:
```
=== RUN   TestRouteSelection
--- PASS: TestRouteSelection (0.00s)
    --- PASS: TestRouteSelection/cursor_0_Backup_calls_RunBackup (0.00s)
    --- PASS: TestRouteSelection/cursor_1_Restore_no-op (0.00s)
    --- PASS: TestRouteSelection/cursor_6_Quit_no-op (0.00s)
    --- PASS: TestRouteSelection/empty_selection_no-op (0.00s)
    --- PASS: TestRouteSelection/cursor_0_propagates_RunBackup_error (0.00s)
```

Full tui suite: 4/4 packages pass with `-race`. All 28 project packages pass with `-race`.

`cmd/tty.go` `defaultRunTUI` updated to capture final model and call `tui.RouteSelection()`. All existing cmd tests (including `TestRunTUI_InjectionPoint`, `TestRunTUI_PropagatesError`, `TestRunTUI_Initialized`) pass unchanged.

### REFACTOR (Phase 5)

Single refactoring: `switch sel.Cursor { case 0: ... }` → `if sel.Cursor == 0 { ... }` to satisfy `gocritic` `singleCaseSwitch` lint rule. Tests re-ran and all 5/5 still pass.

## Files Changed

| File | Action | Lines | What Was Done |
|------|--------|-------|---------------|
| `internal/tui/model.go` | Modified (Ph1) | +18 | `actionResultMsg` struct + case handler in `Update()` |
| `internal/tui/model.go` | Modified (Ph2) | +22 | Search forwarding in `handleKey` ScreenDashboard: search-active path + SetFilter calls |
| `internal/tui/model.go` | Modified (Ph3) | +6 | `case 1:` and `case 4:` in `handleMenuEnter()` — toast feedback for Restore and Profiles |
| `internal/tui/model.go` | Modified (Ph4) | −2 | Removed `ScreenWizard` constant from Screen iota enum; remaining constants auto-renumbered |
| `internal/tui/model_test.go` | Modified (Ph1) | +45 | `errors` import; `TestModel_Update_ActionResult_Success/Error` |
| `internal/tui/model_test.go` | Modified (Ph2) | +109 | `TestModel_Update_SearchForwardsToDashboard` + `TestModel_Update_SearchEscRestoresAllRows` |
| `internal/tui/model_test.go` | Modified (Ph3) | ±30 | Updated `TestModel_Update_MenuEnter_Restore/Profiles` from old nil-cmd assertions to toast visibility assertions |
| `internal/tui/model_test.go` | Modified (Ph4) | +25 / −48 | Added `TestScreenIotaValues` (7 sub-cases); removed 3 wizard tests (TestModel_View_Wizard, TestModel_Update_ScreenChange_Wizard, TestModel_Update_Wizard_Key_Noop) |
| `internal/tui/screens/dashboard.go` | Modified | +31 | `allRows` field; `SetFilter(query)` method (16 lines) |
| `internal/tui/screens/dashboard_test.go` | Modified | +119 | 3 `SetFilter` tests + helpers (`rowIDs`, `contains`) + `table` import |
| `openspec/changes/tui-wiring-gaps/tasks.md` | Modified | 20 boxes | Marked tasks 1.1–1.5, 2.1–2.8, 3.1–3.4, 4.1–4.3, 5.1–5.4 as `[x]` |
| `internal/tui/dispatch.go` | Created (Ph5) | +22 | `RouteSelection()` pure function — routes MenuSelection to Deps actions |
| `internal/tui/dispatch_test.go` | Created (Ph5) | +81 | Table-driven test for `RouteSelection` — 5 cases: backup call, restore no-op, quit no-op, empty no-op, error propagation |
| `cmd/tty.go` | Modified (Ph5) | +8 | `defaultRunTUI` captures `finalModel` from `p.Run()`, dispatches via `tui.RouteSelection()` |

## Test Summary (Cumulative)

- **Total tests written**: 11 (2 Phase 1 + 5 Phase 2 + 2 Phase 3 [modified] + 1 Phase 4 + 1 Phase 5)
- **Total tests removed**: 3 (wizard tests referencing removed ScreenWizard constant)
- **Total tests passing**: 11 (new) + all existing (0 regressions)
- **Layers used**: Unit (11)
- **Pure functions created**: 2 helpers (`rowIDs`, `contains` — test only) + 1 production (`RouteSelection`)
- **Race detector**: Clean (all 28 packages pass with `-race`)

## Quality Gates

| Gate | Result |
|------|--------|
| `go test -race ./...` | ✅ 28/28 packages pass |
| `go vet ./...` | ✅ No warnings |
| `golangci-lint run` | ✅ 0 issues (gocritic singleCaseSwitch resolved via if-statement refactor) |
| Zero `ScreenWizard` code refs | ✅ `rg 'ScreenWizard' --type go internal/` returns 0 matches |
| `go build ./...` | ✅ Clean compilation |

## Deviations from Design

None — implementation matches design.md exactly:
- `RouteSelection()` extracted as a pure function in `internal/tui/dispatch.go`
- Table-driven tests verify cursor routing without touching tea.Program or os.Exit
- `cmd/tty.go` `defaultRunTUI` captures final model from `p.Run()` and dispatches
- Phases 1–4: `ScreenWizard` removed from iota enum, zero code refs, 3 wizard tests removed, 1 new test (`TestScreenIotaValues`) verifies correct sequence

## Issues Found

- `golangci-lint` is installed and passes (0 issues after fixing `singleCaseSwitch` via if-statement refactor)
- Discovery (Phase 2): bubbles/textinput v2 uses `msg.Text` (not `msg.Code`) for printable characters. Test keys must include `Text: string(ch)` to simulate real terminal input.
- Discovery (Phase 5): `Deps` has no `RunRestore` field — cursor 1 (Restore) is a no-op. `RouteSelection` uses `sel.Item == ""` to detect zero-value MenuSelection from empty menuItems, avoiding false match on cursor 0.

## Remaining Phases

- [x] Phase 1: Toast Wiring (tasks 1.1–1.5)
- [x] Phase 2: Search → Dashboard Wiring (tasks 2.1–2.8)
- [x] Phase 3: Menu Items 1 & 4 (tasks 3.1–3.4)
- [x] Phase 4: ScreenWizard Removal (tasks 4.1–4.3)
- [x] Phase 5: Action Dispatch (tasks 5.1–5.4)
