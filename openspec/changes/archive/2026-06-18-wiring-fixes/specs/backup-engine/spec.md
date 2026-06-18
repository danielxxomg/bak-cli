# Delta: backup-engine

## MODIFIED Requirements

### Requirement: Preset-based backup
The system MUST support `quick`, `full`, and `skills` presets. Before scanning files, the system MUST load exclusion rules and propagate scan options to all adapters.

(Previously: Presets resolved categories but exclusion rules were never loaded or applied during scan.)

#### Scenario: Exclusion engine wired in Engine.Run

- GIVEN `Engine.Run()` is called with any preset
- WHEN the engine resolves categories and detects adapters
- THEN `config.LoadExcludes` MUST be called with the config directory and settings
- AND `SetScanOptions` MUST be called on every detected adapter before `ListItems`

#### Scenario: Exclusion engine wired in BackupAction.Run

- GIVEN `BackupAction.Run()` is called
- WHEN the action resolves categories and detects adapters
- THEN `config.LoadExcludes` MUST be called via `a.Config`
- AND `SetScanOptions` MUST be called on every detected adapter before `ListItems`

#### Scenario: .bakignore patterns applied during scanDir

- GIVEN a `.bakignore` file exists with pattern `node_modules/`
- WHEN `scanDir` encounters a `node_modules/` directory
- THEN the directory MUST be skipped
- AND files inside it MUST NOT appear in the backup manifest

#### Scenario: MaxFileSize default cap applied

- GIVEN `config.Load()` returns `DefaultSettings()` with `MaxFileSize=1048576`
- WHEN an adapter scans a file larger than 1 MB
- THEN the file MUST be skipped with a stderr warning

#### Scenario: Custom exclude patterns override defaults

- GIVEN `Settings.ExcludePatterns` is `["*.tmp", "*.cache"]`
- WHEN `LoadExcludes` is called
- THEN only `*.tmp` and `*.cache` patterns are used (defaults replaced)
- AND these patterns are applied during `scanDir`
