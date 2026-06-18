# Proposal: wiring-fixes

## Intent

`quality-ux-overhaul` shipped 50/50 tasks checked, but its verify flagged **6 CRITICAL wiring gaps** + **3 warnings**. Tests pass because mocks bypass production glue. This change wires existing components so end-to-end spec scenarios hold in production and `golangci-lint` exits 0 — unblocking archive.

## Scope

### In Scope
1. **Exclusion engine** — `Engine.Run` + `BackupAction.Run` call `LoadExcludes` + `SetScanOptions`.
2. **Real restore** — `tuiRunRestore` stub → `actions.RestoreAction`.
3. **Real wizard** — `tuiRunWizard` stub → `wizardModel`.
4. **Settings reload** — `model.go:213` → `NewSettingsModelWithSettings`; add `LoadSettings`.
5. **Config defaults** — `Load` applies `DefaultSettings()` when missing; fix test.
6. **Lint** — ifElseChain → switch; goimports on 2 test files.
7. **OAuth clipboard** — inject `atotto/clipboard`; promote dep.
8. **Welcome content** — logo + tagline + "Press Enter to get started".
9. **Error discards** — handle returns (`profiles.go:101,111,154`, `settings.go:110`).

### Out of Scope
Restore TUI progress bridge, `package-lock.json` pattern, cloud `p`/`l`, list sort — follow-up.

## Capabilities

### New Capabilities
None.

### Modified Capabilities
`backup-engine`, `backup-exclude-rules` — `LoadExcludes`+`SetScanOptions` before `ListItems`. `bak-cli` — `Load` applies `DefaultSettings`; `tui.Deps.LoadSettings`; real `RestoreAction`/`wizardModel`. `tui-restore-screen`, `tui-profiles-screen`, `tui-welcome-screen`, `tui` — wiring fixes. `oauth-device-flow` — clipboard injected. `ci-consistency` — re-assert "All-lint-green".

## Approach

One branch, one PR (~300 lines, within 400-line budget). Strict TDD: RED test → GREEN wiring → REFACTOR. ~3h.

## Affected Areas

| Area | Change |
|------|--------|
| `internal/backup/engine.go:56-150`, `internal/actions/backup.go:80-170` | `LoadExcludes`+`SetScanOptions` before `ListItems` |
| `internal/config/config.go:123-168`, `config_test.go:771` | `Load` applies `DefaultSettings()`; test asserts `"quick"` |
| `cmd/root.go:218-223, 334-339` | Real `RestoreAction` + real `wizardModel` |
| `cmd/login.go:77-82`, `login_test.go:61`, `internal/cloud/oauth_device_test.go:136` | Inject clipboard; goimports |
| `internal/tui/model.go:213` | `NewSettingsModelWithSettings`; `LoadSettings` dep |
| `internal/tui/screens/welcome.go:23-46` | Logo + tagline |
| `internal/tui/screens/{settings.go:110, profiles.go:101,111,154,193, restore.go:226}` | Handle errors; switch on state |
| `go.mod` | Promote `atotto/clipboard` |

## Risks

Low: `Deps.RunRestore` signature already matches; defaults applied only when zero + missing file; `atotto/clipboard` no-op without display (nil-safe caller); goimports scoped to 2 flagged files.

## Rollback Plan

Single PR. `git revert <merge-sha>` reverts to stubbed state. No data/schema migration.

## Dependencies

`atotto/clipboard` v0.1.4 already in `go.sum` — promote only.

## Success Criteria

- [ ] Engine integration: `node_modules/` + 5 MB file → backup < 2 MB.
- [ ] `tuiRunRestore` returns real action; `tuiRunWizard` returns user-selected profile.
- [ ] `tui.Deps.LoadSettings` populated; relaunch shows saved `auto_sync`.
- [ ] `config.Load` no-file → `DefaultPreset="quick"`, `MaxFileSize=1048576`, `ConfirmDestructive=true`.
- [ ] `golangci-lint run` exits 0; `go test -race` exits 0; coverage ≥80% on `internal/`.
- [ ] `bak login` with `BAK_GITHUB_OAUTH_CLIENT_ID` copies user code to clipboard.
- [ ] Welcome shows logo + tagline + "Press Enter to get started".
- [ ] No `_ =` on error returns in `internal/tui/screens/profiles.go` / `settings.go`.
