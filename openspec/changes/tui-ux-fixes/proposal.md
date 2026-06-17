# Proposal: TUI UX Fixes — Navigation, Footer, Dashboard, Terminal Thresholds

## Intent

Five UX defects from real-user testing: arrow keys don't navigate, cursor doesn't wrap, help footer vanishes on sub-screens, dashboard always empty (`ListBackups` never wired), "terminal too small" triggers on reasonable windows.

## Scope

### In Scope
- Map arrow keys (up/down) alongside j/k on all navigable screens
- Add wrap-around navigation (first→last, last→first) on menu and settings
- Add persistent help bar footer to settings, dashboard, health, and cloud screens
- Wire `ListBackups` dependency in `cmd/root.go` so dashboard populates
- Lower terminal minimum dimensions from 20×10 to 40×12

### Out of Scope
- New screens, visual redesign, `internal/actions/` changes, bubbles/table wrap-around

## Capabilities

### New Capabilities
None

### Modified Capabilities
- `tui-main-menu`: Arrow key constants, wrap-around in `handleKey` for ScreenMenu
- `tui-dashboard`: Help bar footer; wire `ListBackups` in `cmd/root.go`
- `tui-components`: Help bar in settings, health, and cloud renderers
- `bak-cli`: Lower `minWidth`/`minHeight` constants and per-screen guards

## Approach

1. **Dashboard wiring** — one-line fix in `cmd/root.go`
2. **Arrow keys + wrap-around** — modular arithmetic in menu/settings
3. **Help bar persistence** — each sub-screen appends `components.RenderHelp(keys)`
4. **Terminal minimums** — lower to 40×12; centralize in `styles` package
5. Reference wrap-around pattern from gentle-ai `model.go`

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/tui/keys.go` | Modified | Add arrow key constants |
| `internal/tui/model.go` | Modified | Wrap-around + arrow keys in ScreenMenu; lower min dimensions |
| `internal/tui/screens/settings.go` | Modified | Arrow keys + wrap-around; help bar |
| `internal/tui/screens/dashboard.go` | Modified | Help bar; lower terminal guard |
| `internal/tui/screens/health.go` | Modified | Help bar via RenderHelp |
| `internal/tui/screens/cloud.go` | Modified | Help bar |
| `cmd/root.go` | Modified | Wire `ListBackups` into `tui.Deps` |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Arrow key constants conflict with bindings | Low | Only j/k/q/enter/esc mapped; arrows are free |
| Wrap-around on dashboard breaks table cursor | Low | Only apply to menu/settings; table self-manages |
| Lower minimums cause layout overflow | Low | Test at 40×12; logo guarded at width<40 |

## Rollback Plan

Each fix is an independent commit, independently revertable.

## Dependencies

None — all fixes use existing components (`components.RenderHelp`, `tui.Deps.ListBackups`)

## Success Criteria

- [ ] Arrow up/down navigate menu and settings
- [ ] Cursor wraps: last→first on down, first→last on up
- [ ] Help bar visible on settings, dashboard, health, cloud
- [ ] Dashboard shows backup list when backups exist
- [ ] "Terminal too small" only triggers below 40×12
- [ ] All existing tests pass; new navigation tests ≥80% coverage
