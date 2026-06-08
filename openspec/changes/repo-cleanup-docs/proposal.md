# Proposal: repo-cleanup-docs

## Intent

Fix stale, incorrect, and contradictory documentation across SECURITY.md, CONTRIBUTING.md, README.md, and CHANGELOG.md. Clean up openspec housekeeping debt (stale changes, misplaced verify reports, wrong Go version). Remove/rename misnamed files. Zero code logic changes — this is purely docs, config, and file hygiene.

## Scope

### In Scope
- Fix 8 documentation errors in SECURITY.md, CONTRIBUTING.md, README.md
- Restructure CHANGELOG.md: move v1.0.0–v1.3.0 from [Unreleased] into versioned sections
- Fix openspec/config.yaml Go version (1.26 → 1.25)
- Archive 4 stale openspec changes
- Move 2 floating verify reports into archive
- Delete empty `scripts/` directory
- Rename `examples/presets/custom.yaml` content to match filename
- Fix `.gga` to include `*_test.go` in review
- Rename `cmd/coverage_test.go` → `cmd/wiring_test.go`

### Out of Scope
- Any code logic or behavior changes
- Adding new tests or coverage
- CI/CD pipeline changes
- New features or capabilities

## Capabilities

### New Capabilities
None — this is a cleanup change with no new behavior.

### Modified Capabilities
None — no spec-level requirements are changing.

## Approach

Grouped into 4 independent batches (each committable separately):

**Batch 1 — Docs accuracy (8 fixes)**
- SECURITY.md: replace `filepath.ToSlash` recommendation with `strings.ReplaceAll(path, "\\", "/")`; fix "No encryption at rest" → document encryption added in v0.3.0
- CONTRIBUTING.md: same forbidden function fix; "Go 1.24+" → "Go 1.25+"; `make` → `task` (15+ refs); remove `scripts/` directory reference
- README.md: fix architecture diagram (Makefile → Taskfile.yml); update roadmap (mark v0.2.0/v0.3.0 as released)

**Batch 2 — CHANGELOG restructure**
- Move v1.0.0, v1.1.0, v1.2.0, v1.3.0 entries from `[Unreleased]` into dated version sections
- Leave genuinely unreleased items under `[Unreleased]`

**Batch 3 — openspec housekeeping**
- Fix config.yaml Go version
- Archive 4 stale changes → `openspec/changes/archive/2026-06-08-repo-cleanup/`
- Move 2 verify reports → same archive location

**Batch 4 — File cleanup**
- Delete empty `scripts/` directory
- Fix `examples/presets/custom.yaml` name field to `custom`
- Fix `.gga` L11: remove `*_test.go` exclusion pattern
- `git mv cmd/coverage_test.go cmd/wiring_test.go`

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `SECURITY.md` | Modified | Fix forbidden function ref, encryption status |
| `CONTRIBUTING.md` | Modified | Fix Go version, make→task, forbidden function, stale paths |
| `README.md` | Modified | Fix architecture diagram, roadmap versions |
| `CHANGELOG.md` | Modified | Restructure into versioned sections |
| `openspec/config.yaml` | Modified | Fix Go version |
| `openspec/changes/` | Modified | Archive 4 stale changes, move 2 verify reports |
| `scripts/` | Removed | Empty directory |
| `examples/presets/custom.yaml` | Modified | Fix name field mismatch |
| `.gga` | Modified | Include test files in review |
| `cmd/coverage_test.go` | Renamed | → `cmd/wiring_test.go` |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| CHANGELOG restructure breaks existing links | Low | Grep for anchor links before restructuring |
| Archiving stale changes loses context | Low | Archive preserves all files; nothing is deleted |
| `.gga` change causes CI noise from test files | Low | Intentional — tests should be reviewed |

## Rollback Plan

All changes are file edits, renames, and moves — no code logic. Single `git revert` of the merge commit restores everything. For individual batches, each is a separate commit and can be reverted independently.

## Dependencies

None.

## Success Criteria

- [ ] No references to `filepath.ToSlash` in any .md file
- [ ] No references to `make` commands in CONTRIBUTING.md
- [ ] All Go version references say 1.25
- [ ] CHANGELOG.md has versioned sections for v1.0.0–v1.3.0
- [ ] Zero stale changes in openspec/changes/ (only archive/ and repo-cleanup-docs/)
- [ ] Zero floating verify reports at openspec root
- [ ] `scripts/` directory does not exist
- [ ] `cmd/wiring_test.go` exists, `cmd/coverage_test.go` does not
- [ ] `.gga` includes `*_test.go` in review scope
