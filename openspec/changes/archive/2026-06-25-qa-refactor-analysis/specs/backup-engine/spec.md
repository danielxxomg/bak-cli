# Delta for backup-engine

## ADDED Requirements

### Requirement: Consolidated backup engine

`BackupAction.Run` (CLI path) and `Engine.Run` (TUI path) MUST be consolidated into a single shared implementation (`runBackupWorkflow`). Both callers MUST delegate to this implementation. The consolidated engine MUST skip secret files from the manifest via a `secretRelPaths` map (previously only `Engine.Run` did this). Secret file removal MUST use `RemoveAll` for safety (handles directories).

(Previously: Two parallel ~230-line orchestrators with a behavioral delta — `BackupAction.Run` included secret files in the manifest then removed them from disk, producing dangling references.)

#### Scenario: Backup with secrets produces manifest without secret entries

- GIVEN a config directory containing files matching secret patterns (e.g. `*.env` with `API_KEY=ghp_xxx`)
- WHEN backup runs via either CLI (`BackupAction.Run`) or TUI (`Engine.Run`)
- THEN the manifest `Items` array MUST NOT contain entries for secret files
- AND secret files MUST be removed from the backup directory via `RemoveAll`

#### Scenario: Backup without secrets unchanged

- GIVEN a config directory with no secret-pattern files
- WHEN backup runs
- THEN the manifest `Items` count MUST equal the total files scanned
- AND behavior is identical to pre-consolidation

#### Scenario: CLI and TUI paths use same implementation

- GIVEN both `BackupAction.Run` and `Engine.Run` are called with identical inputs
- WHEN each delegates to `runBackupWorkflow`
- THEN the resulting manifests MUST be byte-identical (same Items, same checksums, same ordering)

#### Scenario: Manifest Items count excludes secrets

- GIVEN 10 config files found by adapters, 2 matching secret patterns
- WHEN backup completes
- THEN manifest `Items` length MUST be 8
- AND the 2 secret files MUST NOT appear as dangling references

#### Scenario: Consolidated engine preserves exclusion pipeline

- GIVEN a preset with exclusion rules and adapter scan options
- WHEN `runBackupWorkflow` executes
- THEN `config.LoadExcludes` MUST be called
- AND `SetScanOptions` MUST be called on every detected adapter before `ListItems`
- AND `.bakignore` patterns MUST be applied during `scanDir`
