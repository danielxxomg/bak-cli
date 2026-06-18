# Tasks: Critical Fixes

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~500 |
| 400-line budget risk | Low |
| Chained PRs recommended | No |
| Suggested split | Single PR |
| Delivery strategy | single-pr |
| Chain strategy | pending |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: pending
400-line budget risk: Low

## Phase 1: Fix Backup Size (CRITICAL)

- [x] 1.1 **RED** — `internal/adapters/generic_test.go`: TestScanRootFiles_AppliesExcludes — verify ScanOptions filters (MatchExclude + MaxFileSize) applied in scanRootFiles
- [x] 1.2 **GREEN** — `internal/adapters/generic.go`: scanRootFiles takes `opts ScanOptions` param; apply MatchExclude + MaxFileSize per entry (mirror scanDir:194-217)
- [x] 1.3 **RED** — `internal/adapters/codex/adapter_test.go`: TestCodexAdapter_WhitelistOnlyConfigs — assert only config files returned, not SQLite DBs
- [x] 1.4 **GREEN** — `internal/adapters/codex/adapter.go`: add RootConfigFiles whitelist (`config.toml`, `instructions.md`, `config.json`, `mcp.json`); set `base.RootConfigFiles`
- [x] 1.5 **RED** — `internal/config/ignore_test.go`: TestDefaultExcludes_IncludesRuntimeDBs — assert `*.sqlite*`, `*.db`, `*_cache.json`, `*.log`, `.git/`, `node_modules/` present
- [x] 1.6 **GREEN** — `internal/config/ignore.go`: expand DefaultExcludes with `*.sqlite*`, `*.sqlite-wal`, `*.sqlite-shm`, `*.db`, `*_cache.json`, `*.jsonl`

## Phase 2: Fix TUI 'q' Dispatch Gate (HIGH)

- [x] 2.1 **RED** — `internal/tui/dispatch_test.go`: TestRouteSelection_OnlyWhenSelected — fires on Enter (Selected=true), NOT on q/Esc (Selected=false)
- [x] 2.2 **GREEN** — `internal/tui/dispatch.go`: add `Selected bool` to MenuSelection; guard `if !sel.Selected { return nil }` after empty-Item check
- [x] 2.3 **GREEN** — `internal/tui/model.go`: set `m.selected = true` in handleMenuEnter; leave false on q/Esc/quit; Selection() populates Selected field

## Phase 3: Add --version Flag (MEDIUM)

- [x] 3.1 **RED** — `cmd/root_test.go`: TestVersionFlag — `bak --version` prints version string
- [x] 3.2 **GREEN** — `cmd/root.go`: set `rootCmd.Version = Version` + `rootCmd.SetVersionTemplate("bak {{.Version}}\n")`

## Phase 4: Add Config Command (MEDIUM)

- [x] 4.1 **RED** — `internal/actions/config_command_test.go`: TestConfigShow_RedactsTokens (token → `***xxxx`), TestConfigSet_Persists (round-trip)
- [x] 4.2 **GREEN** — `internal/actions/config_command.go`: ConfigShowAction (marshal + redact), ConfigSetAction (cfg.Set + cfg.Save)
- [x] 4.3 **RED** — `cmd/config_test.go`: TestConfigShow_DisplaysSettings, TestConfigSet_Persists — cobra.Execute with buffer asserts
- [x] 4.4 **GREEN** — `cmd/config.go`: `config show`, `config get <key>`, `config set <key> <value>` commands; thin RunE → actions

## Phase 5: Fix Restore Interactive (MEDIUM)

- [x] 5.1 **RED** — `cmd/restore_picker_test.go`: TestRestorePicker_SelectsBackup (Enter sets ID), TestRestorePicker_Empty (empty list handled)
- [x] 5.2 **GREEN** — `cmd/restore_picker.go`: restorePickerModel mirroring cmd/pick.go pattern (Init/Update/View, SelectedID, WindowSizeMsg, terminal-too-small guard)
- [x] 5.3 **GREEN** — `cmd/restore.go`: change Args to MaximumNArgs(1); no-arg TTY → launch picker; non-TTY → error with `bak list` hint

## Phase 6: Fix Profile Create (MEDIUM)

- [x] 6.1 **RED** — `cmd/profile_test.go`: TestProfileCreate_NoArgs_LaunchesWizard — wizard collects name when no arg + --interactive
- [x] 6.2 **GREEN** — `cmd/profile.go`: change Args to MaximumNArgs(1); no name + --interactive → launch wizard; no name + no flag → error

## Phase 7: Add Footer Hint (LOW)

- [x] 7.1 **RED** — `internal/tui/screens/menu_test.go`: TestMainMenuFooter_ShowsHelpHint — assert `? help` present in footer
- [x] 7.2 **GREEN** — `internal/tui/screens/menu.go`: append `{Key: "?", Desc: "help"}` to helpKeys slice

## Phase 8: Add Cleanup Command (LOW)

- [x] 8.1 **RED** — `internal/actions/cleanup_test.go`: TestCleanupAction_KeepsNewest (keep 3 of 10 → 7 deleted), TestCleanupAction_DryRun (0 deletions)
- [x] 8.2 **GREEN** — `internal/actions/cleanup.go`: CleanupAction with keep-N logic, DryRun/Force/Confirm injection, descending sort, summary output
- [x] 8.3 **RED** — `cmd/cleanup_test.go`: TestCleanup_KeepN, TestCleanup_DryRun — cobra.Execute with temp backups dir
- [x] 8.4 **GREEN** — `cmd/cleanup.go`: `bak cleanup --keep N --dry-run [--force]` command; thin RunE → CleanupAction
- [x] 8.5 **GREEN** — `cmd/root.go`: register cleanupCmd (via cleanup.go init())

## Phase 9: Cleanup Old Backups (Manual)

- [x] 9.1 Run `./bak cleanup --keep 3 --dry-run` — verify plan output
- [x] 9.2 Run `./bak cleanup --keep 3` — delete with confirmation

## Phase 10: Quality Gates

- [x] 10.1 `go test -race ./...` — all pass
- [x] 10.2 `go vet ./...` — clean
- [x] 10.3 `golangci-lint run` — exit 0
