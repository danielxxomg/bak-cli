# Proposal: critical-fixes

## Intent

Fix 9 issues from manual testing (build `2026-06-18 13:59`):

- **Backup bloat (8.4 MB quick)**: `scanRootFiles` in `internal/adapters/generic.go:286-325` ignores `ScanOptions`; codex has no root-config whitelist → `~/.codex/*.sqlite*` runtime DBs leak.
- **TUI `q` runs backup**: `RouteSelection` (`internal/tui/dispatch.go:12-24`) fires `RunBackup` whenever `Cursor==0` on exit (q/Esc/Quit).
- **Missing CLI affordances**: `--version`, `config show/set`, `restore` (no-arg picker), `profile create` (wizard without name), footer `?` hint, `cleanup` retention.

OAuth 400 (Issue 2) is already fixed in `45a03b4`; this carries verification + `error_description` surfacing only.

## Scope

### In Scope
- `scanRootFiles` honors `ScanOptions` in `generic.go` and `opencode/adapter.go:221-264`.
- Codex `rootConfigFiles` whitelist (`config.toml`, `instructions.md`, `config.json`, `mcp.json`).
- Expand `DefaultExcludes` in `internal/config/ignore.go:19-34` with `*.sqlite*`, `*.db`, `*_cache.json`, `*.jsonl`.
- `MenuSelection.Selected bool`; gate `RouteSelection` behind it.
- Set `rootCmd.Version` in `cmd/version.go`; keep `version` subcommand.
- New `cmd/config.go`: `show` / `get <key>` / `set <key> <value>`, redacting tokens.
- `bak restore` no-arg → interactive picker (TTY) / helpful error (non-TTY).
- `bak profile create --interactive` no-arg → wizard collects name.
- Add `? help` to `internal/tui/screens/menu.go:46-50`.
- New `bak cleanup --keep N --dry-run [--force]` reusing `listBackupsFrom` (`cmd/root.go:116`).
- Surface `error_description` from OAuth 400 in `internal/cloud/httputil.go:53-58`.

### Out of Scope
- Auto-apply retention after each backup (no `settings.retention` field yet).
- Splitting into critical-fixes + cli-ux-fixes (kept as one change per user request).

## Capabilities

### New Capabilities
- `config-cli`: `bak config show|get|set` with dotted keys + token redaction.
- `backup-retention`: `bak cleanup --keep N --dry-run` with confirmation gate.

### Modified Capabilities
- `backup-engine`: `scanRootFiles` applies `ScanOptions`; `DefaultExcludes` covers `*.sqlite*`, `*.db`, `*_cache.json`, `*.jsonl`.
- `generic-adapter`: codex whitelists root config; root scan respects `MaxFileSize`.
- `bak-cli`: `--version` works; TUI dispatch gated by `Selected`; `restore` accepts 0 or 1 arg; OAuth 400 surfaces `error_description`; `?` hint in main-menu footer.
- `wizard-flow`: `profile create` no-arg + `--interactive` launches the wizard, which collects the name.

## Approach

Per-issue, additive, independent commits. New `cmd/` logic delegates to `internal/actions/` (AGENTS.md). Destructive ops have `--dry-run`. Tests table-driven with `setConfigHome` isolation. No new third-party deps.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/adapters/generic.go:286-325` | Modified | `scanRootFiles` honors `ScanOptions` |
| `internal/adapters/codex/adapter.go:17-20` | Modified | Add `rootConfigFiles` whitelist |
| `internal/adapters/opencode/adapter.go:221-264` | Modified | Root scan respects `MaxFileSize` |
| `internal/config/ignore.go:19-34` | Modified | Expand `DefaultExcludes` |
| `internal/tui/dispatch.go:12-24` | Modified | Gate behind `Selected` |
| `internal/tui/model.go:627-642` | Modified | `MenuSelection.Selected` field |
| `internal/tui/screens/menu.go:46-50` | Modified | Add `?` hint |
| `cmd/version.go` | Modified | Set `rootCmd.Version` |
| `cmd/login.go` | Modified | Surface `error_description` |
| `cmd/config.go` | New | `show`/`get`/`set` |
| `cmd/restore.go:31` | Modified | `MaximumNArgs(1)` + picker |
| `cmd/profile.go:51` | Modified | `MaximumNArgs(1)` + wizard |
| `cmd/cleanup.go` | New | Retention command |
| `internal/actions/cleanup.go` | New | Testable action |
| `internal/cloud/httputil.go:53-58` | Modified | Parse `error_description` |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| New excludes drop files users want | Med | `ExcludePatterns` user overrides still win; integration test asserts override semantics |
| `RouteSelection` gate breaks headless backup | Low | No current headless caller needs unguarded dispatch; add unit test for cursor-only exits |
| `bak cleanup` deletes backups accidentally | Med | `--dry-run` default ON, requires `--force`, prints plan, TTY confirmation |
| `config set` redaction misses a key | Low | Centralize helper; test token, api_key, password keys |
| `restore` no-arg blocks scripts | Low | Non-TTY errors with `bak list` hint; TTY path is opt-in |

## Rollback Plan

All fixes additive and isolated — revert per-commit, no schema migrations, no config-format changes. `bak cleanup` is brand new (nothing to roll back). Reverting `DefaultExcludes` keeps existing backups intact; only future backups change.

## SDD Phase Decision

| Phase | Executed | Rationale |
|-------|----------|-----------|
| sdd-explore | ✅ | Root cause analysis for 9 issues with file:line evidence |
| sdd-propose | ✅ | Defined scope, 9 fixes, success criteria |
| sdd-spec | ✅ | 5 domains, 16 requirements, 41 scenarios |
| sdd-design | ❌ Skip | Fixes are additive/wiring — no architecture decisions needed |
| sdd-tasks | ✅ | Plan per-fix tasks with TDD ordering |
| sdd-apply | ✅ | Implement with strict TDD |
| sdd-verify | ✅ | Verify all fixes work end-to-end |
| sdd-archive | ✅ | Close cycle |

Design skipped: all 9 fixes are additive (new commands, wiring changes, expanded defaults) with no architectural tradeoffs. The explore phase already identified exact file:line root causes, making design redundant.

## Dependencies

None. Stdlib + cobra + bubbletea already in tree.

## Success Criteria

- [ ] `bak backup --preset quick` ≤ 1 MB on test env (down from 8.4 MB); `*.sqlite*` excluded.
- [ ] `codex/config.toml` backed up; `codex/logs_2.sqlite` NOT in manifest.
- [ ] TUI `q` exits code 0, creates no backup directory.
- [ ] `bak --version` prints version; `bak version` still works.
- [ ] `bak config show` redacts `providers.*.token` as `***`.
- [ ] `bak config set providers.codeberg.token xyz` persists; `bak config get` returns `xyz`; `show` still redacts.
- [ ] `bak restore` (no arg, TTY) lists backups and lets user pick; `bak restore <id> --dry-run` still works.
- [ ] `bak profile create --interactive` (no name) launches wizard; wizard saves profile with chosen name.
- [ ] Main-menu footer includes `? help`.
- [ ] `bak cleanup --keep 3 --dry-run` lists 75 deletions, 0 files removed; `--force` deletes them.
- [ ] OAuth 400 surfaces `error_description` (not raw body).
- [ ] `go test ./...` passes Ubuntu; `internal/` coverage ≥ 80%; `golangci-lint run` exits 0.
