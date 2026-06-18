# Proposal: TUI Coverage Backfill

## Intent

The TUI package is at 63.1% coverage against the 80% project threshold mandated by AGENTS.md for `internal/tui/`. The root cause is structural: 16 wizard tests live in `cmd/wizard_test.go` even though they exercise `internal/tui/screens/wizard.go`. Go's coverage tool attributes those tests to the `cmd` package — not the screens package — so `wizard.go` shows 0% across all 11 functions. This change moves the wizard tests to their proper package and backfills the remaining low-hanging 0% functions so the threshold is met without scope creep.

## Scope

### In Scope
- Move 16 tests from `cmd/wizard_test.go` → `internal/tui/screens/wizard_test.go` (white-box, `package screens`)
- Keep `TestIsTTY_NotTerminal` in `cmd/` (tests `cmd.isTTY()`)
- Backfill 0% functions in `internal/tui/model.go`: `initRestore`, `initProfiles`, `initCloud`
- Cover partial gaps in `model.go` `Update` (66.7%) and `handleKey` (67.1%) for unhit screen arms
- Backfill 0% render helpers in `internal/tui/screens/restore.go`: `renderErrorState`, `renderBackupList`, `renderRunning`, `renderDone`
- Backfill 0% `Init()` in `settings.go`, `health.go`
- Backfill 0% `renderError` in `profiles.go`
- Cover 50-70% gaps in `cloud.go`, `profiles.go`, `restore.go` `Update`/`View` via table-driven tests
- Cover `components/search.go` `IsActive` (0%) and edge cases in `toast.go`/`modal.go`
- Single PR, ~200 changed lines

### Out of Scope
- `internal/tui/components/` and `internal/tui/styles/` (already pass: 95.1% / 90.9%)
- `cmd/` coverage (covered by E2E per AGENTS.md)
- Golden-file rendering tests (over-scope — 80% threshold accommodates style branches)
- New product behavior, refactors of production code, dependency changes

## Capabilities

> Pure test-coverage refactor — no spec-level behavior changes.

### New Capabilities
- None

### Modified Capabilities
- None

## Approach

**Approach B from `explore.md`**: mechanical move first, then minimal backfill. The 16 wizard tests copy verbatim into `internal/tui/screens/wizard_test.go` with `package screens` (white-box, matches `cloud_test.go` / `dashboard_test.go` convention). `cmd/wizard_test.go` shrinks to a single TTY test. Then table-driven tests for the remaining 0% functions, using existing patterns: spy closures for DI builders (see `internal/actions/dispatch_test.go:50-65`) and substring assertions on `View().Content` for render helpers (see `internal/tui/screens/cloud_test.go:289`). No production code changes, no new dependencies.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `cmd/wizard_test.go` | Modified | Reduce to 1 test (`TestIsTTY_NotTerminal`) |
| `internal/tui/screens/wizard_test.go` | New | Host 16 moved tests in `package screens` |
| `internal/tui/model.go` | Modified (tests) | Cover `initRestore`/`initProfiles`/`initCloud` + `Update`/`handleKey` arms |
| `internal/tui/screens/{cloud,profiles,restore,settings,health}.go` | Modified (tests) | Cover 0% functions and 50-70% branches |
| `internal/tui/components/{search,toast,modal}.go` | Modified (tests) | Edge cases: `IsActive`, tick-expired, escape |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| `package screens` import cycle | Low | All other screens tests use this pattern; no import changes needed |
| `View()` substring assertions become brittle | Low | Same pattern as existing `cloud_test.go` / `dashboard_test.go` — proven stable |
| Some lipgloss style branches stay uncovered | Low | 80% threshold accommodates; non-blocking |
| Test count inflation masks real coverage gap | Low | Verify with `go test -coverprofile` per package, not aggregate |

## Rollback Plan

Single PR revert: `git revert <commit>` restores the original `cmd/wizard_test.go` (17 tests) and removes `internal/tui/screens/wizard_test.go`. No production code touched, no user-facing behavior change — pure test refactor. Backfilled tests live next to the code they cover and revert independently per file.

## SDD Phase Decision

| Phase | Executed | Rationale |
|-------|----------|-----------|
| sdd-explore | ✅ | Needed to measure coverage gaps and identify root cause |
| sdd-propose | ✅ | Defined scope, approach, success criteria |
| sdd-spec | ❌ Skip | Capabilities: None — no new/modified behavior, test-only change |
| sdd-design | ❌ Skip | No architecture decisions — mechanical test migration + pattern backfill |
| sdd-tasks | ✅ | Plan test migration and backfill tasks |
| sdd-apply | ✅ | Implement with strict TDD |
| sdd-verify | ✅ | Verify coverage ≥80% threshold |
| sdd-archive | ✅ | Close cycle |

Spec and design are required when the change introduces new capabilities or modifies system behavior. This change is purely structural (test file relocation + coverage backfill) with zero production code impact. Skipping spec/design avoids unnecessary artifacts while maintaining SDD discipline.

## Dependencies

- None — no new packages, no API changes, no toolchain upgrades

## Success Criteria

- [ ] `internal/tui/screens/wizard.go` coverage ≥80% (currently 0%)
- [ ] `internal/tui/` aggregate coverage ≥80% (currently 63.1%)
- [ ] `internal/tui/screens/` aggregate coverage ≥80% (currently 63.8%)
- [ ] `go test ./...` passes with zero regressions
- [ ] No new dependencies in `go.mod`
- [ ] PR diff stays within 400-line review budget
