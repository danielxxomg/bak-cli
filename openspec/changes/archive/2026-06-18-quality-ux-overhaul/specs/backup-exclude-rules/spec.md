# backup-exclude-rules Specification

## Purpose

Default exclusion patterns, `.bakignore` file support, and max-file-size cap for backup operations. Currently adapters walk ALL files with zero filtering (`generic.go:160-198`, `opencode/adapter.go:129-169`), causing 8.4 MB full-preset backups.

## Requirements

### Requirement: Default exclusion patterns

The system MUST exclude the following patterns by default during directory scanning: `node_modules`, `.git`, `*.lock`, `*.log`, and binary files (`*.png`, `*.jpg`, `*.zip`, `*.tar`, `*.gz`, `*.exe`, `*.dll`, `*.so`, `*.dylib`).

#### Scenario: node_modules excluded

- GIVEN a plugin directory contains `node_modules/` with 50 MB of dependencies
- WHEN a full-preset backup runs
- THEN the `node_modules/` directory SHALL be skipped entirely and not appear in the manifest

#### Scenario: lock files excluded

- GIVEN a skills directory contains `package-lock.json` and `yarn.lock`
- WHEN a backup runs
- THEN neither lock file SHALL appear in the backup manifest

### Requirement: Custom ignore file

The system MUST read `~/.config/bak/ignore` (gitignore syntax) and merge its patterns with the default exclusion list.

#### Scenario: Custom pattern applied

- GIVEN `~/.config/bak/ignore` contains `*.tmp`
- WHEN a backup runs
- THEN files matching `*.tmp` SHALL be excluded from the backup

#### Scenario: Ignore file reload

- GIVEN the ignore file is edited while the bak process is running
- WHEN a new backup starts
- THEN the updated ignore patterns SHALL be read from disk

#### Scenario: Ignore file missing

- GIVEN `~/.config/bak/ignore` does not exist
- WHEN a backup runs
- THEN only default exclusion patterns SHALL apply (no error)

### Requirement: Max file size cap

The system MUST skip individual files exceeding `MaxFileSize` (default: 1 MB). This value SHALL be configurable via settings.

#### Scenario: Large file skipped

- GIVEN a 5 MB file exists in `skills/` and `MaxFileSize` is 1 MB
- WHEN a backup runs
- THEN the file SHALL be skipped and a warning SHALL be emitted to stderr

#### Scenario: Custom max size

- GIVEN `MaxFileSize` is set to 5 MB in config
- WHEN a backup runs on a directory with a 3 MB file
- THEN the 3 MB file SHALL be included in the backup

### Requirement: Opt-out via config

The system MUST allow users to clear or override the default exclusion patterns via the `exclude_patterns` field in settings.

#### Scenario: Clear defaults

- GIVEN `exclude_patterns` is set to an empty array in config
- WHEN a backup runs
- THEN NO default patterns SHALL apply (all files included except `.bakignore` patterns)

#### Scenario: Override defaults

- GIVEN `exclude_patterns` is set to `["*.tmp", "build/"]`
- WHEN a backup runs
- THEN only `*.tmp` and `build/` SHALL be excluded (default patterns replaced)
