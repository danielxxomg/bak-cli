# Proposal: CI Hardening v2

## Intent

Deterministic CI gates are missing, unpinned, or swallowed (`|| true`). Adds teeth to coverage, lint, vulncheck, and pre-commit while keeping GGA advisory (REQ-CI-004).

## Scope

| # | Item | Type | Lines |
|---|------|------|-------|
| A | Enable `maintidx` (<20) + `dupl` (threshold 80) | Config | ~10 |
| B | `.githooks/pre-commit` + `task setup` | Script | ~40 |
| C | `golangci-lint-action@v8` pinned v2.12.2 | Config | ~10 |
| D | Per-package 80% gate `internal/` + tui/screens bump | Script | ~30 |
| E | `govulncheck` blocks HIGH/CRITICAL; pin versions | Config | ~15 |
| F | `go install gga@v2.8.1` replaces brew | Config | ~5 |
| G | Fix 3 `dupl` pairs (cloud, cmd, tui) | Refactor | ~120 |
| H | Document deferred linters | — | 0 |

**Out**: complexity violations (→ `qa-refactor-analysis`), flaky tests, GGA gate change, architectural changes.

## Capabilities

- **New**: `refactoring-linters` — maintidx + dupl; test exclusions
- **Modified**: `ci-consistency` — REQ-CI-005/006/007 (coverage gate, pinned action, govulncheck)

## Approach

- **A**: Add to `.golangci.yml` enable + `_test.go` exclusion
- **B**: Hook: `golangci-lint && go vet && go build && gga run`; `task setup` → `core.hooksPath`
- **C**: Replace `go install @latest` with action@v8 `version: v2.12.2`
- **D**: `task cover:pkg` parses `go test -cover`, fails if `internal/` < 80%; bump tui/screens ~2%
- **E**: Remove `|| true` from govulncheck; parse HIGH/CRITICAL; pin versions
- **F**: `go install .../cmd/gga@v2.8.1`; verify path in apply
- **G**: `pullContentFromAPI` in `cloud/httputil.go`; `loadExcludes` in `cmd/`; `mapBackupInfo` in `tui/model.go`

## Affected Areas

`.golangci.yml` · `.githooks/pre-commit` (new) · `Taskfile.yml` · `ci.yml` · `gga.yml` · `cloud/{httputil,gitea,github_repo}.go` · `cmd/{root,backup}.go` · `tui/model.go` · `CONTRIBUTING.md`

## Risks

| Risk | Sev | Mitigation |
|------|-----|------------|
| GGA `go install` path unverified | Med | Apply prerequisite; binary fallback |
| Cloud consolidation conflicts with archive | Med | Cross-check 2026-06-17 |
| tui/screens at 80.0% flakes | Med | Bump in same change |
| gocognit deferral | Low | maintidx signal; tracked |

## Deferred

`gocognit` (24 violations, 3 SEVERE), `funlen` (5), `nestif` (15) → `qa-refactor-analysis`.

## Rollback

`git revert`. Additive config + isolated refactors.

## Dependencies

- GGA import path verified before gga.yml change
- tui/screens bump lands with per-pkg gate

## Success Criteria

- [ ] `golangci-lint run` passes maintidx + dupl (0 violations)
- [ ] `.githooks/pre-commit` runs full local gate
- [ ] CI uses `golangci-lint-action@v8` v2.12.2
- [ ] `task cover:pkg` fails if `internal/` < 80%
- [ ] `govulncheck` blocks HIGH/CRITICAL
- [ ] `gga.yml` installs without linuxbrew
- [ ] `dupl` 0 violations
- [ ] Total lines < 800
