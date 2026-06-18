# Design: Critical Fixes

> **Note**: The proposal marked `sdd-design` as "skip" (fixes deemed additive/wiring).
> The orchestrator explicitly requested this design to lock non-trivial decisions
> (filter placement, dispatch gate semantics, redaction strategy, picker reuse,
> retention safety). This document supersedes that skip note and is authoritative
> for HOW the 9 fixes are implemented. It stays within the proposal's scope.

## Technical Approach

Additive, per-issue, test-first (strict TDD per `openspec/config.yaml`). Every
new `cmd/` path delegates business logic to `internal/actions/` (AGENTS.md
boundary). Existing abstractions are REUSED, not replaced: `ScanConfigurable` +
`ScanOptions` (Fix 1), `config.Config.Get/Set` dotted-key machinery (Fix 3),
`cmd/pick.go` standalone-picker pattern + `listBackupsFrom` (Fix 4), `FileSystem`
injectable action struct (Fix 5). Destructive ops carry `--dry-run`. No new
third-party deps. No schema migrations.

---

## Fix 1: Backup Size — `ScanOptions` in `scanRootFiles` + Codex Whitelist + `DefaultExcludes`

### Decision

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Filter inside `scanRootFiles` (generic + opencode) | Mirror `scanDir` exactly; one fix site per adapter | ✅ Chosen |
| Filter at `ListItems` caller (action level) | Would need to re-hash/re-stat items; duplicates scanDir logic | Rejected |
| New abstraction over both `scanRootFiles` variants | They already diverge (opencode has `homeDir`, whitelist); merging risks behavior change | Rejected (DRY via shared `MatchExclude` already) |

- `scanRootFiles` gains an `opts ScanOptions` parameter and applies
  `MatchExclude` + `MaxFileSize` per entry, identical to `scanDir`
  (`generic.go:174-204`, `opencode/adapter.go:143-191`).
- Codex gets a `rootConfigFiles` whitelist. Mechanism: add a
  `RootConfigFiles map[string]string` field to `GenericAdapter`
  (`generic.go:26`). When non-nil, `generic.scanRootFiles` skips any root
  entry whose name is not in the map (same semantics as opencode's
  `rootConfigFiles`, `opencode/adapter.go:83`). Codex sets it to
  `{"config.toml":"config", "instructions.md":"config", "config.json":"config", "mcp.json":"mcp"}`.
- `DefaultExcludes` (`ignore.go:19`) expands with `*.sqlite*`, `*.sqlite-wal`,
  `*.sqlite-shm`, `*.db`, `*_cache.json`, `*.jsonl`. `*.log` already present.
- `MaxFileSize` default = 1 MiB (`1048576`) **already** set by
  `DefaultSettings()` + `applyDefaults` (`config.go:73,86`). No change needed;
  the bug is solely that `scanRootFiles` ignored it.

**Rationale**: `SetScanOptions` is already wired (`backup.go:114-125`,
`cmd/backup.go:135`) and stored on the adapter — the gap is that root-file scans
never read it. Whitelist + expanded defaults are belt-and-suspenders: whitelist
prevents future runtime files leaking; excludes catch them even if a whitelist is
absent (other adapters). User `ExcludePatterns` overrides defaults per existing
`LoadExcludes` semantics (spec: "User overrides replace defaults").

### Data Flow

```
cmd/backup.go ExcludesLoader ──► actions.BackupAction ──► SetScanOptions(opts)
        │                                                       │
        └─ config.LoadExcludes (defaults + user + ignore file)   ▼
                                                    GenericAdapter.ScanOpts
                                                          │
        scanDir(opts)  ◄──already applied──┐               │
                                           └── scanRootFiles(opts)  ◄── NOW applies
                                                   │
                                      whitelist (codex) + excludes + MaxFileSize
                                                   ▼
                                            filtered Items
```

### File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/adapters/generic.go` | Modify | `scanRootFiles` takes `opts`; `GenericAdapter` gains `RootConfigFiles` field; apply `MatchExclude` + `MaxFileSize` per entry (mirror `scanDir:194-217`); stderr warning on oversized files |
| `internal/adapters/codex/adapter.go` | Modify | Set `base.RootConfigFiles` whitelist (`config.toml`, `instructions.md`, `config.json`, `mcp.json`); pass `base.ScanOpts` into `scanRootFiles` via `ListItems` (already delegated) |
| `internal/adapters/opencode/adapter.go` | Modify | `scanRootFiles` (`:221`) takes/applies `a.ScanOpts` (excludes + MaxFileSize) — whitelist already present |
| `internal/config/ignore.go` | Modify | Expand `DefaultExcludes` with `*.sqlite*`, `*.sqlite-wal`, `*.sqlite-shm`, `*.db`, `*_cache.json`, `*.jsonl` |
| `internal/adapters/generic_test.go` (or new) | Create/Modify | Table-driven: root sqlite excluded, root oversized skipped+warned, custom excludes apply, codex whitelist incl/excl, all-whitelisted-present |

### Testing Strategy

| Layer | What | Approach |
|-------|------|----------|
| Unit | `scanRootFiles` honors opts | `t.TempDir()` config root; place `logs.sqlite` (excluded), 5MB file (skipped+warned), `data.tmp` with `["*.tmp"]`; assert manifest membership |
| Unit | Codex whitelist | `~/.codex/` with `config.toml` + `logs_2.sqlite`; assert only `config.toml` item returned |
| Unit | `DefaultExcludes` | Assert new patterns present; `LoadExcludes` with `["*.tmp"]` returns only `*.tmp` (override) |
| Integration | End-to-end backup size | `t.TempDir` home + `.codex/` with sqlite DBs; run `BackupAction`; assert `manifest.TotalSize < 1MiB` and no `*.sqlite*` in manifest |

---

## Fix 2: TUI `q` Dispatch Gate (`MenuSelection.Selected`)

### Decision

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Add `Selected bool`; gate `RouteSelection`; keep inline action path | q/Esc never dispatches; inline progress UX preserved; RouteSelection cursor-0 branch becomes contract-only | ✅ Chosen |
| Make cursor-0 Enter exit + dispatch headlessly | Matches spec scenario literally but REGRESSES the in-TUI progress bar | Rejected (loses feature) |
| Delete `RouteSelection` cursor-0 branch entirely | Simplest, but loses the headless contract + spec scenarios | Rejected (spec requires the contract) |

- Add `Selected bool` to `MenuSelection` (`deps.go:75`).
- Add `selected bool` field to `Model` (`model.go:65`), default false.
- `handleMenuEnter` (`model.go:479`) sets `m.selected = true` before returning
  (covers cursors 0-6; inline action/transition proceeds as today).
- `handleKey` q/Esc/`KeyQuit` (`model.go:360`) does NOT set `selected` → stays
  false.
- `Selection()` (`model.go:627`) returns `Selected: m.selected`.
- `RouteSelection` (`dispatch.go:12`): add `if !sel.Selected { return nil }`
  immediately after the empty-`Item` guard. Existing cursor-0 `RunBackup` branch
  retained as the headless contract (exercised by unit tests; not reached by the
  live inline flow because cursor-0 Enter navigates to `ScreenProgress` without
  exiting).

**Rationale**: explore.md confirms the inline `handleMenuEnter` path is the real
backup flow; `RouteSelection`'s cursor-0 branch is "a buggy duplicate that fires
on any exit while the cursor sits at 0." The gate removes the bug without
touching working UX. `Selected=true` only on Enter satisfies both
`tui-dispatch` and `bak-cli` (MODIFIED) specs; the spec scenarios are
RouteSelection-level contracts satisfied by constructing
`MenuSelection{Selected:true, Cursor:0}` in tests (existing `dispatch_test.go`
extended with a `Selected` field).

### Data Flow

```
key press ──► Model.Update ──► handleKey
                │                  │
                │   q/Esc ─────────┼──► tea.Quit  (selected stays false)
                │   Enter ─────────┴──► handleMenuEnter: selected=true
                │                                       │
                └── (program exits) ────────────────────┤
                                                        ▼
                            Model.Selection() → MenuSelection{Selected, Cursor, Item}
                                                        │
                                                        ▼
                  RouteSelection: if !Selected → return nil  (q safe)
                                  else switch Cursor → action (contract)
```

### File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/tui/deps.go` | Modify | Add `Selected bool` to `MenuSelection` |
| `internal/tui/model.go` | Modify | Add `selected bool` field; set true in `handleMenuEnter`; `Selection()` populates `Selected` |
| `internal/tui/dispatch.go` | Modify | Guard `if !sel.Selected { return nil }` after empty-Item check |
| `internal/tui/dispatch_test.go` | Modify | Add `Selected` to table; new cases: `Selected:false, Cursor:0` → no dispatch (the q bug); `Selected:true, Cursor:0` → RunBackup; `Selected:true, Cursor:6` → no-op |
| `internal/tui/model_test.go` | Modify | Assert `Selection().Selected` is false after q, true after Enter |

### Testing Strategy

| Layer | What | Approach |
|-------|------|----------|
| Unit | `RouteSelection` gate | Table-driven (`dispatch_test.go`): `Selected:false` never calls `RunBackup` even at cursor 0; `Selected:true` cursor 0 calls it; cursor 6 no-op |
| Unit | `Model.Selection()` | Construct model, simulate q via `Update(KeyPressMsg{Code:'q'})`, assert `Selection().Selected == false`; simulate Enter, assert true |

---

## Fix 3: `config` Command (`show` / `get` / `set`) + Token Redaction

### Decision

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Reuse `config.Config.Get/Set` (already dotted for providers); extend to `settings.*` | Single dotted-key engine; type-safe; unit-testable in `config` pkg | ✅ Chosen |
| Reuse `actions.SaveSetting` (flat keys) | Different key shape; doesn't cover `settings.default_preset`; would force CLI to use flat keys | Rejected (spec mandates dotted) |
| Redact via recursive JSON-map walk | Catches any key named `token`/`api_key`/`secret`/`password` regardless of schema; future-proof | ✅ Chosen for `show` |
| Redact via typed struct deep-copy | Type-safe but misses future fields; only `ProviderConfig.Token` + legacy `GitHubToken` today | Rejected (spec: "keys containing …") |

- New `cmd/config.go`: parent `config` command + `show` (NoArgs), `get <key>`
  (ExactArgs(1)), `set <key> <value>` (ExactArgs(2)). Each delegates to
  `internal/actions/config_command.go` (AGENTS.md: cmd is thin wiring).
- `actions.ConfigShow(cfg, out)` → marshal `cfg` to indented JSON → recursive
  redaction walk → print. `config.Load()` already runs `applyDefaults`, so
  "no config file → defaults" (spec scenario) is satisfied for free.
- `actions.ConfigGet(cfg, key, out)` → `cfg.Get(key)` → redact the value if the
  leaf key name is sensitive → print. Redaction on `get` is REQUIRED (config-cli
  spec: "Any config output MUST redact…"). `get providers.x.token` returns
  `***xxxx` (intentional; raw tokens read from the config file directly).
- `actions.ConfigSet(cfg, key, value, out)` → `cfg.Set(key, value)` → `cfg.Save()`.
- **Extend `config.Config.Get/Set`** (`config.go:305/351`) to handle
  `settings.<field>` dotted keys (`default_preset`, `auto_sync`,
  `exclude_patterns`, `max_file_size`, `verbose_default`, `default_provider`,
  `confirm_destructive`). Reuses the existing `parseNestedKey` shape with a new
  `settings.` prefix branch. Unknown `settings.*` field →
  `fmt.Errorf("unknown config key: %q …")` (spec: "Set invalid key → helpful
  error").
- Redaction helper `internal/actions/redact.go`:
  `RedactString(key, val string) string` → `***` + last 4 chars (or `***`+val if
  len ≤ 4, e.g. `***ab`); `RedactJSON([]byte) []byte` → `json.Unmarshal` to
  `map[string]any`, recurse, redact any key whose lowercased name contains
  `token`/`api_key`/`secret`/`password`, re-marshal indented. Empty/missing
  values pass through unchanged.

**Rationale**: `Config.Get/Set` already implement dotted provider keys —
extending them keeps all dotted-key logic in one tested place. JSON-walk
redaction matches the spec's "keys containing …" and survives schema additions.
`actions.SaveSetting` stays the TUI's flat-key API (unchanged).

### Data Flow

```
bak config show ──► cmd/config.go ──► actions.ConfigShow(cfg, out)
                                       │
                  config.Load() ───────┘ (applyDefaults already ran)
                                       ▼
                          json.Marshal(cfg) → RedactJSON → out
                                       │
                                       └─ keys ~ token|api_key|secret|password → "***xxxx"

bak config set providers.codeberg.token xyz ──► cmd ──► actions.ConfigSet
                                                         │
                                          cfg.Set(key,val) [extended: settings.* too]
                                                         ▼
                                                   cfg.Save()
```

### File Changes

| File | Action | Description |
|------|--------|-------------|
| `cmd/config.go` | Create | `config` parent + `show`/`get`/`set` subcommands; thin RunE → `runConfig*WithDeps(cmd, args, depsFromCmd(cmd))` |
| `internal/actions/config_command.go` | Create | `ConfigShow`, `ConfigGet`, `ConfigSet` (operate on `*config.Config`; `ConfigSet` saves) |
| `internal/actions/redact.go` | Create | `RedactString`, `RedactJSON` (recursive map walk; sensitive-key matcher) |
| `internal/config/config.go` | Modify | Extend `Get`/`Set` with `settings.<field>` dotted branch; helpful errors on unknown field |
| `cmd/login.go` | Modify | Help text already references `bak config set` — now valid; optionally trim redundant lines |
| `internal/actions/config_command_test.go` | Create | Table: show redacts tokens, shows non-secrets, defaults when no file; set/get round-trip; set nested `settings.default_preset`; invalid key errors |
| `internal/actions/redact_test.go` | Create | Table over key names (`token`, `api_key`, `secret`, `password`, `tokens`), short token `***ab`, long `***7890`, JSON nesting |
| `internal/config/config_test.go` | Modify | `Get/Set` for `settings.*` keys |

### Testing Strategy

| Layer | What | Approach |
|-------|------|----------|
| Unit | Redaction | Table: `RedactString("token","ghp_abcdef1234567890")`→`***7890`; `"ab"`→`***ab`; JSON with nested `providers.*.token` all redacted; non-sensitive keys untouched |
| Unit | `ConfigShow/Get/Set` | `setConfigHome(t, dir)` + temp config; set token, show asserts `***xxxx` and raw absent; get redacts; set `settings.default_preset full` then get returns `full`; invalid key errors |
| Unit | `config.Get/Set settings.*` | Direct table on each settings field |
| E2E-ish | `cmd/config_test.go` | `cobra` `Execute` with `SetArgs` + `SetOut` buffer; assert `bak config show` output contains redacted token |

---

## Fix 4: `restore` Interactive Picker (no-arg)

### Decision

| Option | Tradeoff | Decision |
|--------|----------|----------|
| New standalone `restorePickerModel` in `cmd/` mirroring `cmd/pick.go` | Consistent with existing `pick.go`; returns ID → normal `RestoreAction` flow; small | ✅ Chosen |
| Reuse `screens.RestoreModel` as standalone | Designed as main-TUI sub-screen (ScreenBackMsg, own restore exec flow); adapting is invasive | Rejected |
| Silent most-recent fallback via `actions.ResolveBackupID` | Non-interactive surprise; spec mandates TTY picker or error | Rejected (keep `ResolveBackupID` for other callers) |

- `cmd/restore.go`: `Args: cobra.ExactArgs(1)` → `cobra.MaximumNArgs(1)`.
- `runRestoreWithDeps`: if `len(args)==0`:
  - if `!isTTY()` → `fmt.Errorf("specify a backup-id (see 'bak list') or run 'bak' for interactive mode")` (spec: non-TTY error references `bak list`).
  - else launch `restorePickerModel` (lists `listBackups()` → ID/Date/Size; ↑/↓ + Enter; q/Esc cancels) → selected ID. If cancelled → print "Restore cancelled.", return nil.
  - then proceed with existing flow (validate ID, `RestoreAction`, `--dry-run`).
- `--backup-id` flag NOT added — the positional arg remains the ID source (spec only requires no-arg picker + arg path). `--dry-run` continues to work with a selected id.

**Rationale**: `cmd/pick.go` is the established standalone-picker pattern
(`pickModel` + `tea.NewProgram` + `isTTY` guard). Mirroring it keeps the restore
picker testable (model `Update`/`View` are pure functions per AGENTS.md TUI
rules) and avoids disturbing `screens.RestoreModel`.

### Data Flow

```
bak restore (no arg)
   │
   ├─ !isTTY() ──► error "see 'bak list'"
   │
   └─ isTTY()  ──► restorePickerModel (listBackups → BackupInfo[])
                      │  ↑/↓ + Enter → SelectedID; q/Esc → cancel
                      ▼
                args = [SelectedID]  ──► existing runRestoreWithDeps
                                          │
                            IsValidBackupID → RestoreAction.ResolveBackup → Run (dry-run diff etc.)
```

### File Changes

| File | Action | Description |
|------|--------|-------------|
| `cmd/restore.go` | Modify | `MaximumNArgs(1)`; no-arg branch: `isTTY` → picker → ID; non-TTY → `bak list` error |
| `cmd/restore_picker.go` | Create | `restorePickerModel` (Init/Update/View), `SelectedID()`, `WindowSizeMsg` handling, "terminal too small" guard (<20×10), help bar with `? help` |
| `cmd/restore_picker_test.go` | Create | Table-driven model `Update`/`View`: ↑/↓ cursor bounds, Enter sets SelectedID, q/Esc cancels, empty backups, narrow terminal |
| `cmd/restore_test.go` | Modify | With `isTTY` override: no-arg TTY path returns picked ID; non-TTY errors mentioning `bak list`; `restore <id> --dry-run` still works |

### Testing Strategy

| Layer | What | Approach |
|-------|------|----------|
| Unit | Picker model | Pure `Update`/`View` tests (AGENTS.md: never test `Program.Run`): cursor clamp, Enter selection, q cancel, empty list, narrow terminal message |
| Unit | `runRestoreWithDeps` | `isTTY` var override (`cmd/tty_test.go` pattern); inject picker that returns a fixed ID; assert `RestoreAction` runs with it; non-TTY asserts error text contains `bak list` |

---

## Fix 5: `cleanup` Retention (`--keep N --dry-run [--force]`)

### Decision

| Option | Tradeoff | Decision |
|--------|----------|----------|
| New `CleanupAction` in `internal/actions/` (injectable `FileSystem`) | Testable, follows `BackupAction`/`RestoreAction` pattern; cmd is thin | ✅ Chosen |
| Implement listing in `cmd/` reusing `listBackupsFrom` | `listBackupsFrom` returns `tui.BackupInfo` (loads manifests, formats size) — heavier than needed; not in `internal/actions` | Rejected for core logic (action lists dirs directly) |
| `--dry-run` default ON | Conflicts with spec scenarios (`--keep 3 --force` deletes without `--dry-run`) | Rejected — `--dry-run` is opt-in flag |
| Cloud cleanup | Out of scope (proposal: no `settings.retention`); cloud sync is push/pull-based | Out of scope (documented) |

- New `internal/actions/cleanup.go`: `CleanupAction{ FS, BackupsDir string, Keep int, DryRun, Force bool, Confirm func() bool, Stdout, Stderr io.Writer }`.
- `Run()`:
  1. `FS.ReadDir(BackupsDir)` → collect dir names (skip non-dirs; skip entries without valid `manifest.json`? spec implies all backup dirs — keep simple: all subdirs).
  2. Sort IDs descending (lexicographic == chronological; `YYYYMMDD-HHMMSS` per `backup.go:129`).
  3. `keep := Keep`; if `Keep <= 0` → default (3 per proposal; spec "Default keep value"). Split `toDelete = ids[keep:]`.
  4. If `len(toDelete)==0` → print "Nothing to clean (N backups, keeping N).", return.
  5. If `DryRun` → print "Would delete N backups (keeping K):" + each ID; return (0 deletions).
  6. If `!Force`: if `Confirm == nil` → `fmt.Errorf("cleanup requires --force or a TTY (use --dry-run to preview)")` (spec: non-TTY without force errors). else if `!Confirm()` → print "Cleanup cancelled.", return.
  7. Delete each: `FS.RemoveAll(filepath.Join(BackupsDir, id))`; log `deleted <id>` to stderr; count failures.
  8. Print summary "Deleted N/K backups (M failed)."
- New `cmd/cleanup.go`: `cleanupCmd` with `--keep int` (default 3), `--dry-run bool`, `--force bool`. `runCleanupWithDeps` resolves `backup.BakDir()` + `filepath.Join(bakDir,"backups")`, wires `Confirm` = TTY prompt (`fmt.Fprintf(out, "Delete %d backups? [y/N]: ", n)` + `bufio.NewReader(stdin)`, only when `isTTY()`); non-TTY → `Confirm=nil` so action errors helpfully.

**Rationale**: `CleanupAction` mirrors the injectable-action pattern
(`FS FileSystem`, `Stdin/out/err`) → fully unit-testable with `MockFileSystem`
and an injected `Confirm` func (no real TTY in tests). `--dry-run` flag satisfies
AGENTS.md "destructive ops MUST have --dry-run". Spec's TTY-prompt / non-TTY-error
/ `--force`-skips is captured by the `Confirm` injection. Local-only keeps blast
radius small.

### Data Flow

```
bak cleanup --keep 3 [--dry-run] [--force]
        │
   cmd/cleanup.go: backup.BakDir() → BackupsDir; isTTY() → Confirm fn
        ▼
   actions.CleanupAction.Run()
        │
   FS.ReadDir ──► sort desc ──► toDelete = ids[keep:]
        │
   ├─ DryRun            ──► print plan, 0 deletions
   ├─ !Force && Confirm ──► "Delete N? [y/N]" → y proceeds / N cancels
   ├─ !Force && nil     ──► error "use --force or TTY"
   └─ Force             ──► FS.RemoveAll each (stderr log) → summary
```

### File Changes

| File | Action | Description |
|------|--------|-------------|
| `cmd/cleanup.go` | Create | `cleanupCmd` + `--keep`/`--dry-run`/`--force` flags; `runCleanupWithDeps` wires `BackupsDir` + `Confirm` (TTY-only) |
| `internal/actions/cleanup.go` | Create | `CleanupAction` struct + `Run()`; default keep=3; plan/dry-run/confirm/delete flow |
| `internal/actions/cleanup_test.go` | Create | Table: keep 3 of 10 deletes 7; keep > count deletes 0; dry-run lists + 0 deletions; force skips prompt; Confirm false cancels; Confirm nil + !force errors; ReadDir error propagates; RemoveAll failure counted |
| `cmd/cleanup_test.go` | Create | `isTTY` override; `cobra.Execute` with `--dry-run` asserts plan text + no FS change; `--force` with temp `~/.bak/backups` deletes |
| `cmd/root.go` | Modify | `rootCmd.AddCommand(cleanupCmd)` (via `cleanup.go init()`) |

### Testing Strategy

| Layer | What | Approach |
|-------|------|----------|
| Unit | `CleanupAction` | `MockFileSystem` with temp tree of 10 backup dirs; table over `Keep`, `DryRun`, `Force`, `Confirm` (nil/true/false); assert kept/deleted sets, stderr logs, error messages |
| Unit | Edge cases | 0 backups; `Keep=0` → default 3; ReadDir error; RemoveAll partial failure (count failed) |
| Integration | `cmd/cleanup` | `setConfigHome` + temp `~/.bak/backups`; `--dry-run` lists N, 0 removed; `--force` removes exactly `toDelete` |

---

## Minor Fixes (in scope, low-risk)

| Fix | File | Change |
|-----|------|--------|
| `--version` flag | `cmd/version.go` | `init()`: `rootCmd.Version = Version; rootCmd.SetVersionTemplate("bak {{.Version}}\n")`. Keep `version` subcommand for verbose output. Spec: both work. |
| `profile create` no-arg wizard | `cmd/profile.go` | `Args: MaximumNArgs(1)`; `runProfileCreateWithDeps`: if `len(args)==0 && --interactive` → call `runProfileCreateInteractiveWithDeps(cmd, "", deps)` and use `wm.ProfileName()` as name when arg empty (wizard's NameStep collects it); if `len(args)==0 && !--interactive` → error "provide a profile name or use --interactive". `actions.ProfileCreateInteractive` already accepts `name`; pass `wm.ProfileName()` when arg was empty. |
| Footer `? help` | `internal/tui/screens/menu.go` | Append `{Key: "?", Desc: "help"}` to `helpKeys` (`:46-50`). Existing `↑/↓ navigate`, `enter select`, `q quit` preserved. |
| OAuth `error_description` | `internal/cloud/httputil.go` | `formatAPIError`: try `json.Unmarshal` into `struct{ Error, ErrorDescription, Message string }`; if `error_description` non-empty use `error_description`; else if `error` non-empty use `error`; else current body/statusText fallback. Surfaced in `cmd/login.go` error chain. |

---

## Overall Testing Strategy

- **Strict TDD** (`openspec/config.yaml: testing.strict_tdd: true`): RED → GREEN → REFACTOR per fix. Run `go test ./...` after each.
- **Table-driven** unit tests for every new pure function (`scanRootFiles` opts, `RouteSelection` gate, `RedactString/JSON`, `ConfigGet/Set`, picker model, `CleanupAction`).
- **Isolation**: `t.TempDir()` for all FS tests; `setConfigHome(t, dir)` for config tests (never touch real `~/.config/bak/`); `isTTY` var override for TTY branches; injected `FileSystem`/`Confirm`/`Picker` for action tests.
- **Coverage**: ≥80% for new `internal/` packages (`config_command.go`, `redact.go`, `cleanup.go`) and ≥80% `internal/tui/` (model `Selection`, `RouteSelection`). `cmd/` covered via `cobra.Execute` + buffer asserts (no `os.Exit` unit tests — E2E only per AGENTS.md).
- **Cross-platform**: `scanRootFiles` opts tests use `strings.ReplaceAll(path, "\\", "/")` canonicalization (AGENTS.md); `CleanupAction` uses `filepath.Join`; picker handles `WindowSizeMsg`. No OS-specific code introduced.
- **Verification command**: `go test ./...` + `golangci-lint run` + `go build -o bak.exe .` (per `openspec/config.yaml`).

---

## Risks & Mitigations

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Expanded `DefaultExcludes` drops a file a user wants | Med | `Settings.ExcludePatterns` overrides defaults (existing `LoadExcludes`); integration test asserts override semantics; `!`-negation in ignore file re-includes |
| Codex whitelist omits a future config file | Low | Whitelist is explicit + versioned; expanding it is a one-line change; `DefaultExcludes` is the safety net; documented in `codex/adapter.go` |
| `RouteSelection` gate breaks a headless caller | Low | No current headless caller needs unguarded dispatch (`defaultRunTUI` is the only caller); `dispatch_test.go` covers `Selected:true` contract; gate is additive |
| `config get` redaction makes token retrieval impossible via CLI | Low | Intentional per security rules; raw token readable from `~/.config/bak/config.json` directly; documented in `cmd/config.go` help |
| JSON-walk redaction misses a sensitive key | Low | Matcher lowercases and checks `contains` over `token`/`api_key`/`secret`/`password`; table test covers each + nested; future fields auto-covered by walk |
| `cleanup --force` deletes wrong backups | Med | `--dry-run` preview first (spec scenario: dry-run then force deletes exactly those 7); descending sort unit-tested; `Confirm` gate on TTY; default `--keep 3` conservative; `setConfigHome` isolation in tests |
| `restore` no-arg picker blocks scripts | Low | Non-TTY errors with `bak list` hint; TTY path is opt-in (only when no arg); arg path unchanged |
| `profile create` empty-name wizard saves empty profile key | Low | Use `wm.ProfileName()` when arg empty; wizard NameStep enforces non-empty before confirm (existing `wizard_test.go` covers NameStep); error if wizard returns empty name |

## Migration / Rollout

No migration required. All fixes additive/isolated — revert per-commit. No
config schema change (only `Get/Set` learn `settings.*` keys; on-disk format
unchanged). `DefaultExcludes` change affects only future backups; existing
backups untouched. `bak cleanup` is brand new (nothing to roll back).

## Open Questions

- [ ] `cleanup`: should it skip backup dirs whose `manifest.json` is missing/corrupt, or delete them too? **Recommendation**: delete them too (they're incomplete/corrupt) but warn on stderr. Confirm with team.
- [ ] `config set` for `settings.exclude_patterns` (a JSON array) — accept comma-separated CLI value and split, or require JSON literal? **Recommendation**: accept comma-separated for ergonomics (`bak config set settings.exclude_patterns "*.tmp,*.bak"`), split on `,`. Confirm.
- [ ] Should `config get` on a non-existent provider (e.g. `providers.codeberg.token` when codeberg unset) error or print empty? Current `Config.Get` errors ("provider not configured"). **Recommendation**: keep error (helpful). Confirm.

## Next Step

Ready for `sdd-tasks` to break each fix into TDD-ordered implementation tasks.
