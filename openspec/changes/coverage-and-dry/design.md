# Design: coverage-and-dry

> Phase: design · Artifact store: openspec · Date: 2026-06-24
> Authoritative: `openspec/changes/coverage-and-dry/{proposal,exploration}.md`

## Technical Approach

Delete `opencode.scanDir`, `scanRootFiles`, `ListItems`, `Backup`, `Restore` and reduce the opencode adapter to a ~54-line `GenericAdapter` delegate (kilocode precedent). Before deletion, **generalize `generic.scanRootFiles`** to do the per-entry name→category lookup opencode already does, so generic supports multi-category root files (mcp.json→`mcp`, opencode.json→`config`) instead of hardcoding `config`. This single junction makes opencode delegatable, folds four latent bugs (generic's versions are already correct), and moves the chmod-000 error-branch fixtures into `generic_test` to push `internal/adapters` ≥85%.

## Architecture Decisions

| Decision | Option | Tradeoff | Chosen |
|----------|--------|----------|--------|
| Root multi-category model | (a) keep opencode's lookup, generalize generic | one shared impl | **generalize generic** — opencode's logic lifted into generic |
| Per-entry category gate | keep `!catSet["config"]` vs per-entry `cat` | nil-whitelist kilocode legacy | **per-entry `cat`**, default `"config"` when `RootConfigFiles==nil` (preserves kilocode) |
| scanRootFiles trigger | hardcoded `catSet["config"]` vs "any root-cat requested" | opencode needs mcp-only | **derive from `RootConfigFiles` values; nil⇒`catSet["config"]`** |
| opencode wrapper | keep bespoke logic vs delegate | maintainability | **delegate** (kilocode shape) |
| Error-branch coverage seam | (a) inject `fileHashFn`/walk fn vs (b) chmod-000 fixtures | CI determinism vs prod change | **fixtures** — matches existing `StatFn`-only-for-Detect precedent & opencode_test L378-431 |

## Data Flow

```
Adapter.ListItems(home, cats)
  ├─ for each CategoryDir with IsDir==true → ga.scanDir(sub, cat)   [unchanged]
  └─ if rootScanRequested:  ga.scanRootFiles(configDir, catSet, opts, rootConfigFiles)
        for entry (skip dirs):
          cat = "config"  (default)
          if rootConfigFiles != nil: cat,recognized = rootConfigFiles[name]; skip if !recognized
          if !catSet[cat]: skip          ← per-entry gate (was hardcoded "config")
          excludes · MaxFileSize · stat(wrap) · hash(rel-path,wrap)   [unchanged, generic already correct]
          Item{Category: cat, ...}       ← use looked-up cat
```

`rootScanRequested = (RootConfigFiles!=nil && any value-cat ∈ catSet) || (RootConfigFiles==nil && catSet["config"])`.

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/adapters/generic.go` | Modify | Generalize `scanRootFiles` gate/trigger/`Item.Category`; generalize `ListItems` trigger. ~15 lines. |
| `internal/adapters/generic_test.go` | Modify | Add multi-category + mcp-preservation + moved chmod-000 hash/stat fixtures. |
| `internal/adapters/opencode/adapter.go` | Modify (shrink ~280→~54) | Delete `scanDir`/`scanRootFiles`/`ListItems`/`Backup`/`Restore`; keep consts, `categoryMap`, `rootConfigFiles`, wrapper delegators. |
| `internal/adapters/opencode/*_test.go` | Modify | Keep mcp/opencode.json behavioral tests; delete tests that asserted deleted private helpers; keep chmod-000 tests OR move to generic_test. |

## Interfaces / Contracts

```go
// generic.scanRootFiles — generalized gate (signatures unchanged)
cat, recognized := "config", true
if rootConfigFiles != nil {
    cat, recognized = rootConfigFiles[entryName]
    if !recognized { continue }
}
if !catSet[cat] { continue }                     // was: !catSet["config"]
// ... unchanged exclude/MaxFileSize/stat/hash ...
items = append(items, Item{Category: cat, ...})  // was: Category: "config"
```
`RootConfigFiles map[string]string` (name→category) is **already** the declared type; no struct change needed.

## Bugs Folded In (all: generic already correct, opencode buggy)

1. `scanDir` bare `return relErr` → generic wraps `compute relative path: %w`.
2. `scanRootFiles` bare `return nil, err` (ReadDir) → generic wraps `read config dir: %w`.
3. `scanRootFiles` bare `return nil, infoErr` (stat) → generic wraps `stat %s: %w`.
4. `scanDir`+`scanRootFiles` hash error leaks `absPath` (home dir path) → generic uses rel path.

**Explicit behavior delta**: opencode root files (`opencode.json`/`mcp.json`/`AGENTS.md`) were NEVER size-gated; generic applies `MaxFileSize`. **Default confirmed**: `DefaultSettings().MaxFileSize = 1 MiB` (1048576), returned by `LoadExcludes`. Real-world impact ≈ nil (root configs ≪ 1 MiB); call out in changelog as a deliberate fix.

## Testing Strategy (Strict TDD — red first)

| Layer | What | How |
|-------|------|-----|
| Unit | multi-category root scan (config+mcp) | fixture: `opencode.json`,`mcp.json` at root → assert `Item.Category` per file + both present |
| Unit | mcp preservation regression | backup tree with both files → manifest has both, correct Category, correct hash |
| Unit | kilocode regression | re-run kilocode tests unchanged; assert single-category path through generalized fn |
| Unit | hash/stat error branches | chmod-000 file (hash), chmod-000 subdir (walk) — skipped on Windows via `runtime.GOOS` |
| Unit | MaxFileSize root delta | root file >1 MiB → skipped w/ stderr warn |
| Acceptance | rel-error branch | accepted uncovered (practically unreachable: only cross-volume Windows) |

## Migration / Rollout

No migration. No flag. Behavior delta (root MaxFileSize) is the intended fix; default 1 MiB means existing root configs still backed up.

## Open Questions

- [ ] opencode keeps private consts (`adapterName`,`categoryMap`,`rootConfigFiles`); kilocode exports them. Align names? Out of scope for this change — minimize diff.
- [ ] Move opencode chmod-000 tests into `generic_test` (DRY) vs keep duplicated in opencode_test — task-level decision.