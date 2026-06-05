# bak — Specification

## Capabilities

### backup-engine

### Requirement: Preset-based backup
The system MUST support `quick`, `full`, and `skills` presets.

#### Scenario: Quick preset
- GIVEN OpenCode config exists
- WHEN `bak backup` runs
- THEN manifest created with essential configs only

#### Scenario: Full preset
- GIVEN multiple agent configs exist
- WHEN `bak backup --preset full` runs
- THEN all discoverable configs backed up

### Requirement: Secrets exclusion
The system MUST exclude secrets and generate `.env.example`.

#### Scenario: Secret detected
- GIVEN file contains `API_KEY=secret`
- WHEN backup runs
- THEN secret excluded and `.env.example` generated with placeholder

## restore-engine

### Requirement: Dry-run gate
The system MUST show diff before applying changes.

#### Scenario: Dry-run preview
- GIVEN valid backup ID
- WHEN `bak restore --dry-run <id>` runs
- THEN diff printed, zero files modified

#### Scenario: Git-protected restore
- GIVEN valid backup ID and target git repo
- WHEN `bak restore <id>` runs
- THEN pre-restore auto-committed, diff shown, changes applied, post-restore committed

## cloud-sync

### Requirement: GitHub Gist sync
The system MUST push/pull backups to private GitHub Gists.

#### Scenario: Push round-trip
- GIVEN backup exists and `GITHUB_TOKEN` set
- WHEN `bak push` then `bak pull`
- THEN identical backup restored

## path-normalization

### Requirement: Cross-platform paths
The system MUST store canonical paths and translate on restore.

#### Scenario: Windows to Linux
- GIVEN backup created on Windows
- WHEN restored on Linux
- THEN paths adapted to `~/.config/opencode/`

## agent-adapters

### Requirement: Adapter registry
The system MUST have extensible adapter interface with OpenCode first-class.

#### Scenario: OpenCode discovery
- GIVEN OpenCode installed
- WHEN adapter queried
- THEN correct config paths returned for host OS

#### Scenario: Graceful skip
- GIVEN unregistered agent config exists
- WHEN backup runs
- THEN config ignored without error

## interactive-picker

### Requirement: TUI selection
The system MUST provide bubbletea checkbox UI for selective backup.

#### Scenario: Pick categories
- GIVEN configs exist
- WHEN `bak pick` runs
- THEN user selects categories and backup created

## manifest

### Requirement: Directory format
The system MUST produce `manifest.json` plus agent subdirectories.

#### Scenario: Manifest contents
- GIVEN backup created
- THEN manifest contains version, checksums, os_source, timestamp

#### Scenario: Export archive
- GIVEN backup ID
- WHEN `bak export <id>` runs
- THEN tar.gz created

## Non-functional Requirements
- Performance: backup <2s, restore <5s
- Cross-platform: Windows 10+, macOS 12+, Linux
- Security: HTTPS only, secrets excluded, private gist storage

## Constraints
- No encryption at rest (v2)
- Overwrite only; no merge restore
- No GUI; CLI only
- No session or authentication backup

## Acceptance Criteria
- [ ] `bak backup` creates valid manifest in <2s
- [ ] `bak restore --dry-run` diff matches actual changes
- [ ] Push/pull round-trips without data loss
- [ ] Windows backup restores correctly on macOS/Linux
- [ ] Secrets excluded from all presets by default
