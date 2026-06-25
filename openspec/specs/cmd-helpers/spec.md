# cmd-helpers Specification

## Purpose

Consolidate duplicated helper functions across `internal/actions/`, `internal/backup/`, and `cmd/` into single shared locations, eliminating code duplication and ensuring consistent behavior.

## Requirements

### Requirement: resolveHome shared helper

`resolveHome` (expanding `~/` to user home dir) MUST exist in exactly one location: `internal/actions/helpers.go`. All call sites MUST use this shared implementation.

#### Scenario: single resolveHome implementation

- GIVEN the codebase after refactoring
- WHEN `grep -r "func resolveHome" --include="*.go"` is executed
- THEN exactly one match is found in `internal/actions/helpers.go`

#### Scenario: all call sites use shared resolveHome

- GIVEN `internal/actions/helpers.go` exports `ResolveHome`
- WHEN any package needs home dir expansion
- THEN it imports and calls `actions.ResolveHome`

### Requirement: normalizePath shared helper

`normalizePath` (canonical path normalization using `path.Clean` + `strings.ReplaceAll`) MUST exist in exactly one location: `internal/actions/helpers.go`. All call sites MUST use this shared implementation.

#### Scenario: single normalizePath implementation

- GIVEN the codebase after refactoring
- WHEN `grep -r "func normalizePath\|func NormalizePath" --include="*.go"` is executed
- THEN exactly one match is found in `internal/actions/helpers.go`

#### Scenario: normalizePath uses cross-platform canonical form

- GIVEN `actions.NormalizePath` is called with a Windows-style path
- WHEN the function executes
- THEN it returns `path.Clean(strings.ReplaceAll(p, "\\", "/"))`

### Requirement: loadConfigOr shared helper

`loadConfigOr` (loading config with fallback default) MUST exist in exactly one location: `internal/actions/helpers.go`. All call sites MUST use this shared implementation.

#### Scenario: single loadConfigOr implementation

- GIVEN the codebase after refactoring
- WHEN `grep -r "func loadConfigOr\|func LoadConfigOr" --include="*.go"` is executed
- THEN exactly one match is found in `internal/actions/helpers.go`

#### Scenario: loadConfigOr propagates errors

- GIVEN `actions.LoadConfigOr` is called with an invalid config path
- WHEN the config loader returns an error
- THEN `LoadConfigOr` returns an error

### Requirement: resolveRoot shared helper

`resolveRoot` (determining the backup root directory) MUST exist in exactly one location: `internal/actions/helpers.go`. All call sites MUST use this shared implementation.

#### Scenario: single resolveRoot implementation

- GIVEN the codebase after refactoring
- WHEN `grep -r "func resolveRoot\|func ResolveRoot" --include="*.go"` is executed
- THEN exactly one match is found in `internal/actions/helpers.go`

### Requirement: backup_helpers.go duplication eliminated

The file `internal/backup/backup_helpers.go` MUST NOT contain functions that duplicate logic in `internal/actions/helpers.go`. Any shared helpers MUST be removed from `backup_helpers.go` and callers MUST import from `internal/actions/`.

#### Scenario: no duplicate helpers in backup_helpers.go

- GIVEN the codebase after refactoring
- WHEN `internal/backup/backup_helpers.go` is inspected
- THEN it does not contain `resolveHome`, `normalizePath`, `loadConfigOr`, or `resolveRoot`

#### Scenario: backup package imports from actions

- GIVEN `internal/backup/` needs home dir expansion
- WHEN the code is compiled
- THEN it imports `internal/actions` and calls `actions.ResolveHome`

### Requirement: cmd/ uses shared helpers

The `cmd/` package MUST NOT contain local copies of `resolveHome`, `normalizePath`, `loadConfigOr`, or `resolveRoot`. All such functions MUST be imported from `internal/actions/`.

#### Scenario: cmd/ has no local helper copies

- GIVEN the codebase after refactoring
- WHEN `cmd/` Go files are inspected
- THEN no local `resolveHome`, `normalizePath`, `loadConfigOr`, or `resolveRoot` functions exist

#### Scenario: cmd/ imports from actions

- GIVEN `cmd/` needs path normalization
- WHEN the code is compiled
- THEN it imports `internal/actions` and calls `actions.NormalizePath`
