# Archive Report: tui-wiring-gaps

## Change Summary

**Change**: tui-wiring-gaps
**Status**: PASS — fully implemented, verified, and archived
**Date**: 2026-06-15
**Artifact store**: openspec
**Archive location**: `openspec/changes/tui-wiring-gaps/` (retained in active changes per orchestrator directive)

## Intent

The TUI overhaul implemented all UI components (toast, search, dashboard, menu, progress, settings, health, cloud, shortcuts) but left 5 integration gaps. Components existed in isolation — toast never fired, search didn't filter, two menu items were dead, the wizard screen constant had no implementation, and the TUI selection didn't route to cobra actions. This change wired them together so the TUI is functional end-to-end.

## Phases Implemented

| Phase | Description | Tasks | Status |
|-------|-------------|-------|--------|
| 1 | Toast Wiring | 1.1–1.5 (5) | ✅ Complete |
| 2 | Search → Dashboard Wiring | 2.1–2.8 (8) | ✅ Complete |
| 3 | Menu Items 1 & 4 | 3.1–3.4 (4) | ✅ Complete |
| 4 | ScreenWizard Removal | 4.1–4.3 (3) | ✅ Complete |
| 5 | Action Dispatch | 5.1–5.4 (4) | ✅ Complete |

**Total**: 24/24 tasks complete

## Files Created/Modified

| File | Action | Phases | Description |
|------|--------|--------|-------------|
| `internal/tui/model.go` | Modified | 1, 2, 3, 4 | `actionResultMsg` struct + handler; search forwarding to dashboard; menu cases 1 & 4; ScreenWizard removal |
| `internal/tui/model_test.go` | Modified | 1, 2, 3, 4 | Tests for toast, search forwarding, menu items, screen iota values; removed 3 wizard tests |
| `internal/tui/screens/dashboard.go` | Modified | 2 | `allRows` field + `SetFilter(query)` method |
| `internal/tui/screens/dashboard_test.go` | Modified | 2 | 3 SetFilter tests + test helpers |
| `internal/tui/dispatch.go` | Created | 5 | `RouteSelection()` pure function — routes MenuSelection to Deps actions |
| `internal/tui/dispatch_test.go` | Created | 5 | Table-driven test for RouteSelection — 5 cases |
| `cmd/tty.go` | Modified | 5 | Captures final model from `p.Run()`, dispatches via `tui.RouteSelection()` |

## Spec Deltas Applied

| Delta Section | Action | Main Spec Section | Details |
|---------------|--------|-------------------|---------|
| tui-components (toast) | ADDED | `tui > Toast notification on action completion` | 3 scenarios: success toast, error toast, auto-hide |
| tui-dashboard (search) | ADDED | `tui > Search filters dashboard table rows` | 4 scenarios: matching, no match, clear, empty query |
| tui-main-menu (menu items) | ADDED | `tui > Menu cursor 1 (Restore) has a handler` | 2 scenarios: Restore pressed, Profiles pressed |
| tui-main-menu (wizard) | ADDED | `tui > ScreenWizard constant is resolved` | 1 scenario: wizard constant removed |
| tui-action-dispatch (NEW) | ADDED | `tui > TUI selection routes to cobra actions` | 5 scenarios: backup, restore, browse, quit, out of bounds |

All 5 requirement groups (15 scenarios total) merged into new `tui` capability section in `openspec/specs/bak-cli/spec.md`.

## Quality Gates

| Gate | Result |
|------|--------|
| `go test -race ./...` | ✅ 28/28 packages pass |
| `go vet ./...` | ✅ Clean |
| `golangci-lint run` | ✅ 0 issues |
| `go build ./...` | ✅ Clean compilation |
| Zero `ScreenWizard` code refs | ✅ Only 1 comment reference |

## Discoveries and Lessons Learned

### bubbles/textinput v2 key handling
bubbles/textinput v2 uses `msg.Text` (not `msg.Code`) for printable characters. When writing TUI tests that simulate typing, test keys MUST include `Text: string(ch)` to simulate real terminal input. Using `msg.Code` alone will not produce the expected character in the text input.

### Dispatch extraction pattern
Extracting `RouteSelection()` as a pure function in `internal/tui/dispatch.go` made it trivially testable with table-driven tests — no `tea.Program`, no `os.Exit`, no TTY. The `cmd/tty.go` wrapper just calls `tui.RouteSelection(model.Selection(), deps)`. This pattern should be reused for any future TUI→action routing.

### Deps has no RunRestore field
`Deps` struct currently has no `RunRestore` field, so cursor 1 (Restore) is intentionally a no-op in `RouteSelection`. When restore-from-TUI is desired, extend `Deps` with a `RunRestore` function field and add a case in `RouteSelection`.

### Selection zero-value detection
`RouteSelection` uses `sel.Item == ""` to detect zero-value `MenuSelection` (from empty `menuItems`), avoiding false match on cursor 0. This is important because cursor 0 is a valid selection (Backup) and must not be confused with "no selection."

### gocritic singleCaseSwitch
A single-case `switch` triggers `gocritic`'s `singleCaseSwitch` lint rule. Refactor to `if` statement instead. Minor but easy to miss.

## Verification Issues (non-blocking)

| Severity | Description |
|----------|-------------|
| SUGGESTION | `RouteSelection` only routes cursor 0 (Backup). Cursor 1 (Restore) is a no-op — extend `Deps` when ready. |
| SUGGESTION | `cmd/tty.go` passes `nil, nil` to `RunBackup`; needs real categories/progress channel when TUI drives actual backup. |
| SUGGESTION | `ScreenWizard` still appears in historical comments/docs. Consider cleanup for strict "no string matches." |

## Recommendations for Future Work

1. **RunRestore from TUI**: Add `RunRestore` to `Deps` and wire cursor 1 in `RouteSelection` to trigger the restore flow.
2. **Profiles screen**: Implement a Profiles screen (cursor 4) instead of "coming soon" toast.
3. **Search in other screens**: The search→filter pattern from dashboard could be extended to other table-based screens (cloud backups, profiles).
4. **Real categories in TUI dispatch**: Replace `nil, nil` in `RunBackup` call with actual category selection from the TUI.
5. **Wizard screen reconsideration**: If a first-run wizard is desired in the future, re-add `ScreenWizard` and implement `screens/wizard.go`.

## Archive Contents

- proposal.md ✅
- spec.md ✅ (delta merged into main spec)
- design.md ✅
- tasks.md ✅ (24/24 tasks complete)
- apply-progress.md ✅
- verify-report.md ✅ (PASS)
- archive-report.md ✅ (this file)

## SDD Cycle Complete

The tui-wiring-gaps change has been fully planned, implemented, verified, and archived. All 5 TUI integration gaps are closed. Ready for the next change.
