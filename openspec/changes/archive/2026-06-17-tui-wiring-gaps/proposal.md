# Proposal: TUI Wiring Gaps — Connect Existing Components

## Intent

The TUI overhaul implemented all UI components (toast, search, dashboard, menu, progress, settings, health, cloud, shortcuts) but left 5 integration gaps. Components exist in isolation — toast never fires, search doesn't filter, two menu items are dead, the wizard screen constant has no implementation, and the TUI selection doesn't route to cobra actions. This change wires them together so the TUI is actually functional end-to-end.

## Scope

### In Scope
- Wire `Toast.Show()` to backup/restore completion and error events
- Connect `Search.Filter()` to `DashboardModel` table row filtering
- Add case handlers for menu cursor 1 (Restore) and 4 (Profiles) — route to screen or show "coming soon" toast
- Resolve dead `ScreenWizard` constant — implement minimal wizard screen or remove it
- Bridge `Model.Selection()` output in `defaultRunTUI` to cobra action dispatch (backup/restore)

### Out of Scope
- New UI components or screens beyond the wizard resolution
- Visual redesign or theme changes
- Changes to `internal/actions/` business logic
- Cloud provider implementation
- bubbles/list or other new dependencies

## Capabilities

### New Capabilities
- `tui-action-dispatch`: Bridge TUI selection to cobra action execution after `tea.Quit`

### Modified Capabilities
- `tui-main-menu`: Add case handlers for cursor 1 (Restore) and 4 (Profiles) in `handleMenuEnter`; resolve dead `ScreenWizard` constant
- `tui-dashboard`: Wire search component to filter dashboard table rows
- `tui-components`: Wire toast `Show()` calls to success/error events from action completions

## Approach

Bottom-up wiring in dependency order:
1. **Toast** (simplest) — add `Show()` calls in model.go after action completion messages
2. **Search→Dashboard** — forward `search.Query()` to `DashboardModel.SetFilter()` on each keystroke
3. **Menu items 1 & 4** — add cases in `handleMenuEnter()` with toast feedback or screen transitions
4. **ScreenWizard** — remove dead constant if no wizard screen is planned; implement minimal version if first-run flow is needed
5. **Action dispatch** (most complex) — capture `Selection()` after `p.Run()` returns in `defaultRunTUI`, route to `actions.RunBackup`/`actions.RunRestore`

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/tui/model.go` | Modified | Add toast wiring, menu cases 1/4, search→dashboard forwarding, wizard resolution |
| `internal/tui/screens/dashboard.go` | Modified | Add `SetFilter(query string)` method to rebuild table rows from filtered data |
| `cmd/tty.go` | Modified | Capture `Selection()` after `p.Run()`, route to action functions |
| `internal/tui/screens/wizard.go` | New or removed | Either implement minimal wizard or remove `ScreenWizard` constant |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Search filtering breaks table cursor position | Low | Reset cursor to 0 on filter change |
| Action dispatch blocks TUI exit | Low | Run actions after `p.Run()` returns, not inside tea.Cmd |
| Wizard removal breaks screen enum ordering | Med | Use `iota` with explicit values or leave constant as unused |

## Rollback Plan

Each wiring change is an independent commit. Toast, search, and menu items can be reverted individually without affecting other gaps. Action dispatch is the most coupled — revert by restoring the original `defaultRunTUI` that ignores `Selection()`.

## Dependencies

- tui-overhaul change (all components must exist — they do)

## Success Criteria

- [ ] Toast displays "Backup complete" after a successful backup action
- [ ] Typing in search (`/`) filters dashboard table rows in real-time
- [ ] Pressing enter on Restore (cursor 1) shows a "coming soon" toast or navigates to restore flow
- [ ] Pressing enter on Profiles (cursor 4) shows a "coming soon" toast
- [ ] `ScreenWizard` constant either has a working screen implementation or is removed
- [ ] `defaultRunTUI` routes TUI selection to `actions.RunBackup` / `actions.RunRestore`
- [ ] All existing TUI tests pass; new wiring tests at ≥80% coverage
