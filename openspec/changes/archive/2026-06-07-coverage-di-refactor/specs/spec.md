# Delta for coverage-di-refactor

## ADDED Requirements

### Requirement: E2E guardrail test
The system MUST include an end-to-end test that creates a backup with real files, restores it, and verifies SHA-256 checksums match the manifest.

#### Scenario: Backup→restore round-trip
- GIVEN a temporary directory with config files
- WHEN `bak backup` then `bak restore` run against that directory
- THEN restored files match original checksums
- AND no data loss occurs

#### Scenario: E2E coverage threshold
- GIVEN CI coverage report
- THEN total threshold MUST be 70% until PR3 restores it to 80%

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

## MODIFIED Requirements

None — external behavior remains identical; all changes are internal wiring.

## REMOVED Requirements

None.
