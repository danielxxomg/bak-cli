# Change: coverage-and-dry

> Phase: propose · Artifact store: openspec · Date: 2026-06-24
> Authoritative input: `openspec/changes/coverage-and-dry/exploration.md`

## Why

`bak-cli` enforces ≥80% coverage per `internal/` package (AGENTS.md). Two packages sit at
**70.6%** — `internal/config` and `internal/adapters/opencode`. The prior session left these as the
technical Next Steps; they are now blocking the coverage gate.

The two gaps are **coupled**, not independent. In `internal/adapters/opencode/adapter.go` the two
lowest-coverage functions — `scanDir` (56.8%) and `scanRootFiles` (63.6%) — are **exactly** a
re-implementation of logic that already exists in `internal/adapters/generic.go`
(`GenericAdapter`), to which `internal/adapters/kilocode` already delegates. Writing tests for the
opencode copies would mean testing code that **should be deleted** — those tests would be thrown
away the moment the duplication is removed. So coverage and DRY must be decided together, in one
change.

Consolidating opencode onto `GenericAdapter` also removes **four latent bugs** that live in the
duplicated code (3 bare-error returns without `%w`; `scanDir` aborting the whole scan on an
`os.Stderr.WriteString` failure; `scanRootFiles` silently ignoring `MaxFileSize` despite its doc
comment; an unused `homeDir` parameter). The only reason opencode does not already delegate is that
`generic.scanRootFiles` is hardcoded to the `"config"` category, while OpenCode needs root files for
both `config` (`opencode.json`) and `mcp` (`mcp.json`). Generalizing that one function unblocks the
whole consolidation.

## Decisions (confirmed this session)

1. **Single combined change** `coverage-and-dry` (not split). Coverage config + opencode, DRY
   consolidation, and the 4 latent bugfixes all in one change.
2. **Preserve visible behavior.** Error-message text and stderr format stay unchanged unless the
   consolidation structurally forces a change. No gratuitous message rewriting. Any unavoidable
   change is listed below as a behavior-change note.
3. **Fix all 4 latent opencode bugs in this change** (they are in the code being removed).
4. **Coverage target 85%+** for both `internal/config` and `internal/adapters/opencode` — margin
   above the 80% AGENTS.md floor so the next change does not re-fail the gate.

## What Changes

### 1. Generalize `GenericAdapter.scanRootFiles` to multi-category root files
`internal/adapters/generic.go` — `scanRootFiles` (and the `ListItems` root branch) is today
hardcoded to the `"config"` category. Generalize it: build a name→category lookup from
`RootConfigFiles`; include a root file only if its category ∈ the active `catSet`; set
`Item.Category` to the real category. This keeps `kilocode` (config-only) working and unblocks
opencode (config + mcp).

### 2. Make `opencode` delegate to `GenericAdapter` (kilocode-style thin wrapper)
Delete opencode's hand-rolled `scanDir` / `scanRootFiles` / `ListItems` / `Backup` / `Restore`
duplicate implementations in `internal/adapters/opencode/adapter.go`. opencode becomes a thin
delegating wrapper around `GenericAdapter` (matching the precedent already set by kilocode). This
removes ~140 lines of duplicated logic. Consolidation **folds in the 4 bugfixes** as a side effect
of deleting the buggy copies:
- Wraps the 3 bare-error returns with `%w` (via generic's already-correct wrapping).
- Stops `scanDir` from aborting the scan on an `os.Stderr.WriteString` failure (generic uses
  `fmt.Fprintf(os.Stderr,…)` and ignores the write result).
- Applies `MaxFileSize` to root files consistently (generic checks it; opencode's copy skipped it).
- Drops the unused `homeDir` parameter.

### 3. Adapter coverage: `internal/adapters/opencode` 70.6% → 85%+
After consolidation, cover the thin wrapper + the generalized `generic.scanRootFiles`:
- `SetScanOptions` invocation + assertion (currently **0%**).
- Multi-category `scanRootFiles`: include/exclude by category; `IsNotExist`→nil on empty config dir.
- **`mcp` root-file preservation** — back up a tree containing `mcp.json`, assert it appears in the
  manifest with `Category:"mcp"` (guards the HIGH regression risk).
- `scanDir` error branches: exclude-pattern skip, `MaxFileSize` skip + warning, `SkipDir`,
  hash-error, stat-error, rel-error.
- `Restore`: checksum mismatch, dest-write error, mkdir error.
- kilocode regression check (shares the generalized `scanRootFiles`).

### 4. Config coverage: `internal/config` 70.6% → 85%+
Add table-driven tests (reuse `configtest.SetConfigHome` + `t.TempDir()`; no new doubles):
- The **0% helpers**: `getSettingsField`, `setSettingsField`, `parseBool` (the entire
  `Set("settings.*")` path is dead to tests today), `splitWildcard`.
- `matchSegment` wildcard branches (`*`, `**`).
- Error paths in `Load` (`DefaultPath`/`LoadPath` error propagation, malformed file),
  `Save` (marshal failure, write failure), `Get` (`settings.*` read, missing nested key),
  `Set` (`settings.*` write, invalid bool, unknown key).

### Non-goals
- No new features, no CLI surface changes, no TUI changes.
- No behavior change to backup/restore output beyond what the consolidation structurally forces.
- No coverage work on already-passing packages (adapters root 85.8%).
- No split into separate changes (decided).

## Impact

- **Affected specs (deltas to be created in sdd-spec):**
  - `internal/adapters` — generic adapter multi-category root-file behavior (new capability + spec
    delta), opencode as delegating wrapper (behavior delta).
  - `internal/config` — no behavior change; spec coverage via tests only.
- **Affected code:**
  - `internal/adapters/generic.go` — generalize `scanRootFiles`/`ListItems` (refactor).
  - `internal/adapters/opencode/adapter.go` — replace hand-rolled impls with delegation (refactor;
    deletes ~140 lines; removes 4 latent bugs).
  - `internal/adapters/opencode/adapter_test.go` — new/updated tests.
  - `internal/adapters/generic_test.go` (and kilocode tests) — multi-category + regression tests.
  - `internal/config/*_test.go` — new tests for helpers + error paths.
- **Behavior-change notes (unavoidable, called out per decision 2):**
  1. Root files (`opencode.json`, `mcp.json`) now subject to `MaxFileSize`. No-op in practice if
     the default `ScanOptions.MaxFileSize` is `0` (disabled) — confirm default in design.
  2. 3 error returns now wrapped with `%w` (e.g. `compute relative path: %w`).
  3. Stderr warning format follows generic's form (changed only where opencode's copy differed).
  Any test asserting exact opencode error/stderr text must be updated; users parsing errors are
  affected (rare).

## Intended Approach (commit-level; refined in sdd-tasks)

Strictly-ordered atomic commits — do not test duplicate code that is about to be deleted:

1. `refactor(adapters): generalize GenericAdapter.scanRootFiles for multi-category root files`
2. `refactor(opencode): delegate to GenericAdapter; remove duplicated scan/backup/restore`
   *(commits 1–2 fold in the 4 bugfixes)*
3. `test(adapters): cover generic multi-category scanRootFiles + opencode wrapper + mcp preservation`
4. `test(config): cover settings-key helpers (0%→), wildcards, Load/Save/Get/Set error paths`

Strict TDD: coverage tests follow red/green; refactor commits keep tests green.
Test doubles: reuse `MockFileSystem`, `t.TempDir()`, `configtest.SetConfigHome` — no new doubles.
**Design-phase dependency (to resolve in sdd-design):** `generic.scanDir` calls `FileHash`/
`filepath.WalkDir` directly (not injected), so some error branches may need a crafted fixture
(e.g. dangling symlink) or an injected FS var to reach. Design decides injection vs crafted fixture.

## Delivery

- `delivery_strategy: ask-always` — chained-PR decision deferred to the sdd-tasks Review Workload
  Guard (if forecast >400 lines or high risk, ask before apply).
- `review_budget_lines: 800`.
- Strict TDD active.

## Risks

- **`mcp` root-file regression (HIGH)** — if `scanRootFiles` generalization is wrong, `mcp.json`
  stops being backed up. Guard with the dedicated test in change 3.
- **kilocode regression (MEDIUM)** — kilocode shares the generalized `scanRootFiles`; re-run its
  tests, verify `RootConfigFiles`/categories still behave.
- **`MaxFileSize` behavior change (LOW–MEDIUM)** — root files now filtered; confirm default
  `ScanOptions.MaxFileSize==0` makes it a no-op; document either way.
- **Error-string / stderr-format change (LOW)** — update tests asserting exact text only where
  forced.
- **`generic.scanDir` error-branch coverage ceiling (LOW)** — some branches need crafted FS
  state; ≥85% achievable, may cap below 100%.

## Next

Ready for **sdd-spec** (delta specs: `internal/adapters`, `internal/config`) and **sdd-design**
(consolidation mechanics, multi-category lookup, scanDir injection decision) — both depend only on
the proposal and may run in parallel.