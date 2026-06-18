# Delta for backup-engine

## ADDED Requirements

### Requirement: scanRootFiles applies ScanOptions

`scanRootFiles` MUST apply `ScanOptions` (exclude patterns and `MaxFileSize`) to every root-level file, matching `scanDir` behavior.

#### Scenario: Root SQLite excluded

- GIVEN `DefaultExcludes` contains `*.sqlite*`
- WHEN `scanRootFiles` encounters `logs.sqlite` in adapter config root
- THEN the file MUST be skipped
- AND it MUST NOT appear in the backup manifest

#### Scenario: Root oversized file excluded

- GIVEN `MaxFileSize` is 1 MB
- WHEN `scanRootFiles` encounters a 5 MB file at root
- THEN the file MUST be skipped with a stderr warning

#### Scenario: Custom excludes apply to root files

- GIVEN `Settings.ExcludePatterns` is `["*.tmp"]`
- WHEN `scanRootFiles` encounters `data.tmp` at root
- THEN the file MUST be skipped

### Requirement: DefaultExcludes covers runtime artifacts

`DefaultExcludes` MUST include `*.sqlite*`, `*.sqlite-wal`, `*.sqlite-shm`, `*.db`, `*_cache.json`, and `*.log`.

#### Scenario: SQLite WAL excluded by default

- GIVEN default configuration with no user overrides
- WHEN backup scans `logs_2.sqlite-wal`
- THEN the file MUST be skipped

#### Scenario: Cache JSON excluded by default

- GIVEN default configuration
- WHEN backup scans `models_cache.json`
- THEN the file MUST be skipped

#### Scenario: User overrides replace defaults

- GIVEN `Settings.ExcludePatterns` is `["*.tmp"]`
- WHEN `LoadExcludes` is called
- THEN only `*.tmp` is used (defaults replaced)

### Requirement: Codex adapter root config whitelist

The Codex adapter MUST restrict root-level backup to a whitelist: `config.toml`, `instructions.md`, `config.json`, `mcp.json`.

#### Scenario: Whitelisted file backed up

- GIVEN `~/.codex/config.toml` exists
- WHEN Codex adapter scans root files
- THEN `config.toml` MUST be included in backup

#### Scenario: Non-whitelisted file skipped

- GIVEN `~/.codex/logs_2.sqlite` exists
- WHEN Codex adapter scans root files
- THEN the file MUST NOT be included in backup

#### Scenario: All whitelisted files present

- GIVEN all four whitelisted files exist in `~/.codex/`
- WHEN Codex adapter scans root files
- THEN exactly those four files are included
