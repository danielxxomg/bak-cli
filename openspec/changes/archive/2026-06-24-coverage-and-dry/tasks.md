# Tasks: coverage-and-dry

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~600 (4 commits: 150 + 200 + 150 + 100) |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | 4 stacked PRs (one per commit) |
| Delivery strategy | ask-almost |
| Chain strategy | stacked-to-main |

Decision needed before apply: Yes
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: High

### Suggested Work Units

| Unit | Goal | Likely PR | Notes |
|------|------|-----------|-------|
| 1 | Generalize `scanRootFiles` for multi-category | PR 1 | `internal/adapters/adapter.go` + `adapter_test.go`; kilocode stays green |
| 2 | Shrink opencode to kilocode-style wrapper | PR 2 | Delete ~280 lines; fold 4 bugfixes; add mcp.json regression test |
| 3 | Cover opencode wrapper + scanDir error branches | PR 3 | chmod 000 fixtures (skip Windows); SetScanOptions; ≥85% adapters |
| 4 | Cover config helpers + Load/Save/Get/Set errors | PR 4 | Independent; 0%→85% config; can ship first or last |

## Phase 1: Generalize GenericAdapter (PR 1 — ~150 lines)

Strict TDD: RED → GREEN → REFACTOR.

- [x] 1.1 **RED**: Add table-driven tests in `internal/adapters/adapter_test.go` for multi-category `RootFiles`: file included when category requested, excluded when not, scan runs once for multiple categories. Verify tests FAIL (compile error: `RootFiles` field missing).
- [x] 1.2 **RED**: Add test for `MaxFileSize` applied to root files (oversized file skipped + stderr warning). Verify FAIL.
- [x] 1.3 **GREEN**: Add `RootFiles map[string]string` field to `GenericAdapter` in `internal/adapters/adapter.go`. Generalize `scanRootFiles`: `cat := rootConfigFiles[name]` (default `"config"` when nil), gate `!catSet[cat]`, `Item{Category: cat}`. Invoke `scanRootFiles` ONCE per `ListItems`.
- [x] 1.4 **GREEN**: Honor `opts.MaxFileSize` in `scanRootFiles`; skip + warn to stderr. All tests pass; kilocode adapter still green.
- [x] 1.5 **REFACTOR**: Extract `catSet` build into helper if repeated; ensure `scanRootFiles` signature clean. `go test ./internal/adapters/...` green.

## Phase 2: Shrink opencode to wrapper (PR 2 — ~200 lines)

Strict TDD: RED → GREEN → REFACTOR.

- [x] 2.1 **RED**: Add regression test in `internal/adapters/opencode/adapter_test.go`: mcp.json backed up with `Category="mcp"` (hard preservation). Verify FAIL (current opencode may pass, but locks behavior).
- [x] 2.2 **RED**: Add test: opencode `ListItems` produces identical output (same RelPaths, Categories, hashes, sizes) for full category set. Snapshot current output as golden.
- [x] 2.3 **GREEN**: Delete `scanDir`, `scanRootFiles`, `ListItems`, `Backup`, `Restore` hand-rolled impls from `internal/adapters/opencode/adapter.go` (~280 lines). Replace with package-level `var base = adapters.GenericAdapter{...}` (kilocode pattern). Keep: consts, `categoryMap`, `rootConfigFiles`, wrapper delegators, `SetScanOptions`.
- [x] 2.4 **GREEN**: Wire `rootConfigFiles` into `base.RootFiles`. Verify 4 latent bugfixes folded (bare errors wrapped, unused `homeDir` removed, stderr-write continues, MaxFileSize applies to root). All opencode + kilocode tests green.
- [x] 2.5 **REFACTOR**: Remove dead code; ensure `go vet ./...` clean. Verify mcp.json regression test passes.

## Phase 3: Cover adapters error branches (PR 3 — ~150 lines)

Strict TDD: RED → GREEN.

- [x] 3.1 **RED**: Add chmod 000 fixture test in `internal/adapters/opencode/adapter_test.go`: `scanDir` error branch returns wrapped error with lowercase context. Skip on Windows (`runtime.GOOS`). Verify FAIL (branch uncovered).
- [x] 3.2 **RED**: Add stderr-write failure test: `scanDir` continues on `os.Stderr.WriteString` error (inject via `var stderrWriter` if needed). Verify FAIL.
- [x] 3.3 **GREEN**: Wrap `scanDir` stat errors with `fmt.Errorf("scan dir %s: %w", dir, err)`. Add stderr-write error handling (log to verbose, continue). Verify tests pass.
- [x] 3.4 **GREEN**: Add `SetScanOptions` test (options propagate to base). Add opencode wrapper delegation tests (Name, Detect forward). Run `go test -cover ./internal/adapters/...`; verify ≥85%.
- [x] 3.5 **Note**: `rel-error` branch (relative path computation) stays uncovered — acceptable per design (coverage >85% overall).

## Phase 4: Cover config helpers (PR 4 — ~100 lines)

Strict TDD: RED → GREEN (no production code change).

- [x] 4.1 **RED**: Add table-driven tests in `internal/config/config_test.go` for `getSettingsField`: all documented aliases resolve to canonical JSON key. Verify FAIL (tests don't compile or fail).
- [x] 4.2 **RED**: Add tests for `setSettingsField` (writes through to JSON blob), `parseBool` (rejects "yes", "2", ""). Verify FAIL.
- [x] 4.3 **RED**: Add `splitWildcard` tests: trailing `*`, leading `*`, embedded `*`, no wildcard, empty. Add `matchSegment` tests: exact, wildcard, mismatch, empty. Verify FAIL.
- [x] 4.4 **RED**: Add `Load` error tests: missing file (zero value, no error), malformed JSON (error), unreadable file (error). Add `Save` error test: unwritable directory. Add `Get`/`Set` unknown key test. Verify FAIL.
- [x] 4.5 **GREEN**: All tests pass (code already correct; tests lock behavior). Run `go test -cover ./internal/config/...`; verify ≥85%.

## Implementation Order

1. **PR 1 → PR 2 → PR 3** (sequential; each depends on previous).
2. **PR 4 is independent** — can ship first (unblocks config coverage) or last (no dependency).
3. **Stacked-to-main**: each PR merges to main; no feature branch accumulation.

## Risks

- **Test-ordering trap**: Task 2.1 (mcp regression) must run BEFORE Task 2.3 (deletion) — otherwise behavior is lost.
- **Coverage ceiling**: `rel-error` branch in `scanDir` stays uncovered; document as acceptable in PR description.
- **Windows chmod**: skip chmod 000 tests on Windows via `runtime.GOOS` check.
- **Root files now subject to MaxFileSize**: document in changelog (intended fix, not regression).
