# Tasks: wiring-fixes

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~300 |
| 800-line budget risk | Low |
| Chained PRs recommended | No |
| Suggested split | Single PR |
| Delivery strategy | ask-on-risk |
| Chain strategy | pending |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: pending
800-line budget risk: Low

## Phase 1: Config defaults

- [x] 1.1 **RED**: Write `internal/config/config_test.go` — `TestLoad_AppliesDefaultsWhenMissing`: Load() with no settings section returns DefaultSettings values (preset="quick", auto_sync=false, max_file_size=1048576, confirm_destructive=true)
- [x] 1.2 **GREEN**: Modify `internal/config/config.go` — Load()/LoadPath() call DefaultSettings() when settings section missing, merge without losing existing fields
- [x] 1.3 **REFACTOR**: Fix test `TestSettings_LoadDefaultsWhenMissing` — assert DefaultPreset=="quick" not ""

## Phase 2: Settings persistence wiring

- [x] 2.1 **RED**: Write `internal/tui/model_test.go` — `TestNewModel_LoadsSettings`: NewModel with settings param uses loaded values, not defaults
- [x] 2.2 **GREEN**: Modify `internal/tui/model.go` — NewModel accepts settings parameter, passes to NewSettingsModelWithSettings
- [x] 2.3 **GREEN**: Modify `cmd/root.go` — load config.Settings before TUI start, pass to NewModel

## Phase 3: Exclusion engine wiring

- [x] 3.1 **RED**: Write `internal/backup/engine_test.go` — `TestEngine_Run_AppliesExcludes`: engine calls ExcludesLoader, passes to adapter.SetScanOptions
- [x] 3.2 **GREEN**: Modify `internal/backup/engine.go` — add ExcludesLoader field (nil-safe), call in Run before scanning
- [x] 3.3 **RED**: Write `internal/actions/backup_test.go` — `TestBackupAction_Run_AppliesExcludes`: action calls ExcludesLoader
- [x] 3.4 **GREEN**: Modify `internal/actions/backup.go` — add ExcludesLoader field, call before engine.Run
- [x] 3.5 **GREEN**: Modify `cmd/root.go` — wire config.LoadExcludes as ExcludesLoader in BackupAction

## Phase 4: Real restore wiring

- [x] 4.1 **RED**: Write `cmd/root_test.go` — `TestTuiRunRestore_RealAction`: calls RestoreAction, returns diff on dry-run, restores on confirm
- [x] 4.2 **GREEN**: Modify `cmd/root.go` — replace tuiRunRestore stub with real RestoreAction call, buffer capture for diff output

## Phase 5: Real wizard wiring

- [x] 5.1 **RED**: Write `cmd/root_test.go` — `TestTuiRunWizard_RealWizard`: launches wizardModel, returns ProfileInfo
- [x] 5.2 **GREEN**: Modify `cmd/root.go` — replace tuiRunWizard stub with real tea.NewProgram(wizardModel)

## Phase 6: Lint fixes

- [x] 6.1 Modify `internal/tui/screens/profiles.go` — ifElseChain → switch
- [x] 6.2 Modify `internal/tui/screens/restore.go` — ifElseChain → switch
- [x] 6.3 Run goimports on modified test files

## Phase 7: OAuth clipboard wiring

- [x] 7.1 **RED**: Write `cmd/login_test.go` — `TestLogin_OAuthClipboard`: user code auto-copied, graceful fallback
- [x] 7.2 **GREEN**: Modify `cmd/login.go` — import atotto/clipboard, inject copy function into DeviceClient

## Phase 8: Welcome content

- [x] 8.1 **RED**: Write `internal/tui/screens/welcome_test.go` — `TestWelcomeView_HasLogoAndTagline`: ASCII logo + tagline present
- [x] 8.2 **GREEN**: Modify `internal/tui/screens/welcome.go` — add ASCII logo from styles, tagline "Pack your AI coding setup. Move anywhere."

## Phase 9: Error handling

- [x] 9.1 Modify `internal/tui/screens/profiles.go` — replace `_ =` with proper error handling, error messages lowercase with context
- [x] 9.2 Modify `internal/tui/screens/settings.go` — replace `_ =` with proper error handling

## Phase 10: Quality gates

- [x] 10.1 `go test -race ./...` — all pass
- [x] 10.2 `go vet ./...` — clean
- [x] 10.3 `golangci-lint run` — exit 0
