# Design: wiring-fixes

## Technical Approach

Wire existing, already-tested components into the production paths they were built for. No new capabilities — nine surgical wiring fixes addressing the six CRITICAL gaps + three warnings from `quality-ux-overhaul/verify-report.md`. Strategy: inject dependencies at the seams (`cmd/` and action constructors); nil-safe defaults preserve current behavior; strict TDD (RED → GREEN → REFACTOR) per `openspec/config.yaml`.

## Architecture Decisions

| # | Decision | Choice | Rejected | Rationale |
|---|----------|--------|----------|-----------|
| 1 | Exclusion injection | `ExcludesLoader func() (adapters.ScanOptions, error)` field on `Engine` + `BackupAction` (nil-safe) | Import `config` into `backup`; expand `actions.Config` with Settings | Matches existing `ProgressFn` injection pattern; keeps `backup` decoupled from `config`; zero-value = current behavior (adapter.go contract); testable via fake loader. cmd/ wiring closure calls `config.LoadExcludes` ("via a.Config" intent). |
| 2 | Restore TUI capture | `tuiRunRestore` builds `actions.RestoreAction` with `Stdout=&bytes.Buffer`, returns `buf.String()` | New RestoreAction variant returning string | Reuses existing action verbatim; mirrors `cmd/restore.go`; diff text captured from action's own output. |
| 3 | Restore confirm gate | TUI modal is the confirmation; call action with `Force=true` when `dryRun=false` | Let action prompt via Stdin | TUI already showed dry-run + modal; action's stdin prompt would block the TUI. Mandatory dry-run still honored (`dryRun=true` first). |
| 4 | Wizard launch | `tuiRunWizard` runs `wizardModel` via `tea.NewProgram` (same pattern as `runLoginInteractiveWithDeps`); maps `selectedProvider/selectedPreset` → `ProfileInfo`; cancel (`!confirmed`) → error | New wizard type | Reuses real 5-step wizard. **Name source = open question.** |
| 5 | Settings reload | Add `LoadSettings func() (screens.Settings, error)` to `tui.Deps`; `NewModel`/`screenChangeMsg` calls it, passes result to `NewSettingsModelWithSettings` | Load inside `screens` | Settings load is a cmd/config concern; keeps `screens` decoupled; nil-safe (nil → defaults). |
| 6 | Config defaults | `LoadPath` calls `applyDefaults` after unmarshal + migration: fills **zero-value** `Settings` fields from `DefaultSettings()` only | Always overwrite; defaults in every caller | Preserves existing non-zero fields (scenario "not overwritten"); single source of truth. |
| 7 | Clipboard | `cmd/login.go` imports `atotto/clipboard`, sets `DeviceClient.Clipboard: clipboard.WriteAll` | Wrap in interface | `DeviceClient.Clipboard` already injectable + nil-safe + graceful (`oauth_device.go:101-105`); promote go.mod dep. |
| 8 | Error discards | Replace `_ =` with `if err := …; err != nil { m.Msg = "…: " + err.Error() }` (lowercase, context) | Return error from Update | Bubbletea `Update` can't return errors; `m.Msg` is the existing toast channel (`profiles.go` already uses it). |
| 9 | Lint | `if/else` → `switch` (`profiles.go:193`, `restore.go:226`); `goimports -w` on 2 test files | Disable gocritic | Mechanical; satisfies All-lint-green. |

## Data Flow

Exclusion (Fix 1):

    cmd/backup.go ──ExcludesLoader(closure)──→ BackupAction.Run
                                                      │
                         config.Load + config.LoadExcludes → ScanOptions
                                                      │
                                  SetScanOptions(opts) on each ScanConfigurable adapter
                                                      │
                                                  ListItems (filtered)

Restore (Fix 2):

    TUI modal confirm → tuiRunRestore(id, dryRun)
        → actions.RestoreAction{Stdout: buf, DryRun, Force: !dryRun}
        → ResolveBackup + Run → buf.String() → restoreDryRunResultMsg

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/backup/engine.go` | Modify | Add `ExcludesLoader` field; call it + `SetScanOptions` before `ListItems` (after detect, ~L95) |
| `internal/actions/backup.go` | Modify | Add `ExcludesLoader` field; call it + `SetScanOptions` before `ListItems` (~L158) |
| `internal/actions/pick_backup.go` | Modify | Inject real `ExcludesLoader` (config.Load + LoadExcludes) into Engine (~L144) |
| `cmd/backup.go`, `cmd/root.go` | Modify | Wire `ExcludesLoader` into BackupAction (tuiRunBackup + runBackupWithDeps) |
| `cmd/root.go:218-223` | Modify | `tuiRunRestore` → real `RestoreAction` (Stdout=buf, Force=!dryRun) |
| `cmd/root.go:334-339` | Modify | `tuiRunWizard` → `tea.NewProgram(wizardModel)`; map results; cancel→error |
| `internal/tui/deps.go` | Modify | Add `LoadSettings func() (screens.Settings, error)` |
| `internal/tui/model.go:213` | Modify | Call `deps.LoadSettings`; use `NewSettingsModelWithSettings` |
| `internal/config/config.go:167` | Modify | `applyDefaults` after migration (fill zero Settings fields) |
| `internal/config/config_test.go:771` | Modify | Assert `DefaultPreset=="quick"`, `MaxFileSize==1048576`, `ConfirmDestructive==true` |
| `cmd/login.go:77-82` | Modify | Import `atotto/clipboard`; set `Clipboard: clipboard.WriteAll` |
| `go.mod` | Modify | Promote `atotto/clipboard` to direct (remove `// indirect`) |
| `internal/tui/screens/welcome.go:23-46` | Modify | Add `styles.RenderLogo(width)` + tagline + "Press Enter to get started" |
| `internal/tui/screens/profiles.go:101,111,154,193` | Modify | Handle errors → `m.Msg`; ifElseChain→switch |
| `internal/tui/screens/settings.go:110` | Modify | Handle `saveFunc` error → status msg |
| `internal/tui/screens/restore.go:226` | Modify | ifElseChain→switch |
| `internal/actions/login_test.go`, `internal/cloud/oauth_device_test.go` | Modify | `goimports -w` |

## Interfaces / Contracts

```go
// ExcludesLoader returns scan filters. Nil = current behavior (no exclusions).
// Wired in cmd/ to: config.Load() -> paths.ConfigDir("bak") -> config.LoadExcludes(dir, cfg.Settings).
type ExcludesLoader func() (adapters.ScanOptions, error)

// tui.Deps addition (nil-safe: nil -> zero-value defaults).
LoadSettings func() (screens.Settings, error)
```

`applyDefaults(s *Settings)` mutates in place: for each field, if zero-value, set from `DefaultSettings()`. Non-zero fields untouched.

## Testing Strategy

| Layer | What | How |
|-------|------|-----|
| Unit | ExcludesLoader applied before ListItems | Inject fake loader returning `node_modules/` + 1MB cap; assert adapter `ScanOpts` set, dir skipped |
| Unit | `tuiRunRestore` real action | Fake backup dir; assert buf contains "Dry-run diff:" not hardcoded string; error surfaced |
| Unit | `tuiRunWizard` launch + cancel | Inject stub `tea.NewProgram` (package var) returning `confirmed=false` → error |
| Unit | `LoadSettings` wiring | `NewModel` with fake dep → settings screen shows persisted `auto_sync=true` |
| Unit | `config.Load` defaults | Missing settings → quick/1048576/true; non-zero preserved (existing `"full"` stays) |
| Integration | `<2MB` goal | Fixture: `node_modules/` (50MB) + 5MB file → backup <2MB |
| Lint | All-lint-green | `golangci-lint run` exit 0 |

## Migration / Rollout

No migration. `git revert <merge-sha>` restores stubbed state. Defaults applied only to zero-value Settings fields on load; no schema bump.

## Open Questions

- [ ] **Wizard profile name**: `wizardModel` captures provider/preset/adapters/categories but NOT a name; spec scenario lists "name" as step 1. Options: (a) add a name step to `wizardModel` (mode `profile-create` only, ~30 lines — small logic addition), or (b) derive name from `selectedProvider` (e.g. `"github"` or `"github-quick"`). Recommend (a) to match spec; needs orchestrator confirmation since it's logic, not pure wiring.
- [ ] **Restore git auto-commit** (`RestoreAction.GitDir`): `cmd/restore.go` doesn't set it today; mirror that in `tuiRunRestore` to stay consistent. Full git-safety wiring is a separate follow-up (out of scope per proposal).
