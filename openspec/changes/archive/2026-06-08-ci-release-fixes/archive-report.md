# Archive Report: ci-release-fixes

**Archived**: 2026-06-08
**Change**: ci-release-fixes
**Project**: bak-cli
**Artifact Store**: openspec

## Summary

CI pipeline fixes aligning Go version across all jobs, fixing cross-platform binary verification in CI, and making the Taskfile binary name OS-conditional.

## Tasks Status

| Phase | Task | Status |
|-------|------|--------|
| Phase 1: Fix CI Go Version | 1.1 Update security job Go version to '1.25' | ✅ |
| Phase 1: Fix CI Go Version | 1.2 Update goreleaser job Go version to '1.25' | ✅ |
| Phase 1: Fix CI Go Version | 1.3 Update build job Go version to '1.25' | ✅ |
| Phase 2: Fix Build Verification | 2.1 Fix Unix verify step (`./bak.exe` → `./bak`) | ✅ |
| Phase 2: Fix Build Verification | 2.2 Confirm Windows step correct | ✅ |
| Phase 3: Fix Taskfile Binary | 3.1 OS-conditional `BINARY` var in Taskfile.yml | ✅ |
| Phase 4: Verification | 4.1–4.3 Grep checks and local build test | ✅ |

**All tasks complete**: 7/7

## Specs Synced

| Domain | Action | Details |
|--------|--------|---------|
| ci-consistency | Created | 3 requirements added (Go version consistency, cross-platform binary verification, Taskfile binary name) |

## Archive Contents

- `proposal.md` ✅ — Intent, scope, approach, risks, rollback plan
- `specs/ci-consistency/spec.md` ✅ — Delta spec with 3 ADDED requirements and scenarios
- `design.md` ✅ — Architecture decisions with alternatives considered
- `tasks.md` ✅ — 7/7 tasks complete, all checked
- `archive-report.md` ✅ — This file

## Missing Artifacts

- `verify-report.md` — Not present in the original change folder. This change was archived without a formal verification report. The orchestrator initiated the archive, confirming the change is complete.

## Source of Truth Updated

- `openspec/specs/ci-consistency/spec.md` — Created (new domain)

## Risks

None identified. All changes were limited to YAML configuration files (`.github/workflows/ci.yml` and `Taskfile.yml`). No code logic was modified.

## Notes

- This change had no verify-report.md artifact. Archive proceeded per orchestrator instructions.
- No destructive merge was needed — `ci-consistency` is a new domain spec.
- Config `rules.archive` was respected (no destructive deltas to warn about).
