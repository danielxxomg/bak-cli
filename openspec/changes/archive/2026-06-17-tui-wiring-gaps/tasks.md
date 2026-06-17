# Tasks: TUI Wiring Gaps

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 180–220 |
| 400-line budget risk | Low |
| Chained PRs recommended | No |
| Suggested split | Single PR |
| Delivery strategy | ask-on-risk |
| Chain strategy | pending |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: pending
400-line budget risk: Low

## Phase 1: Toast Wiring (Gap 1)

- [x] 1.1 Write test in `internal/tui/model_test.go`: sending `actionResultMsg{err: nil}` to `Update()` calls `toast.Show()` with success message and TTL > 0
- [x] 1.2 Write test: sending `actionResultMsg{err: errors.New("fail")}` to `Update()` calls `toast.Show()` with error text and TTL > 0
- [x] 1.3 Define `actionResultMsg` struct in `internal/tui/model.go` with field `err error`
- [x] 1.4 Add `case actionResultMsg:` handler in `Model.Update()` that calls `m.toast.Show(msg.Error(), 3)` — use success text when `err == nil`, error text otherwise
- [x] 1.5 Run `go test -race ./...`, `go vet ./...`, and `golangci-lint run` — verify all pass

## Phase 2: Search → Dashboard Wiring (Gap 2)

- [x] 2.1 Write test in `internal/tui/screens/dashboard_test.go`: `SetFilter("conf")` on a populated dashboard returns only rows containing "conf" (case-insensitive)
- [x] 2.2 Write test: `SetFilter("")` restores all original rows
- [x] 2.3 Write test: `SetFilter("xyz")` with no matches shows empty table rows
- [x] 2.4 Add `allRows []table.Row` field to `DashboardModel` in `internal/tui/screens/dashboard.go`; populate it in `NewDashboardModel`
- [x] 2.5 Implement `SetFilter(query string)` method on `DashboardModel`: case-insensitive substring filter on `allRows`, rebuild `table.Rows()`, reset cursor to 0
- [x] 2.6 Write test in `internal/tui/model_test.go`: when search is active and query changes, `dashboard.SetFilter()` is called with the current query
- [x] 2.7 Wire search forwarding in `Model.Update()` for `ScreenDashboard`: after forwarding to dashboard, call `m.dashboard.SetFilter(m.search.Query())`
- [x] 2.8 Run `go test -race ./...`, `go vet ./...`, and `golangci-lint run` — verify all pass

## Phase 3: Menu Items 1 & 4 (Gap 3)

- [x] 3.1 Write test in `internal/tui/model_test.go`: cursor=1 + enter → toast shows "coming soon" message
- [x] 3.2 Write test: cursor=4 + enter → toast shows "coming soon" message
- [x] 3.3 Add `case 1:` and `case 4:` in `handleMenuEnter()` in `internal/tui/model.go` — call `m.toast.Show("Restore: coming soon", 3)` / `m.toast.Show("Profiles: coming soon", 3)` and return `m, nil`
- [x] 3.4 Run `go test -race ./...`, `go vet ./...`, and `golangci-lint run` — verify all pass

## Phase 4: ScreenWizard Removal (Gap 4)

- [x] 4.1 Write test in `internal/tui/model_test.go`: compile-time check that no `ScreenWizard` constant exists (grep-based or verify screen iota values match expected sequence)
- [x] 4.2 Remove `ScreenWizard` from the `Screen` iota enum in `internal/tui/model.go`; verify no switch cases or references to it exist in the codebase
- [x] 4.3 Run `go test -race ./...`, `go vet ./...`, and `golangci-lint run` — verify all pass

## Phase 5: Action Dispatch (Gap 5)

- [x] 5.1 Write test in `internal/tui/dispatch_test.go`: table-driven test for `RouteSelection(sel MenuSelection, deps Deps) error`:
  - cursor 0 (Backup) → calls `deps.RunBackup`, returns nil
  - cursor 1 (Restore) → calls `deps.RunRestore` (or no-op if not wired), returns nil
  - cursor 6 (Quit) → no action, returns nil
  - empty selection → no action, returns nil
- [x] 5.2 Implement `RouteSelection()` as a pure function in `internal/tui/dispatch.go`:
  - Takes `MenuSelection` and `Deps`
  - Switches on `sel.Cursor`
  - Calls appropriate action function from `Deps`
  - Returns error if action fails
- [x] 5.3 Update `cmd/tty.go` `defaultRunTUI`: after `p.Run()` returns, call `tui.RouteSelection(model.Selection(), deps)` instead of inline switch
- [x] 5.4 Run `go test -race ./...`, `go vet ./...`, and `golangci-lint run` — verify all pass
