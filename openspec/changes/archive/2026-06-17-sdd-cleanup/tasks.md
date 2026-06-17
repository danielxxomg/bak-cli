# Tasks: SDD Cleanup — Close All Open Change Lifecycles

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~200–280 (new markdown content + git mv renames) |
| 800-line budget risk | Low |
| Chained PRs recommended | No |
| Suggested split | Single PR |
| Delivery strategy | ask-always |
| Chain strategy | stacked-to-main |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: stacked-to-main
800-line budget risk: Low

## Workstream A: Archive 7 Ready Changes

### A.1 Pre-flight verification
- [x] Verify all 7 changes have `verify-report.md` AND `archive-report.md`: `ci-fix`, `cloud-consolidation`, `generic-adapter`, `path-normalization`, `test-hardening`, `tui-ux-fixes`, `tui-wiring-gaps`

### A.2 Archive each change
- [x] `git mv openspec/changes/ci-fix/ openspec/changes/archive/2026-06-17-ci-fix/`
- [x] `git mv openspec/changes/cloud-consolidation/ openspec/changes/archive/2026-06-17-cloud-consolidation/`
- [x] `git mv openspec/changes/generic-adapter/ openspec/changes/archive/2026-06-17-generic-adapter/`
- [x] `git mv openspec/changes/path-normalization/ openspec/changes/archive/2026-06-17-path-normalization/`
- [x] `git mv openspec/changes/test-hardening/ openspec/changes/archive/2026-06-17-test-hardening/`
- [x] `git mv openspec/changes/tui-ux-fixes/ openspec/changes/archive/2026-06-17-tui-ux-fixes/`
- [x] `git mv openspec/changes/tui-wiring-gaps/ openspec/changes/archive/2026-06-17-tui-wiring-gaps/`

## Workstream B: Finalize actions-di

### B.1 Write archive-report.md
- [x] Create `openspec/changes/actions-di/archive-report.md` — derive from `verify-report.md` (1161/1161 tests, 15/15 tasks, PASS verdict) and `tasks.md` (4 phases, all `[x]`). Include: summary, completeness table, build/test evidence, verdict.

### B.2 Archive
- [x] `git mv openspec/changes/actions-di/ openspec/changes/archive/2026-06-17-actions-di/`

## Workstream C: Close tui-overhaul

### C.1 Mark Phase 11 items done with cross-refs
- [x] Edit `openspec/changes/tui-overhaul/tasks.md` — change items 11.1–11.5 from `[ ]` to `[x]` and append cross-ref notes:
  - 11.1 (Wizard) → resolved by `tui-wiring-gaps/verify-report.md` (ScreenWizard constant removed)
  - 11.2 (Restore/Profiles) → resolved by `tui-wiring-gaps/verify-report.md` (menu cursor handlers)
  - 11.3 (Post-TUI dispatch) → resolved by `tui-wiring-gaps/verify-report.md` (RouteSelection)
  - 11.4 (Dashboard search) → resolved by `tui-ux-fixes/verify-report.md` (search filter integration)
  - 11.5 (Toast triggering) → resolved by `tui-ux-fixes/verify-report.md` (toast on async ops)

### C.2 Archive
- [x] `git mv openspec/changes/tui-overhaul/ openspec/changes/archive/2026-06-17-tui-overhaul/`

## Workstream D: Prune Stale Folders

### D.1 Delete empty folder
- [x] `git rm -r openspec/changes/core-commands-fix/` (empty — no tracked files; rmdir sufficient)

### D.2 Backfill adapter-knowledge
- [x] Create `openspec/changes/adapter-knowledge/proposal.md` — summarize from `tasks.md`: validate adapter configRelPath and categoryMap against design docs, 7 adapters
- [x] Create `openspec/changes/adapter-knowledge/spec.md` — extract 4 requirements from tasks: export identifiers, knowledge test, fix adapters, registry-driven discovery
- [x] Create `openspec/changes/adapter-knowledge/design.md` — table of expectedKnowledge per adapter (from tasks Phase 2 fixes)
- [x] Create `openspec/changes/adapter-knowledge/archive-report.md` — 13/13 tasks complete, all tests pass, derived from `tasks.md`

### D.3 Archive adapter-knowledge and coverage-explore
- [x] `git mv openspec/changes/adapter-knowledge/ openspec/changes/archive/2026-06-17-adapter-knowledge/`
- [x] `git mv openspec/changes/coverage-explore/ openspec/changes/archive/2026-06-17-coverage-explore/`

## Final: Commit

- [x] Stage and commit all workstreams: `chore(sdd): archive 11 changes, close sdd-cleanup cycle`

## Success Verification

- [x] `ls openspec/changes/` shows only `archive/` and `sdd-cleanup/`
- [x] 11 new `2026-06-17-*` folders in `archive/`
- [x] `tui-overhaul/tasks.md` Phase 11 items 11.1–11.5 all `[x]` (before move)
- [x] `git log --follow` preserves history for moved folders
