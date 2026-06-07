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

## engineering-quality

### Requirement: E2E guardrail test
The system MUST include an end-to-end test that creates a backup with real files, restores it, and verifies SHA-256 checksums match the manifest.

#### Scenario: Backup→restore round-trip
- GIVEN a temporary directory with config files
- WHEN `bak backup` then `bak restore` run against that directory
- THEN restored files match original checksums
- AND no data loss occurs

#### Scenario: E2E coverage threshold
- GIVEN CI coverage report
- THEN total threshold MUST be 80% until PR3 restores it to 80%

### Requirement: Adapter testability
The system MUST test `ConfigAdapter` methods (`Detect`, `ListItems`, `Backup`, `Restore`) using `t.TempDir()` and injected paths.

#### Scenario: LoadYAMLAdapters injection
- GIVEN a mock home directory
- WHEN `LoadYAMLAdapters(homeDir)` is called
- THEN adapters load from the provided path without calling `os.UserHomeDir`

#### Scenario: Register testability
- GIVEN `register.All()` with injected config path
- WHEN tests run in isolated temp directories
- THEN adapter registry resolves paths deterministically

### Requirement: Action DI wiring
The system MUST inject dependencies into `internal/actions/` instead of calling `os` or `hostname` directly.

#### Scenario: ProviderFactory injection
- GIVEN `Push` and `Pull` actions
- WHEN provider creation is needed
- THEN `ProviderFactory` interface is used — no direct `NewGitHubProvider` call

#### Scenario: restoreFile via FS
- GIVEN a `Restore` action with injected `FileSystem`
- WHEN `restoreFile()` copies a file
- THEN it reads from and writes to the injected `a.FS` — not `os.Open`/`os.Create`

#### Scenario: HostnameFunc injection
- GIVEN an `Actions` struct
- THEN `HostnameFunc` field supplies hostname for manifest metadata
- AND tests inject a static value without calling `os.Hostname`

#### Scenario: Mock provider compliance
- GIVEN a hand-rolled `MockProvider` implementing the provider interface
- THEN compile-time check `var _ Provider = (*MockProvider)(nil)` passes

### Requirement: Command extraction
The system MUST extract all business logic from `cmd/*.go` `RunE` into testable functions in `internal/actions/`.

#### Scenario: Profile CRUD delegation
- GIVEN `bak profile add|remove|list`
- WHEN `RunE` executes
- THEN it delegates to an extracted function testable without cobra

#### Scenario: List local delegation
- GIVEN `bak list`
- WHEN `RunE` executes
- THEN it calls a testable helper for local backup enumeration

#### Scenario: Export delegation
- GIVEN `bak export <id>`
- WHEN `RunE` executes
- THEN it delegates to a testable export function

#### Scenario: Diff delegation
- GIVEN `bak diff <id>`
- WHEN `RunE` executes
- THEN it delegates to a testable diff function

#### Scenario: Verify delegation
- GIVEN `bak verify <id>`
- WHEN `RunE` executes
- THEN it delegates to a testable verify function

#### Scenario: Login stdin injection
- GIVEN `bak login`
- WHEN credentials are read
- THEN stdin is injectable for tests

### Requirement: CI pipeline fixes
The system MUST fix build and lint issues blocking CI.

#### Scenario: Lint version pinning
- GIVEN `.github/workflows/ci.yml`
- THEN `golangci-lint` version matches Go version in `go.mod`

#### Scenario: Build tag compliance
- GIVEN `parseSchtasksCSV` in Windows-specific file
- THEN `//go:build windows` tag is present and the file uses `_windows.go` suffix

#### Scenario: Rate limit resilience
- GIVEN GitHub API rate limiting
- THEN task actions include retry/backoff or caching workaround

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
