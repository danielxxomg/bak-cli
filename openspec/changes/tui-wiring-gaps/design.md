# Design: TUI Wiring Gaps

## Technical Approach

All wiring happens in existing files — no new packages. Each gap is a small, focused change that connects two already-working pieces. The implementation order follows the dependency chain: toast first (others use it), then search, then menu items, then wizard resolution, then action dispatch.

## Architecture Decisions

| Decision | Choice | Rejected | Rationale |
|----------|--------|----------|-----------|
| Toast trigger mechanism | Custom `actionResultMsg` tea.Msg | Polling in Update loop | Clean separation: actions return results as messages, model reacts |
| Search→Dashboard bridge | `DashboardModel.SetFilter(query)` rebuilds rows | Pass Search into DashboardModel | Dashboard stays self-contained; root model owns the search component |
| Menu items 1 & 4 | "Coming soon" toast via `Toast.Show()` | New screen stubs | No new screens in scope; toast gives immediate user feedback |
| ScreenWizard resolution | Remove constant + renumber iota | Implement minimal wizard | Wizard is out of scope for this change; dead code removal is cleaner |
| Action dispatch location | After `p.Run()` returns in `defaultRunTUI` | Inside tea.Cmd | Actions are synchronous; running them after TUI exit avoids blocking the event loop |
| Filter storage in dashboard | Store `allRows` alongside filtered `table.Rows()` | Re-query `ListBackups` on each keystroke | Avoids repeated I/O; filter is a pure in-memory operation |

## Data Flow

```
Toast wiring:
  action completes → actionResultMsg{err} → model.Update()
    → m.toast.Show(msg, 3) → toast renders for 3s → auto-hides

Search wiring:
  user types '/' → m.search.Activate()
  user types chars → m.search.Update(msg) → m.search.Query() changes
    → m.dashboard.SetFilter(m.search.Query()) → table rows rebuild

Menu items 1 & 4:
  cursor=1, enter → handleMenuEnter() → m.toast.Show("Restore: coming soon", 3)
  cursor=4, enter → handleMenuEnter() → m.toast.Show("Profiles: coming soon", 3)

Wizard resolution:
  Remove ScreenWizard from iota enum → renumber ScreenSettings..ScreenHealth
  → update any switch cases that reference removed constant

Action dispatch:
  p.Run() returns → model := finalModel.(tui.Model)
    → sel := model.Selection()
    → switch sel.Cursor { case 0: actions.RunBackup(); case 1: actions.RunRestore() }
```

## File Changes

| File | Change | Gap |
|------|--------|-----|
| `internal/tui/model.go` | Add `actionResultMsg` type; handle it in Update to call `toast.Show()` | 1 (toast) |
| `internal/tui/model.go` | Add cases 1 and 4 in `handleMenuEnter()` with toast feedback | 2 (menu) |
| `internal/tui/model.go` | Forward `search.Query()` to `dashboard.SetFilter()` on each dashboard key event | 3 (search) |
| `internal/tui/model.go` | Remove `ScreenWizard` constant; renumber iota | 4 (wizard) |
| `internal/tui/screens/dashboard.go` | Add `allRows` field; add `SetFilter(query string)` method; filter is case-insensitive substring | 3 (search) |
| `cmd/tty.go` | Capture model after `p.Run()`; switch on `Selection().Cursor` to call actions | 5 (dispatch) |

## Testing Strategy

| Gap | Test approach | Key assertions |
|-----|---------------|----------------|
| Toast | Direct `Model.Update()` call with `tea.Msg` (from go-testing decision gate: "TUI state transition") | `m.toast` is visible with correct message |
| Toast | Direct `Model.Update()` call with `tea.Msg` | Toast shows error text |
| Search (forwarding) | Direct `Model.Update()` call with `tea.Msg` | Dashboard filter called with search query |
| Search (filter) | Table-driven unit test for `SetFilter()` | `table.Rows()` contains only matching rows, empty query restores all rows |
| Menu 1 | Direct `Model.Update()` call with `tea.Msg` | Toast shows "coming soon" |
| Menu 4 | Direct `Model.Update()` call with `tea.Msg` | Toast shows "coming soon" |
| Wizard | Compile-time check (grep for removed constant) | No references to `ScreenWizard` |
| Dispatch | Table-driven test for `routeSelection()` pure function | Routes cursor 0 → `RunBackup`, cursor 1 → no-op, cursor 6 → nil, empty → nil |

All tests use table-driven format per AGENTS.md. Mock `Deps` via function fields. No real TTY required.

For action dispatch, extract routing into a pure function `routeSelection(sel MenuSelection, deps Deps) error` and test it with table-driven cases. `defaultRunTUI` just calls `routeSelection(model.Selection(), deps)` — no unit test needed for that wrapper.

## Quality Gates

Each phase MUST pass these before moving to the next:
- `go test -race ./...` — all tests pass with race detector
- `go vet ./...` — no static analysis warnings
- `golangci-lint run` — no linting issues (if installed)

## Risk Assessment

**Overall: Low.** All components exist and are tested. These are connection changes, not new functionality.

- Search filter on large datasets: dashboard shows local backups (typically <100), so substring filter is O(n) with negligible cost
- Action dispatch blocking: actions run after TUI exit, so no event loop blocking
- ScreenWizard removal: if any test references it, the test will fail at compile time — easy to catch

## Rollback Plan

Each gap is one commit. Revert order (reverse of implementation):
1. Action dispatch — restore original `defaultRunTUI`
2. ScreenWizard — restore constant (or keep removed if no references)
3. Menu items 1 & 4 — remove cases from `handleMenuEnter()`
4. Search→Dashboard — remove `SetFilter()` call from model, remove method from dashboard
5. Toast — remove `actionResultMsg` handling
