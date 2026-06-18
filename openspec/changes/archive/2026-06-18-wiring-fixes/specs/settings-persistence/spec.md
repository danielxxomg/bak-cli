# Delta: settings-persistence

## MODIFIED Requirements

### Requirement: Settings round-trip
The system MUST load persisted settings when the TUI starts and display them in the settings screen. Settings changes MUST persist to `config.json` and survive relaunch.

(Previously: `model.go:213` used `NewSettingsModel(m.deps.SaveSetting)` with hardcoded defaults. No `LoadSettings` dependency existed. Relaunching the TUI always showed default values.)

#### Scenario: NewModel loads settings from config

- GIVEN `config.json` has `auto_sync: true` and `verbose_default: true`
- WHEN the TUI model is initialized via `NewModel`
- THEN `tui.Deps` MUST include a `LoadSettings` function
- AND the settings screen MUST be initialized with persisted values via `NewSettingsModelWithSettings`

#### Scenario: Settings changes persist to config.json

- GIVEN the user toggles `auto_sync` to `true` in the settings screen
- WHEN `saveFunc` is called
- THEN `config.json` MUST be updated with the new value
- AND subsequent `config.Load()` calls MUST return `auto_sync: true`

#### Scenario: Relaunch shows persisted values

- GIVEN `auto_sync` was saved as `true` in a previous session
- WHEN the TUI is relaunched and the settings screen is opened
- THEN the `auto_sync` toggle MUST show as enabled (not the default `false`)
