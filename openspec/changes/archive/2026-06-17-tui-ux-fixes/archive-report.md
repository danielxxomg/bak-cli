# Archive Report — TUI UX Fixes

## Change Summary

**Change ID**: tui-ux-fixes
**Project**: bak-cli
**Date**: 2026-06-17
**Verdict**: PASS WITH WARNINGS
**Tasks**: 22/22 complete

Five TUI UX defects fixed from real-user testing:

1. **Dashboard wiring** — `ListBackups` was never injected into `tui.Deps`, so the dashboard always showed empty. Wired `listBackups()` via `backup.BakDir()` + `manifest.Load()` in `cmd/root.go`.
2. **Arrow key navigation** — Only j/k worked. Added `tea.KeyDown`/`tea.KeyUp` handling in main menu and settings via `msg.Code`.
3. **Wrap-around navigation** — Cursor clamped at boundaries. Replaced with modular arithmetic (`(cursor+1)%len` / `(cursor-1+len)%len`) on menu and settings.
4. **Persistent help bar** — Only main menu had a help footer. Added `components.RenderHelp` to settings, dashboard (all 3 return paths), health (idle + completed), and cloud screens.
5. **Terminal minimums** — 20x10 was too aggressive. Lowered to 40x12, centralized as `styles.MinWidth`/`styles.MinHeight` (single source of truth).

## Files Created/Modified

| File | Action | Description |
|------|--------|-------------|
| `cmd/root.go` | Modified | Added `listBackups()` + `listBackupsFrom()`; wired `ListBackups` into `tui.Deps` |
| `cmd/root_test.go` | Modified | 4 tests for manifest scanning, empty dirs, corrupt manifests |
| `internal/tui/keys.go` | Modified | Added `KeyDownArrow`/`KeyUpArrow` string constants |
| `internal/tui/model.go` | Modified | Wrap-around + arrow keys in ScreenMenu; `styles.MinWidth`/`MinHeight`; dimensional too-small message |
| `internal/tui/model_test.go` | Modified | 8 arrow/wrap cases + 2 updated; MinSizeGuard expanded 5->8 cases |
| `internal/tui/styles/styles.go` | Modified | Added exported `MinWidth=40`, `MinHeight=12` constants |
| `internal/tui/screens/settings.go` | Modified | Arrow keys + wrap-around; help bar; `styles.MinWidth`/`MinHeight` guard |
| `internal/tui/screens/settings_test.go` | Modified | 8 arrow+wrap tests; 2 help bar tests; 5 MinSizeGuard cases |
| `internal/tui/screens/dashboard.go` | Modified | `renderDashboardHelp()` on all 3 return paths; `styles.MinWidth`/`MinHeight` guard |
| `internal/tui/screens/dashboard_test.go` | Modified | 3 help bar tests; 5 MinSizeGuard cases |
| `internal/tui/screens/health.go` | Modified | Replaced inline help with `components.RenderHelp`; `styles.MinWidth`/`MinHeight` guard |
| `internal/tui/screens/health_test.go` | Modified | Extended rerun footer test; idle help bar test; 5 MinSizeGuard cases |
| `internal/tui/screens/cloud.go` | Modified | Restructured to `strings.Builder`; `renderCloudHelp()` helper |
| `internal/tui/screens/cloud_test.go` | Modified | 2 help bar tests (provider + no-provider) |

**Total**: 14 files changed, ~388 lines

## Quality Gates

| Check | Result |
|-------|--------|
| `go test -race ./...` | 28/28 packages pass |
| `go vet ./...` | Zero warnings |
| `golangci-lint run` | 0 issues |
| Coverage `internal/tui/` | 91.7% |
| Coverage `internal/tui/components/` | 97.9% |
| Coverage `internal/tui/screens/` | 95.0% |
| Coverage `internal/tui/styles/` | 90.5% |

## Warnings

### 1. Dashboard arrow-key forwarding lacks explicit test coverage

- **Spec scenario**: "Arrow keys on dashboard" requires down arrow forwarded to table sub-model.
- **Implementation**: Correctly forwards via default path in `model.go` and fall-through in `dashboard.go`.
- **Gap**: No test sends `tea.KeyDown`/`tea.KeyUp` to `DashboardModel.Update` and asserts table cursor movement. Existing `TestDashboard_Update_NavigateDown/Up` only cover j/k.
- **Recommendation**: Add a narrow test sending arrow keys to dashboard and asserting cursor change.
- **Severity**: Low — implementation is correct, test gap only.

## Design Deviations (Accepted)

| Design Intent | Actual Implementation | Rationale |
|---------------|----------------------|-----------|
| `msg.String()` matching `"down"`/`"up"` | `msg.Code` with `tea.KeyDown`/`tea.KeyUp` | Type-safe, matches existing patterns, avoids rewriting all cases |
| `config.BackupDir()` + `backup.ListManifests()` | `backup.BakDir()` + `manifest.Load()` | Design referenced non-existent functions; used real helpers |
| Cloud `content +=` concatenation | `strings.Builder` + `renderCloudHelp()` helper | Consistency with other screens |

## Lessons Learned

1. **bubbletea v2 arrow keys**: `tea.KeyDown`/`tea.KeyUp` as `msg.Code` runes is the recommended pattern over `msg.String()` matching. The design initially specified string matching, but the rune approach is more type-safe and avoids edge cases (space->"space", enter->"enter").
2. **Circular import avoidance**: `screens` package cannot import `tui` package. Centralizing `MinWidth`/`MinHeight` in `styles` (imported by both) is the correct pattern for shared TUI constants.
3. **Nil dependency guards**: The existing `ListBackups == nil` guard in `initDashboard` was designed for test isolation. Keeping it while wiring the real function in `cmd/root.go` gives both production behavior and test flexibility.
4. **Help bar on all return paths**: Dashboard has 3 return paths (error, empty, populated). Each must append the help bar independently — easy to miss one.

## Specs Synced

No delta spec sync performed. The change spec (`spec.md`) is a standalone document at the change root — no `specs/{domain}/` subdirectory exists, and no TUI-specific main specs exist in `openspec/specs/`. The spec remains in the change directory as the audit trail.

## Archive Contents

- proposal.md ✅
- spec.md ✅
- design.md ✅
- tasks.md ✅ (22/22 tasks complete)
- apply-progress.md ✅
- verify-report.md ✅
- archive-report.md ✅ (this file)

## SDD Cycle Status

**COMPLETE** — The change has been fully planned, implemented, verified, and archived.
