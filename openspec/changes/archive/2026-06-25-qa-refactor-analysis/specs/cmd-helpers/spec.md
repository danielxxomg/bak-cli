# cmd-helpers Specification

## Purpose

Consolidate duplicated helper functions across `internal/actions/`, `internal/backup/`, and `cmd/` into single canonical implementations in the appropriate leaf package.

## Requirements

### Requirement: resolveBackupID canonical

Three forms of backup-ID resolution (`push.go:201`, `pick_backup.go:33`, `cleanup.go:60`) MUST be consolidated into `backup.LatestBackupID(backupsDir) (string, error)` and `backup.ListBackupIDs(backupsDir) ([]string, error)` in `internal/backup/`. All call sites MUST use these canonical functions. `backup.ResolveBackupID` (different signature, returns backupDir) stays separate.

#### Scenario: resolves by latest (descending sort)

- GIVEN a backups directory with 3 backups sorted by timestamp
- WHEN `LatestBackupID(backupsDir)` is called
- THEN it returns the most recent backup ID

#### Scenario: resolves by index

- GIVEN `ListBackupIDs(backupsDir)` returns `["a", "b", "c"]`
- WHEN a call site accesses index 0
- THEN it gets the first backup ID from the sorted list

#### Scenario: resolves by name (explicit ID)

- GIVEN a valid backup ID passed directly
- WHEN `backup.ResolveBackupID(id)` is called
- THEN it returns the corresponding backup directory path

#### Scenario: not-found returns error

- GIVEN an empty backups directory
- WHEN `LatestBackupID(backupsDir)` is called
- THEN it returns a non-nil error indicating no backups found

#### Scenario: all call sites use canonical function

- GIVEN the refactored codebase
- WHEN grepping for inline backup-ID resolution logic in `actions/` and `cmd/`
- THEN zero inline implementations remain
- AND all sites import from `internal/backup`

### Requirement: hostname helper consolidated

Three sites of hostname resolution (`backup.go:137-147`, `push.go:119-131`, `engine.go:127-133`) MUST be consolidated into `resolveHostname(fn HostnameFunc, verbose bool, errOut io.Writer) string` in `internal/backup/` (the leaf package). The canonical function MUST live in `internal/backup/` — not `internal/actions/` — because `internal/backup` cannot import `internal/actions` without creating a circular dependency. The function MUST use the injected `HostnameFn` when non-nil, fall back to `os.Hostname()`, and default to `"unknown"` on error.

#### Scenario: hostname returns correct value via injected fn

- GIVEN `HostnameFn` returns `("myhost", nil)`
- WHEN `resolveHostname` is called
- THEN it returns `"myhost"`

#### Scenario: hostname falls back to os.Hostname

- GIVEN `HostnameFn` is nil
- WHEN `resolveHostname` is called
- THEN it calls `os.Hostname()` and returns the result

#### Scenario: hostname defaults to unknown on error

- GIVEN `HostnameFn` returns an error AND `os.Hostname()` also fails
- WHEN `resolveHostname` is called
- THEN it returns `"unknown"`
- AND a warning is written to `errOut` when verbose is true

### Requirement: loadConfig helper consolidated

Two identical sites of config loading (`pull.go:71-83`, `push.go:178-185`) MUST be consolidated into a `loadConfigOr(loader ConfigLoader) (*config.Config, error)` method. The method MUST use the injected `ConfigLoader` when non-nil, falling back to `config.Load()`.

#### Scenario: loadConfig returns correct config via injected loader

- GIVEN `ConfigLoader` returns a valid config
- WHEN `loadConfigOr` is called
- THEN it returns the injected config

#### Scenario: loadConfig falls back to config.Load

- GIVEN `ConfigLoader` is nil
- WHEN `loadConfigOr` is called
- THEN it calls `config.Load()` and returns the result

#### Scenario: error handling preserved

- GIVEN `ConfigLoader` returns an error
- WHEN `loadConfigOr` is called
- THEN the error is propagated to the caller
