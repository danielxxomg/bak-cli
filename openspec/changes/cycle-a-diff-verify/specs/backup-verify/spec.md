# Delta for backup-verify

## ADDED Requirements

### Requirement: Verify command contract
The system MUST provide `bak verify <id>` that resolves the backup directory, loads `manifest.json`, and validates every file's SHA-256 hash against the manifest.

#### Scenario: Successful verification
- GIVEN a valid backup ID with intact files
- WHEN `bak verify <id>` runs
- THEN all file hashes match the manifest entries
- AND the command exits 0 with a success message including the file count

#### Scenario: Corrupted file detected
- GIVEN a backup where one file's content differs from its manifest hash
- WHEN `bak verify <id>` runs
- THEN the command exits 1 on the first hash mismatch
- AND stderr contains the corrupted file's path

#### Scenario: Missing backup
- GIVEN a non-existent backup ID
- WHEN `bak verify <id>` runs
- THEN the command exits 1 with an error indicating the backup was not found

### Requirement: Path traversal prevention
The system MUST reject backup IDs that resolve outside the backup root directory.

#### Scenario: Traversal blocked
- GIVEN a backup ID containing `../`
- WHEN `bak verify <id>` runs
- THEN the command exits 1 with a path traversal error
- AND no files outside the backup directory are accessed

### Requirement: Output format
The system MUST produce human-readable pass/fail output.

#### Scenario: Verbose progress
- GIVEN a valid backup and `--verbose` flag
- WHEN `bak verify --verbose <id>` runs
- THEN progress of each checked file is printed
