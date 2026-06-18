# Verification Report: critical-fixes

- **Change**: `critical-fixes`
- **Branch**: `fix/critical-fixes`
- **Mode**: Full artifacts (proposal + specs + design + tasks)
- **Strict TDD**: configured in `openspec/config.yaml` (`testing.strict_tdd: true`); no Strict-TDD runner module present → standard verify (TDD red/green evidenced by task ordering + tests exist).
- **Date**: 2026-06-18 (re-verify after 3 test fixes)
- **Verifier**: sdd-verify executor
- **Re-verify commit**: `1614300` — `fix: resolve 3 test failures from critical-fixes verify`

## Verdict: PASS

The previous verify FAILED on 3 `cmd`-package test failures (C1a/C1b/C1c). All three are
resolved in commit `1614300`, with new regression tests guarding each fix. The four
verification gates all pass with fresh (non-cached) execution:

- `go test -race -count=1 ./...` → exit 0, no panics, no failures.
- `go vet ./...` → exit 0.
- `golangci-lint run` → `0 issues`, exit 0.
- All 9 fixes spot-checked (runtime + unit tests) → PASS.

Task 6.1's required test `TestProfileCreate_NoArgs_LaunchesWizard` now EXISTS and PASSES
(it was previously marked `[x]` without the test existing). Task 10.1 (`go test -race ./...`
all pass) now actually passes. The proposal success criterion "`go test ./...` passes
Ubuntu" is MET.

Remaining items are non-blocking WARNINGS (coverage below 80% for two `internal/`
packages, test-name deviations from `tasks.md`, and a `*.sqlite` vs `*.sqlite*` pattern
literal) — all carried over from the previous verify; none break the 9 fixes or the test
suite.

---

## Completeness Table

| Artifact | Present | Notes |
|---|---|---|
| proposal.md | Yes | 9 issues, success criteria, scope |
| tasks.md | Yes | 32 tasks across 10 phases, all marked `[x]` |
| design.md | Yes | 5 fixes detailed; supersedes proposal's "design skip" note |
| specs/ (5) | Yes | backup-engine, backup-retention, bak-cli, config-cli, tui-dispatch |
| verify-report.md | Yes | this file (re-verify) |

### Task Completion Audit

All 32 boxes are checked. The two previously mis-marked tasks are now legitimately
complete:

| Task | Previously | Now |
|---|---|---|
| 6.1 `TestProfileCreate_NoArgs_LaunchesWizard` | Test did NOT exist; stale `TestProfileCreate_NoArgs` FAILED | Test EXISTS and PASSES (`cmd/profile_test.go`); stale test rewritten to assert `MaximumNArgs(1)` accepts 0 args. |
| 10.1 `go test -race ./...` — all pass | FAILED in `cmd` (3 failures + panic) | PASS — `go test -race -count=1 ./...` exit 0. |

All other tasks verified implemented (source + tests): 1.1–1.6, 2.1–2.3, 3.1–3.2,
4.1–4.4, 5.1–5.3, 6.2, 7.1–7.2, 8.1–8.5, 9.1–9.2, 10.2, 10.3.

---

## Build / Test / Coverage Evidence

### Quality gates (fresh execution, cache cleared)

| Gate | Command | Result | Status |
|---|---|---|---|
| Build | `go build -o /tmp/bak-verify .` | exit 0 | PASS |
| Tests (race, fresh) | `go clean -testcache && go test -race -count=1 ./...` | `ok` for all 29 packages, exit 0, no panics | PASS |
| Vet | `go vet ./...` | exit 0 | PASS |
| Lint | `golangci-lint run` | `0 issues.`, exit 0 | PASS |

All packages PASS with `-race -count=1`: `cmd` (2.372s), `internal/actions`,
`internal/adapters` + all 10 adapter sub-packages, `internal/backup`, `internal/cloud`,
`internal/config`, `internal/crypto`, `internal/diff`, `internal/git`, `internal/manifest`,
`internal/paths`, `internal/presets`, `internal/restore`, `internal/schedule`,
`internal/tui` + `{components,screens,styles}`, `tests/e2e`.

### Previously-failing tests — now PASSING

| Test | Previous | Now | Fix |
|---|---|---|---|
| `TestExecute_VerboseFlag` | PANIC: `-v` shorthand conflict (`--version` vs `--verbose`) | PASS | `cmd/version.go` registers `--version` manually WITHOUT `-v` shorthand, pre-empting cobra's `initDefaultVersionFlag` auto-registration. Switched to `cmd.Println`/`cmd.Printf` for testability. New regression tests `TestVersionFlagNoConflictWithVerbose` + `TestVersionAndVerboseExecuteNoPanic`. |
| `TestProfileCreate_NoArgs` | FAIL (stale: asserted `ExactArgs(1)` behavior) | PASS | Rewritten to assert `MaximumNArgs(1)` accepts 0 args (wizard mode). New `TestProfileCreate_NoArgs_LaunchesWizard` (task 6.1's required test) + `TestProfileCreate_NoArgs_InteractiveAttempt`. |
| `TestRunRestore_MissingArgs` | FAIL (help-flag leakage from `TestRestoreCmd_Help`) | PASS | Explicitly resets `restoreCmd` help flag between tests (pflag v1.0.9 does not reset flag values on empty Parse). New `TestRestoreHelpFollowedByExecute` proves isolation. Refactored inline subcommand loops into `findSubcommand` helper. |

### Coverage (per `internal/` package, success criterion: ≥80%)

| Package | Coverage | Meets ≥80%? |
|---|---|---|
| internal/actions | 84.1% | Yes |
| internal/adapters | 85.8% | Yes |
| internal/adapters/codex | 83.3% | Yes |
| internal/adapters/opencode | 70.6% | **No (WARNING W4)** — `SetScanOptions` 0.0%, `scanRootFiles` 63.6% |
| internal/backup | 83.1% | Yes |
| internal/cloud | 82.5% | Yes |
| internal/config | 70.6% | **No (WARNING W1)** |
| internal/crypto | 87.5% | Yes |
| internal/diff | 96.3% | Yes |
| internal/git | 80.4% | Yes |
| internal/manifest | 86.0% | Yes |
| internal/paths | 82.1% | Yes |
| internal/presets | 81.4% | Yes |
| internal/restore | 86.1% | Yes |
| internal/schedule | 95.6% | Yes |
| internal/tui | 80.4% | Yes |
| internal/tui/components | 95.8% | Yes |
| internal/tui/screens | 80.0% | Yes |
| internal/tui/styles | 90.9% | Yes |

---

## Runtime Evidence (binary built from branch, isolated HOME + XDG_CONFIG_HOME)

| Fix | Command(s) run | Observed | Expected (spec) | Status |
|---|---|---|---|---|
| `--version` | `bak --version` | `bak dev`, exit 0 | version string, exit 0 | PASS |
| `--version` subcommand | `bak version` | `bak dev` + commit/built/runtime, exit 0 | subcommand still works | PASS |
| `--verbose --version` | `bak --verbose --version` | `bak dev`, exit 0 (no conflict) | no shorthand conflict at runtime | PASS |
| config show (no file) | `bak config show` | defaults JSON (`default_preset: quick`, `max_file_size: 1048576`), exit 0 | default settings | PASS |
| config set token | `bak config set providers.codeberg.token ghp_abcdef1234567890` | `Saved ... = ***7890`; raw `ghp_abcdef1234567890` persisted to `config.json` | raw persisted, output redacted | PASS |
| config show redacts | `bak config show` after set | `"token": "***7890"`; raw NOT in output | `***7890`, raw absent | PASS |
| config get redacts | `bak config get providers.codeberg.token` | `***7890` | redacted | PASS |
| config set nested | `bak config set settings.default_preset full` → `get` | `full` | `full` | PASS |
| config set invalid | `bak config set settings.nope xyz` | `Error: set settings.nope: unknown config key: "settings.nope"`, exit 1 | helpful error | PASS |
| restore non-TTY no-arg | `bak restore` | `Error: specify a backup-id (see 'bak list') or run 'bak' for interactive mode`, exit 1 | error references `bak list` | PASS |
| profile create no-arg | `bak profile create` | `Error: provide a profile name or use --interactive`, exit 1 | helpful error mentioning `--interactive` | PASS |
| cleanup dry-run | `bak cleanup --keep 3 --dry-run` (10 backups) | lists 7 "Would delete", keeping 3 newest | 7 listed, 0 removed | PASS |
| cleanup dry-run files | `ls ~/.bak/backups \| wc -l` after dry-run | 10 | 0 files removed | PASS |
| cleanup force | `bak cleanup --keep 3 --force` | deletes 7, `Deleted 7/7 backups (0 failed)`, exit 0; 3 newest remain (`...108`, `...109`, `...110`) | 7 deleted, 3 newest kept | PASS |
| cleanup keep>count | `bak cleanup --keep 5 --dry-run` (3 left) | `Nothing to clean (3 backups, keeping 3).`, exit 0 | nothing to delete | PASS |

Note: `profile create <name>` in an empty isolated config errors `no providers configured —
run 'bak login' or 'bak config set' first` (expected with no providers; the with-name path
is covered by `TestProfileCreate_RcloneProvider` / `TestProfileCreate_AdaptersAndCategories`,
both PASS).

---

## Spec Compliance Matrix

### backup-engine

| Requirement / Scenario | Implementation Evidence | Covering Test (runtime pass) | Status |
|---|---|---|---|
| scanRootFiles applies ScanOptions | `generic.go:296` takes `opts`; applies `MatchExclude` (326) + `MaxFileSize` (341-344); `opencode/adapter.go:222` mirrors | `TestScanRootFiles_AppliesExcludes` (3 sub-tests: sqlite, oversized, custom) PASS | COMPLIANT |
| Root SQLite excluded | `*.sqlite` in DefaultExcludes; scanRootFiles applies MatchExclude | `TestScanRootFiles_AppliesExcludes/excludes_sqlite_files` PASS | COMPLIANT |
| Root oversized excluded | `generic.go:342` warns + skips when size > MaxFileSize | `TestScanRootFiles_AppliesExcludes/skips_oversized_root_files` PASS | COMPLIANT |
| Custom excludes apply to root | `generic.go:326` applies opts.Excludes | `TestScanRootFiles_AppliesExcludes/custom_exclude_patterns_apply` PASS | COMPLIANT |
| DefaultExcludes covers runtime artifacts | `ignore.go:19-40`: `*.sqlite`, `*.sqlite-wal`, `*.sqlite-shm`, `*.db`, `*_cache.json`, `*.log` | `TestDefaultExcludes_IncludesRuntimeDBs` PASS | COMPLIANT (see W3) |
| SQLite WAL excluded by default | `*.sqlite-wal` pattern | covered by `TestDefaultExcludes_IncludesRuntimeDBs` | COMPLIANT |
| Cache JSON excluded by default | `*_cache.json` pattern | covered by `TestDefaultExcludes_IncludesRuntimeDBs` | COMPLIANT |
| User overrides replace defaults | existing `LoadExcludes` semantics (user `ExcludePatterns` replaces defaults) | `TestDefaultExcludes_IncludesRuntimeDBs` + existing ignore tests | COMPLIANT |
| Codex root config whitelist | `codex/adapter.go:27-29`: `config.toml`, `instructions.md`, `config.json`, `mcp.json`; `generic.go:36-41` `RootConfigFiles` | `TestAdapter_WhitelistOnlyConfigs` PASS | COMPLIANT |
| Whitelisted file backed up | whitelist includes `config.toml` | `TestAdapter_WhitelistOnlyConfigs` asserts `config.toml` included | COMPLIANT |
| Non-whitelisted file skipped | sqlite not in whitelist | `TestAdapter_WhitelistOnlyConfigs` asserts sqlite excluded | COMPLIANT |
| All whitelisted files present | whitelist enumerated | test covers `config.toml` + `instructions.md` | COMPLIANT |

### tui-dispatch

| Requirement / Scenario | Implementation Evidence | Covering Test | Status |
|---|---|---|---|
| RouteSelection requires explicit selection | `dispatch.go:19` `if !sel.Selected { return nil }` | `TestRouteSelection` (7 cases) PASS | COMPLIANT |
| Enter confirms selection | `model.go` handleMenuEnter sets `m.selected=true`; `deps.go:78` `Selected bool`; `model.go:645` `Selected: m.selected` | `TestRouteSelection/cursor_0_Selected=true_calls_RunBackup` PASS | COMPLIANT |
| Quit does not set Selected | q/Esc leave `selected=false` | `TestRouteSelection/cursor_0_Selected=false_does_NOT_call_RunBackup_(q_gate)` PASS | COMPLIANT |
| Esc does not set Selected | same gate | covered by Selected=false cases | COMPLIANT |
| q exits cleanly (no backup dir, code 0) | gate prevents dispatch; runtime exit 0 | `TestRouteSelection` Selected=false no-op | COMPLIANT (dispatch-level; no full TUI e2e) |
| Esc exits cleanly | gate | covered | COMPLIANT |
| Quit menu item exits cleanly | cursor-6 + Selected=true → no-op in RouteSelection | `TestRouteSelection/cursor_6_Selected=true_Quit_no-op` PASS | COMPLIANT |

### bak-cli

| Requirement / Scenario | Implementation Evidence | Covering Test / Runtime | Status |
|---|---|---|---|
| `bak --version` flag | `version.go:29-30` sets `rootCmd.Version` + template; manual `--version` flag w/o `-v` shorthand (version.go init) | Runtime `bak --version` → `bak dev` exit 0; `TestVersionFlag`, `TestVersionFlagNoConflictWithVerbose`, `TestVersionAndVerboseExecuteNoPanic` PASS | COMPLIANT |
| Flag prints version, exit 0 | runtime confirmed | `TestVersionFlag` + `TestVersionAndVerboseExecuteNoPanic` PASS | COMPLIANT |
| Subcommand still works | `version.go:16-26` keeps `versionCmd` | Runtime `bak version` works | COMPLIANT |
| `bak restore` interactive picker | `restore.go:33` `MaximumNArgs(1)`; `restore.go:54-89` picker/TTY guard | `TestRestorePicker_*` (5) PASS; `TestRunRestore_MissingArgs` + `TestRestoreHelpFollowedByExecute` PASS; runtime non-TTY error + with-arg dry-run PASS | COMPLIANT |
| TTY no-arg picker | `restore.go:60-88` launches `restorePickerModel` | model unit tests PASS; TTY e2e not run (no TTY) | COMPLIANT (model-tested) |
| Non-TTY no-arg errors (refs `bak list`) | `restore.go:55-57` | runtime confirmed | COMPLIANT |
| With arg still works | `restore.go:91-112` | runtime `restore <id> --dry-run` PASS | COMPLIANT |
| `bak profile create` wizard without name | `profile.go:51` `MaximumNArgs(1)`; `profile.go:85-87` + `255-296` wizard path | `TestProfileCreate_NoArgs` (rewritten) PASS; `TestProfileCreate_NoArgs_LaunchesWizard` PASS; `TestProfileCreate_NoArgs_InteractiveAttempt` PASS; runtime confirmed | COMPLIANT |
| Interactive no-arg launches wizard | `profile.go:85-87, 259-296` uses `wm.ProfileName()` when arg empty | `TestProfileCreate_NoArgs_InteractiveAttempt` PASS; wizard model covered by `wizard_test.go` | COMPLIANT |
| Non-interactive no-arg errors | `profile.go:90-92` "provide a profile name or use --interactive" | `TestProfileCreate_NoArgs_LaunchesWizard` PASS; runtime confirmed | COMPLIANT |
| With name arg works | `profile.go:79-80, 108-114` | `TestProfileCreate_RcloneProvider`, `TestProfileCreate_AdaptersAndCategories` PASS | COMPLIANT |
| Footer includes `? help` | `menu.go:49` `{Key:"?", Desc:"help"}` | `TestRenderMainMenu/full_menu_at_80_cols` asserts "help" present (PASS) | COMPLIANT (see W2) |
| Other hints preserved | `menu.go:47-50` keeps ↑/↓, enter, q | `TestRenderMainMenu` asserts navigate/select/quit | COMPLIANT |
| TUI selection routes only when Selected (MODIFIED) | `dispatch.go:19` gate | `TestRouteSelection` PASS | COMPLIANT |
| Quit via key press → no action, exit 0 | gate + selected=false | `TestRouteSelection` Selected=false cases | COMPLIANT |
| Selection out of bounds → zero MenuSelection | `dispatch.go:13` empty-Item guard | `TestRouteSelection/empty_selection_no-op` PASS | COMPLIANT |

### config-cli

| Requirement / Scenario | Implementation Evidence | Covering Test / Runtime | Status |
|---|---|---|---|
| `bak config show` displays settings | `cmd/config.go:46-56` → `actions.ConfigShow` | `TestConfigShow_RedactsTokens`, `TestConfigShow_NoConfig` PASS; runtime confirmed | COMPLIANT |
| Show redacts tokens (`***7890`) | `actions/redact.go` `RedactJSON` recursive walk | `TestConfigShow_RedactsTokens` PASS; runtime `***7890`, raw absent | COMPLIANT |
| Show non-secret values | redaction skips non-sensitive keys | runtime `default_preset: full` shown in full | COMPLIANT |
| Show with no config file → defaults | `config.Load()` runs `applyDefaults` | `TestConfigShow_NoConfig` PASS; runtime defaults shown | COMPLIANT |
| `bak config set` persists | `cmd/config.go:130-136` → `actions.ConfigSet` (Set + Save) | `TestConfigSet_Persists`, `TestConfigSet_InvalidKey` PASS; runtime raw persisted to file | COMPLIANT |
| Set token value (raw in file, redacted in show) | raw persisted; `***xyz` in show | runtime: file has `ghp_...`, show has `***7890` | COMPLIANT |
| Set nested key | `config.go` Get/Set extended for `settings.*` | `TestConfigSet_SettingsRoundTrip` PASS; runtime `settings.default_preset full` | COMPLIANT |
| Set invalid key → helpful error | `config.go` `unknown config key` error | `TestConfigSet_InvalidKey` PASS; runtime confirmed | COMPLIANT |
| Token redaction in output (any key ~ token/api_key/secret/password) | `redact.go` matcher lowercases + contains | `TestConfigShow_RedactsTokens` PASS | COMPLIANT |
| Short token `***ab` | `RedactString` `***`+val when len ≤ 4 | runtime `xyz` → `***xyz` | COMPLIANT |
| Multiple tokens all redacted | recursive JSON walk | `TestConfigShow_RedactsTokens` PASS | COMPLIANT |

### backup-retention

| Requirement / Scenario | Implementation Evidence | Covering Test / Runtime | Status |
|---|---|---|---|
| `bak cleanup --keep N` retains N newest | `actions/cleanup.go` sort desc + `toDelete=ids[keep:]` | `TestCleanupAction_KeepsNewest` PASS; runtime --force deletes 7 keeps 3 newest | COMPLIANT |
| Keep 3 of 10 → 3 remain, 7 deleted | runtime confirmed (3 newest `...108/109/110` remain) | `TestCleanupAction_KeepsNewest` + runtime | COMPLIANT |
| Keep more than exist → nothing, message | runtime `Nothing to clean (3 backups, keeping 3)` | `TestCleanupAction_KeepAboveCount` PASS | COMPLIANT |
| Default keep value | `cmd/cleanup.go:40` `--keep` default 3 | runtime `cleanup --force` (no --keep) uses 3 | COMPLIANT |
| Confirmation required without --dry-run | `actions/cleanup.go` Confirm gate; `cmd/cleanup.go:73` TTY prompt | `TestCleanupAction_ConfirmFalse`, `TestCleanupAction_ConfirmNilNoForce` PASS | COMPLIANT |
| TTY prompt `Delete N? [y/N]` | `cmd/cleanup.go:74-80` | unit-tested via injected Confirm | COMPLIANT (injected) |
| Non-TTY without force errors | `actions/cleanup.go` Confirm==nil + !Force → error | `TestCleanupAction_ConfirmNilNoForce` PASS; runtime confirmed (previous verify with 8 backups) | COMPLIANT |
| Force skips prompt | `cmd/cleanup.go:73` `!cleanupForce` gate | runtime `--force` deletes without prompt | COMPLIANT |
| `--dry-run` shows plan, no deletion | `actions/cleanup.go` DryRun branch | `TestCleanupAction_DryRun` PASS; runtime 7 listed, 0 removed | COMPLIANT |
| Dry-run lists deletions | runtime lists 7 | `TestCleanupAction_DryRun` PASS | COMPLIANT |
| Dry-run with no deletions | runtime `Nothing to clean` | covered | COMPLIANT |
| Dry-run then force deletes exactly those | descending sort deterministic | `TestCleanupAction_KeepsNewest` + runtime | COMPLIANT |

### OAuth error_description (proposal "Minor Fixes")

| Requirement / Scenario | Implementation Evidence | Covering Test / Runtime | Status |
|---|---|---|---|
| Surface `error_description` from API error responses | `cloud/httputil.go:55` `formatAPIError` parses JSON, prefers `error_description` (line 69-70), then `error`, then `message`, falls back to raw body / status text | Called from 13 sites (gist.go, gitea.go, github_gist.go, github_repo.go, content_types.go, oauth_device.go); exercised by `TestGist_InvalidToken`, `TestGiteaProvider_Pull_NotFound`, `TestGiteaProvider_List_NoToken`, `oauth_device_test.go` (asserts `error_description`); `internal/cloud` coverage 82.5% | COMPLIANT |

---

## Design Coherence

| Design Decision | Code Alignment | Status |
|---|---|---|
| Filter inside `scanRootFiles` (generic + opencode), mirror `scanDir` | `generic.go:296-344`, `opencode/adapter.go:222` apply MatchExclude + MaxFileSize | COHERENT |
| `RootConfigFiles` field on GenericAdapter + codex whitelist | `generic.go:36-41`, `codex/adapter.go:27-29` | COHERENT |
| `MenuSelection.Selected` + gate `RouteSelection` | `deps.go:78`, `dispatch.go:19`, `model.go:645` | COHERENT |
| Reuse `config.Config.Get/Set` for dotted keys incl. `settings.*` | `config.go` extended; `actions.ConfigShow/Get/Set` use it | COHERENT |
| Recursive JSON-map redaction | `actions/redact.go` `RedactJSON` | COHERENT |
| New `restorePickerModel` mirroring `cmd/pick.go` | `cmd/restore_picker.go`; `restore.go:70-88` | COHERENT |
| `CleanupAction` injectable FS + Confirm | `actions/cleanup.go`; `cmd/cleanup.go:62-83` | COHERENT |
| `--version` via `rootCmd.Version` + manual flag (no `-v` shorthand) | `version.go:29-30` + init() manual `Bool("version",...)` | COHERENT — **resolved**: previous deviation (C1a shorthand conflict) fixed by registering `--version` without `-v` shorthand; `TestVersionFlagNoConflictWithVerbose` asserts `vf.Shorthand == ""`. |
| `profile create` no-arg wizard via `wm.ProfileName()` | `profile.go:255-296` | COHERENT — test gap (C2) **resolved**: `TestProfileCreate_NoArgs_LaunchesWizard` added. |
| OAuth `error_description` surfacing | `cloud/httputil.go` `formatAPIError` | COHERENT — now verified (13 call sites + tests). |

---

## Issues

### CRITICAL

None. All previously-CRITICAL issues (C1a/C1b/C1c) are resolved.

### Resolved (previously CRITICAL)

- **C1a — `--version`/`-v` shorthand conflict.** RESOLVED in `cmd/version.go`: registers `--version` manually without `-v` shorthand, pre-empting cobra's `initDefaultVersionFlag` auto-registration that conflicted with `--verbose` (`-v`). Regression tests `TestVersionFlagNoConflictWithVerbose` (asserts `Shorthand == ""` + bool type) and `TestVersionAndVerboseExecuteNoPanic` (Execute with both flags) added.
- **C1b — `TestProfileCreate_NoArgs` stale + missing wizard test.** RESOLVED in `cmd/profile_test.go`: stale test rewritten to assert `MaximumNArgs(1)` accepts 0 args; task 6.1's required `TestProfileCreate_NoArgs_LaunchesWizard` added (asserts no-arg/no-`--interactive` errors with `interactive` mention) plus `TestProfileCreate_NoArgs_InteractiveAttempt`.
- **C1c — `TestRunRestore_MissingArgs` test isolation.** RESOLVED in `cmd/restore_test.go`: explicitly resets `restoreCmd` help flag between tests (pflag v1.0.9 does not reset flag values on empty Parse); `TestRestoreHelpFollowedByExecute` proves the isolation; inline subcommand loops consolidated into `findSubcommand` helper.

### WARNING

- **W1 — `internal/config` coverage = 70.6% (<80%).** AGENTS.md mandates per-package `internal/` coverage ≥80%, and the proposal success criterion requires `internal/` coverage ≥80%. This change extended `config.go` `Get`/`Set` with `settings.*` branches. The shortfall was not remediated by this change and the criterion is not met for this package. (Pre-existing; persisted from previous verify.)
- **W4 — `internal/adapters/opencode` coverage = 70.6% (<80%).** Newly surfaced in this re-verify (previous verify did not enumerate this package). Per-function: `SetScanOptions` 0.0%, `scanRootFiles` 63.6%. This change (`7a2d7c7`) modified `opencode/adapter.go` (+32 -12) to apply `ScanOptions` (MatchExclude + MaxFileSize) in `scanRootFiles`, mirroring `generic.go`. The behavior is spec-compliant via the generic `TestScanRootFiles_AppliesExcludes`, but the opencode mirror has no package-local covering test, so a divergence would not be caught. Recommend an opencode-local test mirroring the generic one.
- **W2 — Test names deviate from `tasks.md` (behavior is tested, names differ).**
  - Task 1.3 specified `TestCodexAdapter_WhitelistOnlyConfigs`; actual is `TestAdapter_WhitelistOnlyConfigs` (`internal/adapters/codex/adapter_test.go:315`) — PASSES.
  - Task 2.1 specified `TestRouteSelection_OnlyWhenSelected`; actual is `TestRouteSelection` (`internal/tui/dispatch_test.go:13`) with `Selected=true/false` cases — PASSES.
  - Task 7.1 specified `TestMainMenuFooter_ShowsHelpHint`; actual is `TestRenderMainMenu` (`internal/tui/screens/menu_test.go:16`) which asserts the word "help" is present but does not explicitly assert the `?` character. `?` IS rendered (`menu.go:49`), so the spec scenario is satisfied at runtime, but the assertion is weaker than the spec literal (`? help`).
- **W3 — `DefaultExcludes` uses `*.sqlite` (not `*.sqlite*`).** Spec (`backup-engine/spec.md`) literally lists `*.sqlite*`. Implementation (`ignore.go:24-26`) uses `*.sqlite`, `*.sqlite-wal`, `*.sqlite-shm` separately. Behaviorally equivalent for the spec scenarios (`logs_2.sqlite` matched by `*.sqlite`; `logs_2.sqlite-wal` by `*.sqlite-wal`), but a literal pattern-string deviation. Note: `*.sqlite` does NOT match a hypothetical `logs.sqlite.bak`; `*.sqlite*` would. Low risk, flagged for spec fidelity.

### SUGGESTION

- **S1** — The `cmd/` test suite relies heavily on a shared global `rootCmd` with `SetArgs`/`Execute` and no per-test reset. The `MaximumNArgs(1)` changes for `restore` and `profile create` (correct per spec) exposed latent isolation bugs (C1b, C1c), now patched with targeted flag resets. Consider a `resetRootCmd()` helper or `t.TempDir`-scoped command construction for new cmd tests to prevent recurrence.
- **S2** — Add opencode-local coverage for `SetScanOptions` + `scanRootFiles` ScanOptions application (mirrors `TestScanRootFiles_AppliesExcludes`) to lift `internal/adapters/opencode` above 80% and guard the mirror against divergence (see W4).
- **S3** — The fix commit used `NO-VERIFY: GGA flagged remaining table-driven test violations that are pre-existing patterns...` with a stated follow-up PR. Ensure that follow-up lands so GGA can return to enforcing without `--no-verify`.

---

## Final Verdict

**PASS**

The previous FAIL was driven entirely by 3 `cmd`-package test failures (C1a/C1b/C1c). All
three are resolved in commit `1614300` with new regression tests:

1. `go test -race -count=1 ./...` → exit 0, no panics, no failures (all 29 packages PASS).
2. `go vet ./...` → exit 0. `golangci-lint run` → 0 issues, exit 0.
3. Task 6.1's required `TestProfileCreate_NoArgs_LaunchesWizard` now EXISTS and PASSES.
4. All 9 fixes spot-checked at runtime and via targeted unit tests → PASS (including Fix 9
   `error_description` surfacing, now confirmed via 13 call sites + tests).

Remaining items are non-blocking WARNINGS carried over from the previous verify (W1
`internal/config` 70.6%, W2 test-name deviations, W3 `*.sqlite` pattern literal) plus a
newly-surfaced W4 (`internal/adapters/opencode` 70.6%, partly from this change's
ScanOptions mirror lacking a package-local test). None break the 9 fixes or the test suite,
and none are CRITICAL per the verification decision gates (no failing tests, no unchecked
tasks, no untested/failed spec scenarios).

Archive readiness: **unblocked**. Recommend addressing W1/W4 coverage in a follow-up to
fully meet the ≥80% per-package `internal/` criterion.
