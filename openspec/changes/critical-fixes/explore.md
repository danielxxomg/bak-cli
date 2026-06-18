# Exploration: bak-cli manual-testing critical fixes

Deep investigation of 9 issues found during manual testing of the current
build (`./bak`, built 2026-06-18 13:59, includes commit `45a03b4`).

Each issue traces the real call path through source code with exact
file:line references. All claims verified against source + live repro.

---

## Issue 1: BACKUP SIZE STILL 8.4MB (CRITICAL)

- **Symptom**: `bak backup` (quick preset) produces a 8.37 MB backup with
  33 files, even though the exclusion engine was wired in `wiring-fixes`.
- **Root cause**: THREE compounding defects.
  1. `internal/adapters/generic.go:286-325` — `scanRootFiles()` backs up
     **every regular file in the adapter config root** as the `"config"`
     category, with **NO exclude-pattern check and NO MaxFileSize check**.
     Only `scanDir()` (the recursive variant, `generic.go:174-204`) applies
     `opts.Excludes` / `opts.MaxFileSize`. So `SetScanOptions()` (wired in
     `cmd/backup.go:135-152` and `internal/actions/backup.go:114-125`) is a
     **no-op for root-level config files**. The ExcludesLoader runs, the
     patterns are returned, the adapter stores them — but `scanRootFiles`
     never reads them. Same gap in `internal/adapters/opencode/adapter.go:221-264`
     (no MaxFileSize; opencode is saved by its `rootConfigFiles` whitelist).
  2. `internal/adapters/codex/adapter.go:17-20` — the codex adapter maps
     `"config": {SubPath: "", IsDir: false}` with **no file whitelist**
     (unlike opencode's `rootConfigFiles`). So every file in `~/.codex/`
     root is treated as "config".
  3. `internal/config/ignore.go:19-34` — `DefaultExcludes` does not cover
     runtime databases/caches: `*.sqlite`, `*.sqlite-wal`, `*.sqlite-shm`,
     `*.db`, `*_cache.json`.

  **Evidence (live manifest `~/.bak/backups/20260618-190509/manifest.json`):**
  `TotalSize: 8775092` (8.37 MB), `FileCount: 33`, `Preset: quick`,
  `Categories: ["config"]`. The bloat is entirely the **codex** adapter:
  - `codex/logs_2.sqlite-wal` — 4 342 512 B (4.1 MB)
  - `codex/logs_2.sqlite` — 2 494 464 B (2.4 MB)
  - `codex/state_5.sqlite-wal` — 1 161 872 B (1.1 MB)
  - `codex/state_5.sqlite` — 180 224 B
  - `codex/models_cache.json` — 140 474 B
  - `codex/memories_1.sqlite-wal`, `goals_1.sqlite-wal`, `*.sqlite-shm` …

  These are Codex runtime logs/state/memories SQLite DBs — not user config.

- **Fix** (all three needed):
  1. Make `scanRootFiles()` honor `ScanOptions` — pass `opts` into it and
     apply `MatchExclude` + `MaxFileSize` per entry (mirror `scanDir`).
     Do this in `generic.go` AND `opencode/adapter.go`.
  2. Add a `rootConfigFiles`-style whitelist to the codex adapter (only
     back up real config: `config.toml`, `instructions.md`, `config.json`,
     `mcp.json` — NOT `*.sqlite*`, `*_cache.json`, `*history*`).
  3. Expand `DefaultExcludes` with runtime/cache patterns:
     `*.sqlite`, `*.sqlite-wal`, `*.sqlite-shm`, `*.db`, `*-shm`, `*-wal`,
     `*_cache.json`, `*.jsonl` (history). Optionally set a default
     `MaxFileSize` (e.g. 1 MB) as a safety net.
- **Effort**: medium (the exclude-wiring is correct; the gap is that
  `scanRootFiles` bypasses it + codex has no whitelist + defaults too narrow).

---

## Issue 2: OAUTH LOGIN FAILS — `api error 400` (CRITICAL)

- **Symptom**: `./bak login` returns
  `oauth login: device code: device code: api error 400`.
- **Root cause**: RESOLVED in the current build. The 400 was caused by a
  **missing/invalid OAuth `client_id`**: before commit `45a03b4`
  (2026-06-18 12:22, "feat(cmd): hardcode GitHub OAuth Client ID"), the
  Device Flow was only wired when `BAK_GITHUB_OAUTH_CLIENT_ID` was set
  (`cmd/login.go` old code: "Wire OAuth Device Flow if client ID env var
  is set"). With the env var unset, no valid `client_id` reached
  `internal/cloud/oauth_device.go:169-191` (`requestDeviceCode`), so
  GitHub's `POST /login/device/code` returned 400.

  The double `device code:` prefix matches the wrap chain:
  `requestDeviceCode` (`oauth_device.go:180`) → `RequestToken`
  (`oauth_device.go:93`); the `api error 400` suffix comes from
  `formatAPIError` (`internal/cloud/httputil.go:53-58`).

  The request itself is correct: URL `https://github.com` + `/login/device/code`
  (`oauth_device.go:15,179`), `Accept: application/json`,
  `Content-Type: application/x-www-form-urlencoded`, `scope=gist`
  (`httputil.go:14-29`, `oauth_device.go:174-176`). Verified live: with the
  current binary the device-code step **succeeds** (returns a user code,
  e.g. `C2C3-2F6C`) and polls — no 400.

- **Remaining risks** (not the original 400, but worth fixing):
  1. `cmd/login.go:75-78` — `BAK_GITHUB_OAUTH_CLIENT_ID` **overrides** the
     hardcoded default `Ov23liGOBgrjOlus0xwt`. If a user has a stale/invalid
     env var, the 400 returns. Should fall back to the hardcoded default
     only when the env var is empty (it does) — but document it / warn.
  2. The token-exchange step (`pollAccessToken`, `oauth_device.go:194-206`)
     after browser authorization is **unverified** (could not complete
     browser auth in this environment).
  3. The client ID is embedded in the binary as a shared value; if the
     OAuth App is revoked or rate-limited, all users break.
- **Fix**: No code fix required for the 400 itself (already fixed by
  `45a03b4`). Recommend: (a) verify the token-exchange path end-to-end with
  a real browser authorize; (b) add a clearer error when GitHub returns 400
  (surface `error_description` instead of raw body); (c) document the env
  var override.
- **Effort**: small (verification + error-message polish) — but flag as
  "needs end-to-end browser-auth verification" before calling it done.

---

## Issue 3: TUI 'q' CREATES BACKUP (HIGH)

- **Symptom**: Pressing `q` in the TUI creates a backup instead of quitting.
- **Root cause**: Post-exit dispatch fires on cursor position, not on an
  explicit selection.
  - `internal/tui/model.go:356-371` — in `ScreenMenu`, `q` (`KeyQuit`,
    `keys.go:7`) correctly returns `tea.Quit`. So the TUI **does quit**.
  - `cmd/tty.go:18-33` — `defaultRunTUI` calls
    `tui.RouteSelection(model.Selection(), deps)` **after** the program
    exits.
  - `internal/tui/dispatch.go:12-24` — `RouteSelection` runs
    `deps.RunBackup(nil, nil)` whenever `sel.Cursor == 0`.
  - `internal/tui/model.go:627-642` — `Selection()` returns the current
    cursor, whose **default is 0** (`NewModel`, `model.go:108`), and menu
    item 0 is `"Create backup"` (`deps.go:85`).

  So: `q` quits the TUI → `RouteSelection` sees `Cursor==0` → runs a
  **headless backup** (`RunBackup(nil, nil)`, no progress channel). The
  in-TUI "Create backup" path (`handleMenuEnter` case 0, `model.go:481-493`)
  is a *different* code path that shows progress and does NOT exit — so
  `RouteSelection`'s backup branch is effectively a buggy duplicate that
  fires on any exit while the cursor sits at 0.

- **Fix**: Gate dispatch behind an explicit "confirmed selection" flag.
  Add a `Selected bool` (or `Confirmed bool`) to `MenuSelection`, default
  `false`. Set it `true` **only** in the path that is meant to dispatch
  headlessly (currently no such path — `handleMenuEnter` handles actions
  inline). Then `RouteSelection` does `if !sel.Selected { return nil }`
  before any dispatch. This makes `q`/`Esc`/the "Quit" menu item all safe.
  (Alternatively remove the `Cursor==0` backup branch entirely since the
  TUI already runs backups inline with progress — but keep the gate for
  future headless actions.)
- **Effort**: small (model flag + one guard in `dispatch.go` + tests).

---

## Issue 4: `--version` FLAG (MEDIUM)

- **Symptom**: `bak --version` → `Error: unknown flag: --version`.
  `bak version` (subcommand) works.
- **Root cause**: `cmd/version.go:16-26` registers only a `version`
  **subcommand**. `cmd/root.go:27-62` never sets `rootCmd.Version`, so
  cobra does not auto-register the `--version` persistent flag. (Verified
  live: `./bak --version` → "unknown flag: --version"; `./bak version` →
  "bak dev …".)
- **Fix**: In `cmd/version.go` `init()` (or `cmd/root.go` `Execute`),
  set `rootCmd.Version = Version` and optionally a custom template via
  `rootCmd.SetVersionTemplate("bak {{.Version}}\n")`. This makes both
  `bak --version` and `bak version` work. Keep the subcommand for the
  verbose output (commit/date/runtime).
- **Effort**: small.

---

## Issue 5: `config` COMMAND MISSING (MEDIUM)

- **Symptom**: `bak config show` → `Error: unknown command "config" for "bak"`.
- **Root cause**: There is **no `cmd/config.go`** (confirmed: `cmd/` has no
  config file). Settings are only reachable via the TUI `ScreenSettings`
  (`internal/tui/model.go:211-216`, `cmd/root.go:365-383` `loadSettingsForTUI`,
  `tuiSaveSetting`). The config operations DO exist in
  `internal/actions/config_ops.go` (SaveSetting, etc.) but are not exposed
  on the CLI.
  **Worse**: `cmd/login.go:33-35` help text instructs users to run
  `bak config set providers.codeberg.token <token>` — a command that does
  **not exist**. `cmd/login_test.go:101-102` even asserts the error
  mentions `'config set'`, so a test enforces this broken reference.
  (Verified live: `./bak config show` → unknown command.)
- **Fix**: Add `cmd/config.go` with `show`, `get <key>`, `set <key> <value>`
  subcommands, reusing `config.Load()` + `internal/actions/config_ops.go`
  + `cfg.Save()`. `show` prints the redacted config JSON; `set`/`get`
  support dotted keys (`providers.codeberg.token`, `settings.default_preset`).
  Must redact tokens in `show` output per `AGENTS.md` security rules.
- **Effort**: medium (new command + dotted-key setter + redaction + tests;
  reuse existing `config_ops`).

---

## Issue 6: `restore` NEEDS ARG (MEDIUM)

- **Symptom**: `bak restore --dry-run` →
  `Error: accepts 1 arg(s), received 0`.
- **Root cause**: `cmd/restore.go:31` — `Args: cobra.ExactArgs(1)` requires
  a backup-id. There is no interactive fallback when the arg is omitted.
  (Verified live.) An interactive restore picker **exists in the TUI**
  (`internal/tui/model.go:223-228`, `ScreenRestore`, `initRestore`
  `model.go:688-716`) but the CLI `restore` command does not use it.
- **Fix**: Change `Args` to `cobra.MaximumNArgs(1)`. When no arg is given:
  - if `isTTY()`: list backups (reuse `listBackupsFrom`, `cmd/root.go:116`)
    and launch an interactive picker (reuse the TUI restore screen or a
    simple numbered prompt), then proceed;
  - if not a TTY: error with a helpful message
    (`specify a backup-id (see `bak list`) or run `bak` for interactive mode`).
  Always keep `--dry-run` working with a selected id.
- **Effort**: medium (interactive picker wiring + non-TTY error path + tests).

---

## Issue 7: `profile create` NEEDS ARG (MEDIUM)

- **Symptom**: `bak profile create` →
  `Error: accepts 1 arg(s), received 0`.
- **Root cause**: `cmd/profile.go:51` — `Args: cobra.ExactArgs(1)` requires
  a profile name even when `--interactive` is passed. The wizard exists
  (`--interactive`, `cmd/profile.go:82-84,193-244`) but cannot be launched
  without a name argument. (Verified live.)
- **Fix**: Change `Args` to `cobra.MaximumNArgs(1)`. When no name is given
  **and** `--interactive` is set: launch the wizard and let the wizard
  collect the name (the wizard already has a name step per the
  `wiring-fixes` summary: "Wizard name step added"). When no name and not
  interactive: error with a helpful message (`provide a profile name or use
  `--interactive``). When a name is given, behave as today.
- **Effort**: small (relax Args + branch on `--interactive` when name
  empty; wizard already collects name).

---

## Issue 8: FOOTER `?` HINT (LOW)

- **Symptom**: The help/shortcuts overlay exists but is not discoverable.
- **Root cause**: `internal/tui/screens/menu.go:46-50` — the main-menu
  footer help bar lists only `↑/↓ navigate • enter select • q quit`. It
  does **not** mention `?`. The overlay itself works: `internal/tui/model.go:187-190`
  toggles `showHelp` on `?`, and `model.go:569-571` renders
  `screens.RenderShortcuts` as an overlay. So the feature is wired but the
  affordance is missing.
- **Fix**: Add `{Key: "?", Desc: "help"}` (or "shortcuts") to the
  `helpKeys` slice in `RenderMainMenu` (`menu.go:46-50`). Optionally add
  the same hint to other screens' help bars for consistency.
- **Effort**: small (one HelpKey entry + test update).

---

## Issue 9: CLEANUP 76 BACKUPS (LOW)

- **Symptom**: User has 78 backups (147 MB in `~/.bak/backups/`) and wants
  to keep only the 3 newest, delete the rest.
- **Root cause**: No retention/prune logic exists. Grep for
  `cleanup|prune|retention|KeepLast|DeleteOld` finds only error-cleanup
  locals (`cleanupOnError` in `internal/backup/engine.go:142`,
  `internal/actions/backup.go:155`) and test teardowns — **no command and
  no retention policy**. Backup IDs are timestamps `YYYYMMDD-HHMMSS`
  (`internal/actions/backup.go:129`, `internal/backup/engine.go:120`), so
  lexicographic sort == chronological sort, making pruning trivial.
- **Fix**: Add a `bak cleanup` (or `bak prune`) command:
  - `--keep N` (default 3) number of newest backups to retain;
  - `--dry-run` (mandatory per `AGENTS.md`: destructive ops need dry-run);
  - scan `~/.bak/backups/` (reuse `cmd/root.go:116 listBackupsFrom`),
    sort descending by id, delete entries after index N;
  - never delete without confirmation unless `--force`;
  - optionally a `settings.retention.keep_last` config field applied
    automatically after each backup.
  Must use `filepath.Join`, redact nothing sensitive (manifests have no
  secrets — secrets are stripped at backup time), and log deletions to
  stderr.
- **Effort**: medium (new command + dry-run diff + confirmation + tests;
  list helper already exists).

---

## Cross-cutting notes

- **Two parallel backup paths exist**: `internal/backup/engine.go` (`Engine`)
  and `internal/actions/backup.go` (`BackupAction`). `cmd/backup.go` uses
  `BackupAction`; `Engine` appears unused by `cmd/`. Any exclude/scan fix
  must be applied consistently — but the real bug lives in the shared
  `internal/adapters/generic.go` `scanRootFiles`, which both ultimately
  reach via adapter `ListItems`.
- **Issue 3's `RouteSelection` headless backup is also why a "stale
  cursor" can surprise users** — worth treating as a security/UX gate.
- **`AGENTS.md` compliance for fixes**: `config show` must redact tokens;
  `cleanup` must require `--dry-run`; new `cmd/` files must delegate to
  `internal/actions/`; table-driven tests required; >80% coverage.

## Ready for Proposal

Yes. Suggest splitting into two changes given the severity mix:
- **critical-fixes** (Issues 1, 3) — ship first, small blast radius, high
  impact (broken core backup + surprising backup-on-quit).
- **cli-ux-fixes** (Issues 4, 5, 6, 7, 8, 9) — new commands/flags, medium
  effort, can be a chained PR set.
Issue 2 is already fixed in the build; carry only verification + error
polish into one of the changes.
