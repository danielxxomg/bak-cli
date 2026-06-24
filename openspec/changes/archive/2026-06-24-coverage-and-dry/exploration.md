# Exploration: coverage-and-dry

> Phase: explore · Artifact store: openspec · Date: 2026-06-24
> Scope: (1) coverage gaps in `internal/config` (70.6%) and `internal/adapters/opencode` (70.6%);
> (2) DRY assessment of `scanRootFiles`/`scanDir` in `internal/adapters/opencode/adapter.go`.

## Current State

`bak-cli` enforces ≥80% coverage per `internal/` package (AGENTS.md). Two packages sit at **70.6%**:

- `internal/config` — settings/config load/save, ignore-pattern matching, v0.1→v0.2 migrations.
- `internal/adapters/opencode` — backup/restore adapter for OpenCode configs.

Both are exercised by existing table-driven tests using `t.TempDir()` real-fs and the
`configtest.SetConfigHome` helper, but several branches (error paths, settings-key parsing,
wildcard matching, the two scan functions) are untested.

The adapter layer has a **generic base** (`internal/adapters/generic.go`, `GenericAdapter`)
already consumed by `internal/adapters/kilocode` (a ~54-line thin delegating wrapper).
`internal/adapters/opencode/adapter.go` does **NOT** use it — it hand-rolls an almost identical
`scanDir` + `scanRootFiles` + `ListItems` + `Backup` + `Restore`. This is the real DRY surface.

## Affected Areas

- `internal/adapters/opencode/adapter.go` — duplicated scan/backup/restore logic (333 lines); the
  two lowest-coverage funcs (`scanDir` 56.8%, `scanRootFiles` 63.6%) live here.
- `internal/adapters/generic.go` — `GenericAdapter`, `scanDir` (82.9%), `scanRootFiles` (79.4%),
  `MatchExclude`; the consolidation target. `scanRootFiles` is hardcoded to the `"config"` category.
- `internal/adapters/kilocode/adapter.go` — existing thin-wrapper precedent (regression risk if
  `generic.scanRootFiles` is generalized).
- `internal/adapters/util.go` — shared `CopyFile`/`FileHash` helpers (already reused, no change).
- `internal/config/config.go` — `getSettingsField`/`setSettingsField`/`parseBool` at **0%**; `Save`
  71.4%, `Get` 78.3%, `Set` 76.7%, `Load` 75%, `DefaultPath` 75%.
- `internal/config/ignore.go` — `splitWildcard` **0%**, `matchSegment` 72.7% (wildcard branch).
- `internal/config/testutil/configtest.go` — `SetConfigHome` helper to reuse for config tests.

## Coverage Gap Map

### `internal/config` (70.6% → ≥80%)

| Function                | Cov.   | Untested branch / reason                                                    |
|-------------------------|--------|-----------------------------------------------------------------------------|
| `getSettingsField`      | 0.0%   | Never called — `Set("settings.*")` path untested.                           |
| `setSettingsField`      | 0.0%   | Same — the settings-key write path in `Set`.                                |
| `parseBool`             | 0.0%   | Same — bool coercion for settings fields.                                   |
| `splitWildcard`         | 0.0%   | `matchSegment` wildcard branch never reaches it.                            |
| `DefaultPath`           | 75.0%  | `UserConfigDir` error branch.                                               |
| `Load`                  | 75.0%  | `DefaultPath` error propagation branch.                                     |
| `LoadPath`              | 83.3%  | Malformed/unparseable file branch.                                          |
| `Save`                  | 71.4%  | Marshal failure + write-failure branches.                                   |
| `Get`                   | 78.3%  | `settings.*` read branch + missing nested key.                              |
| `Set`                   | 76.7%  | `settings.*` write branch + invalid bool + unknown key.                     |
| `parseSettingsKey`      | 75.0%  | Non-`settings.` prefix (false) + empty key.                                 |
| `parseNestedKey`        | 87.5%  | Edge segments.                                                              |
| `matchSegment`          | 72.7%  | `*` wildcard + `**` glob segments.                                          |
| `migrateV020`           | 80.0%  | One migration sub-branch.                                                   |
| `ParseIgnore` / `LoadExcludes` | ~90% | Comment/blank-line + IO error edges.                                |

### `internal/adapters/opencode` (70.6% → ≥80%)

| Function          | Cov.  | Untested branch / reason                                                     |
|-------------------|-------|------------------------------------------------------------------------------|
| `SetScanOptions`  | 0.0%  | Never invoked by any test.                                                   |
| `scanDir`         | 56.8% | exclude-pattern skip, MaxFileSize skip+warning, `SkipDir`, hash-error, stat-error, rel-error. |
| `scanRootFiles`   | 63.6% | `IsNotExist`→nil, exclude skip, whitelist-unrecognized skip, hash-error.     |
| `Restore`         | 72.7% | checksum-mismatch, dest-write error, mkdir error.                           |
| `Backup`          | 90.9% | One error branch.                                                           |
| `ListItems`/`Detect` | ~88% | Minor edges.                                                                |

### `internal/adapters` (root pkg, reference) — 85.8% (already passing)

`generic.scanDir` 82.9%, `generic.scanRootFiles` 79.4% — the consolidation beneficiaries.

## DRY Assessment

### Q1: Do opencode's `scanDir` (L143) and `scanRootFiles` (L222) share >70% logic?

**No — ~45% shared, below the threshold.** They share *concepts* (exclude check, canonical
path, `Item` build, `FileHash`) but the **traversal mechanics differ**: `scanDir` uses
`filepath.WalkDir` over a subdirectory tree; `scanRootFiles` uses `os.ReadDir` over flat
top-level entries with a whitelist gate. Forcing these two together would be awkward. **Do not
consolidate them with each other.**

### Q2: The real duplication — opencode ↔ `generic.go` (>70%, consolidate)

`opencode.scanDir` (L143-216) vs `generic.scanDir` (L181-249): **~88% identical.** Same
`WalkDir` skeleton, skip-root, `Rel`, `MatchExclude`, `MaxFileSize`, canonical, `Item`, `FileHash`.

`opencode.scanRootFiles` (L222-284) vs `generic.scanRootFiles` (L296-366): **~78% identical.**
Same `ReadDir`/`IsNotExist`/iterate/whitelist/exclude/canonical/`Item`/`FileHash` flow.

**Divergences (and latent bugs found in opencode):**

1. **3 bare-error returns** violate AGENTS.md `%w` wrapping:
   `scanDir:157` (`return relErr`), `scanRootFiles:228` (ReadDir `err`), `scanRootFiles:262`
   (`infoErr`). `generic` wraps all three (`"compute relative path: %w"`, `"read config dir: %w"`,
   `"stat %s: %w"`).
2. **`scanDir` aborts the entire scan if `os.Stderr.WriteString` fails** (L186-188). `generic`
   uses `fmt.Fprintf(os.Stderr,…)` and ignores the write result. Stderr write failure aborting a
   backup is a latent bug.
3. **`scanRootFiles` does NOT check `MaxFileSize`**, yet its doc comment claims it "mirrors
   scanDir's filtering." Stale doc + missing filter. `generic` checks it.
4. **Unused `homeDir` parameter** on opencode `scanDir` (passed by `ListItems`, never read).
5. **Path normalization differs**: opencode `paths.Slash(relPath)` vs generic
   `strings.ReplaceAll(relPath,"\\","/")` (the AGENTS.md-canonical form).

### Why opencode didn't already use `GenericAdapter`

`generic.scanRootFiles` is hardcoded to the `"config"` category (`!catSet["config"]` continue at
L307; `Category:"config"` at L356). OpenCode needs root files for **two** categories — `config`
(`opencode.json`) **and** `mcp` (`mcp.json`). So opencode couldn't delegate as-is.

## Approaches

1. **Consolidate opencode onto `GenericAdapter` + generalize `scanRootFiles`** *(recommended)*
   - Generalize `generic.scanRootFiles`/`ListItems` to scan root files for **any** category present
     in `RootConfigFiles` (build name→category lookup; include file only if its category ∈ `catSet`;
     set `Item.Category` to the real category). Then opencode becomes a kilocode-style thin wrapper.
   - Pros: deletes ~140 lines of duplicated scan logic; **auto-fixes all 4 latent bugs** (wrapping,
     stderr-abort, missing MaxFileSize, unused param); removes the two lowest-coverage funcs so
     opencode coverage rises structurally; kilocode already proves the pattern.
   - Cons: **behavior changes** — root files now subject to `MaxFileSize` (currently not); error
     message text changes (wrapped); stderr warning format changes. Must preserve `mcp` root-file
     behavior and re-verify kilocode (it shares the generalized `scanRootFiles`).
   - Effort: Medium.

2. **Keep opencode standalone, only extract a shared `scanDir` helper**
   - Pros: smaller blast radius; no kilocode regression surface.
   - Cons: leaves `scanRootFiles` duplication + all latent bugs; opencode coverage stays low on
     hand-rolled code; half-measure that we'd revisit.
   - Effort: Low-Medium.

3. **Coverage-only (no DRY)** — just add tests to opencode's `scanDir`/`scanRootFiles`.
   - Pros: zero behavior risk.
   - Cons: writes tests for code that **should be deleted**; if DRY is ever done, those tests are
     thrown away (wasted + merge friction). Coupling makes this the wrong first move.
   - Effort: Medium.

## Recommendation

**ONE combined change (`coverage-and-dry`), executed as strictly-ordered atomic commits** — because
the opencode coverage and DRY work are **coupled**: you must not write tests for scan code you are
about to delete. Order:

1. `refactor(adapters): generalize GenericAdapter.scanRootFiles for multi-category root files`
2. `refactor(opencode): delegate to GenericAdapter; remove duplicated scan/backup/restore`
   *(commits 1-2 fold in the bug fixes: wrapped errors, MaxFileSize on root files, stderr fix)*
3. `test(adapters): cover generic multi-category scanRootFiles + opencode wrapper + mcp preservation`
4. `test(config): cover settings-key helpers (0%→), wildcards, Load/Save/Get/Set error paths`

`internal/config` coverage is **independent** of the adapter consolidation — commit 4 can ship first
or last. If the total diff exceeds ~400 lines or reviewer focus is a concern, split per the
`chained-pr` guidance into (A) `config-coverage` (pure additive tests, zero risk) and (B)
`opencode-dry-and-coverage` (the coupled refactor). Primary recommendation stays one change.

### Test-doubles reuse (no new doubles required)

- **config**: reuse `configtest.SetConfigHome(t, dir)` (`internal/config/testutil/configtest.go`) —
  handles `XDG_CONFIG_HOME`/`APPDATA`/`HOME` per-OS. Pair with `t.TempDir()`.
- **adapters**: reuse the existing `t.TempDir()` real-fs pattern from `opencode/adapter_test.go`;
  use `adapters.MockFileSystem` (per AGENTS.md) only where error injection is needed.
- **Testability caveat**: `generic.scanDir` calls `FileHash`/`filepath.WalkDir` directly (not
  injected), so hash-error/stat-error branches are hard to reach without a real broken file (e.g. a
  dangling symlink). Reaching 100% on those may need either a crafted symlink or an injected FS var
  — a decision for the design/apply phase, not exploration.

## Risks

- **`mcp` root-file regression (HIGH)** — if the `scanRootFiles` generalization is wrong,
  `mcp.json` stops being backed up. Needs a dedicated test (back up a tree containing `mcp.json`,
  assert it appears in the manifest with `Category:"mcp"`).
- **kilocode regression (MEDIUM)** — kilocode shares the generalized `scanRootFiles`; verify its
  `RootConfigFiles`/categories still behave. Re-run kilocode tests.
- **`MaxFileSize` behavior change (LOW-MEDIUM)** — root files now filtered by `MaxFileSize`. If the
  default `ScanOptions.MaxFileSize` is `0` (disabled), no effect in practice; confirm the default bak
  uses. Document the change either way.
- **Error-string / stderr-format change (LOW)** — any test asserting exact opencode error text must
  be updated; users parsing errors are affected (rare).
- **Coverage ceiling on `generic.scanDir` error branches (LOW)** — some branches need crafted FS
  state; may cap below 100% but ≥80% is achievable.

## Ready for Proposal

**Yes.** The change is well-scoped: a coupled DRY-consolidation + coverage backfill across two
packages, with the latent bugs fixed as a side effect of consolidation. Recommend the orchestrator
hand to **sdd-propose** with the four-commit plan above and the one-split-option note.
