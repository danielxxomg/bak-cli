# Proposal: SDD Cleanup — Close All Open Change Lifecycles

## Intent

`openspec/changes/` holds 12 active folders; 7 already have verify + archive-reports, others are functionally complete. Unify all SDD debt in one pass: archive the ready ones, mark `tui-overhaul` Phase 11 done by reference, prune empty/exploration folders. **No code touched.**

## Scope

### In Scope

- **A. Archive 7 ready** (verify ✓ + archive-report ✓): `ci-fix`, `cloud-consolidation`, `generic-adapter`, `path-normalization`, `test-hardening`, `tui-ux-fixes`, `tui-wiring-gaps`
- **B. Finalize `actions-di`**: write missing `archive-report.md` (verify-report ✓, 1161/1161 tests pass) → archive
- **C. Close `tui-overhaul`**: mark Phase 11 items 11.1–11.5 `[x]` in `tasks.md` w/ cross-refs to `tui-wiring-gaps/verify-report.md` + `tui-ux-fixes/verify-report.md` (gaps already verified) → archive
- **D. Prune stale folders**: `core-commands-fix/` (empty) → delete; `adapter-knowledge/` (13/13 tasks, only `tasks.md`) → backfill proposal+spec+design+archive-report → archive; `coverage-explore/` → `git mv` to `archive/2026-06-17-coverage-explore/`

### Out of Scope

- Code changes; re-running tests/lint; modifying `openspec/specs/` (deltas already merged or N/A).

## Capabilities

### New Capabilities

None.

### Modified Capabilities

None.

## Approach

One `git mv` per change.

1. **A** → 7× `git mv openspec/changes/<name>/ openspec/changes/archive/2026-06-17-<name>/`
2. **B** → write `actions-di/archive-report.md` from verify-report + tasks → `git mv` to `archive/2026-06-17-actions-di/`
3. **C** → edit `tui-overhaul/tasks.md` Phase 11 (11.1–11.5 `[x]` w/ cross-refs) → `git mv` to `archive/2026-06-17-tui-overhaul/`
4. **D** → `rm -rf core-commands-fix/`; backfill `adapter-knowledge/{proposal,spec,design,archive-report}.md` from `tasks.md` → `git mv`; `git mv coverage-explore/` to archive

## Affected Areas

| Area | Impact |
|------|--------|
| `openspec/changes/` | 12 active → 0 |
| `openspec/changes/archive/2026-06-17-*/` | 11 new |
| `openspec/changes/tui-overhaul/tasks.md` | Phase 11 items 11.1–11.5 `[x]` |
| `openspec/changes/actions-di/archive-report.md` | New |
| `openspec/changes/adapter-knowledge/{proposal,spec,design,archive-report}.md` | New |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Archived change's delta not in `openspec/specs/` | Low | Pre-flight: confirm no spec change OR delta already merged before `git mv` |
| `adapter-knowledge` backfill thin | Medium | Use `tasks.md` as truth — minimal proposal, 4–5 requirements, design = expectedKnowledge table |
| `coverage-explore` name collides w/ future cycle | Low | Distinct `2026-06-17-coverage-explore` (vs `2026-06-16-coverage-improvement` already archived) |
| Phase 11 cross-refs rot | Low | Refs point to `verify-report.md` paths inside archive |
| `git mv` history loss | Low | All moves use `git mv` |

## Rollback Plan

1. `git restore openspec/changes/` reverts the tree.
2. `git revert <cleanup-sha>` reverses all moves.
3. Bad Phase 11 cross-ref → revert only `tui-overhaul/tasks.md` + its move.

## Dependencies

None.

## Success Criteria

- [ ] `ls openspec/changes/` shows 0 non-archive folders
- [ ] 11 new `2026-06-17-*` folders in `archive/`
- [ ] `tui-overhaul/tasks.md` Phase 11 items 11.1–11.5 all `[x]`
- [ ] `git log --follow` preserves history; `openspec/specs/` unchanged
- [ ] Conventional Commits commit (or one per workstream)
