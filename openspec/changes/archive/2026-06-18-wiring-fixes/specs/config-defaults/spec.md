# Delta: config-defaults

## MODIFIED Requirements

### Requirement: Default settings
The system MUST apply `DefaultSettings()` when the settings section is missing from `config.json` or when no config file exists. Existing config fields MUST NOT be lost.

(Previously: `Load()` and `LoadPath()` returned zero-value `Settings` when the file was missing or the `settings` key was absent. `DefaultSettings()` existed but was never called.)

#### Scenario: Load applies defaults when settings missing

- GIVEN `config.json` exists with `github_token` but no `settings` key
- WHEN `config.Load()` is called
- THEN `cfg.Settings.DefaultPreset` MUST be `"quick"`
- AND `cfg.Settings.MaxFileSize` MUST be `1048576`
- AND `cfg.Settings.ConfirmDestructive` MUST be `true`

#### Scenario: DefaultPreset is "quick"

- GIVEN no config file exists
- WHEN `config.Load()` is called
- THEN `cfg.Settings.DefaultPreset` MUST be `"quick"`

#### Scenario: Existing config preserves other fields

- GIVEN `config.json` has `providers`, `profiles`, and `active_profile` set
- WHEN `config.Load()` applies `DefaultSettings()`
- THEN `cfg.Providers`, `cfg.Profiles`, and `cfg.ActiveProfile` MUST retain their original values
- AND only the zero-value `Settings` fields are populated from defaults

#### Scenario: Existing non-zero settings are not overwritten

- GIVEN `config.json` has `settings.default_preset: "full"`
- WHEN `config.Load()` is called
- THEN `cfg.Settings.DefaultPreset` MUST remain `"full"` (not overwritten to `"quick"`)
