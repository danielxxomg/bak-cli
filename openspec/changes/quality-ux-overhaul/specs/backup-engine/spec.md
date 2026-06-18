# Delta for backup-engine

## ADDED Requirements

### Requirement: Progress callback on engine

The system MUST accept an optional `progressFn func(currentFile string, filesDone int, filesTotal int)` field on `Engine`. The callback SHALL be invoked once per file during backup. A nil `progressFn` MUST be safe (no-op).

#### Scenario: Callback invoked per file

- GIVEN an engine with 10 files to back up and a non-nil `progressFn`
- WHEN `Engine.Run()` executes
- THEN `progressFn` SHALL be called 10 times with incrementing `filesDone` and `filesTotal=10`

#### Scenario: Nil callback safe

- GIVEN an engine with `progressFn` set to nil
- WHEN `Engine.Run()` executes
- THEN the backup SHALL complete without panic

### Requirement: Scan options with exclusions

The system MUST accept `ScanOptions{Excludes []string, MaxFileSize int64}` in `generic.go:scanDir` and the OpenCode adapter. Files matching any exclude pattern or exceeding `MaxFileSize` SHALL be skipped.

#### Scenario: Exclude pattern applied

- GIVEN `ScanOptions{Excludes: ["node_modules"]}` and a directory containing `node_modules/`
- WHEN `scanDir` executes
- THEN `node_modules/` entries SHALL NOT appear in the returned items

#### Scenario: MaxFileSize applied

- GIVEN `ScanOptions{MaxFileSize: 1048576}` and a 2 MB file
- WHEN `scanDir` executes
- THEN the 2 MB file SHALL NOT appear in the returned items

#### Scenario: Zero-value ScanOptions

- GIVEN `ScanOptions{}` (zero value, no excludes, no size limit)
- WHEN `scanDir` executes
- THEN behavior SHALL be identical to the current implementation (all files included)

## MODIFIED Requirements

### Requirement: Preset-based backup

The system MUST support `quick`, `full`, and `skills` presets. During backup, the system MUST apply exclusion rules from `ScanOptions` (default patterns + custom ignore + max file size) before copying files.

(Previously: presets copied all discovered files with zero exclusion filtering)

#### Scenario: Quick preset

- GIVEN OpenCode config exists
- WHEN `bak backup` runs
- THEN manifest created with essential configs only

#### Scenario: Full preset

- GIVEN multiple agent configs exist
- WHEN `bak backup --preset full` runs
- THEN all discoverable configs backed up, excluding files matching exclusion rules

#### Scenario: Full preset with large plugin

- GIVEN a full preset backup and a plugin directory with a 5 MB binary
- WHEN `bak backup --preset full` runs with default `MaxFileSize` of 1 MB
- THEN the 5 MB binary SHALL be excluded from the backup
