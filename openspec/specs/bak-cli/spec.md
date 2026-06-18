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

### Requirement: Cross-platform path normalization
The system MUST store canonical paths, translate on restore, and use `strings.ReplaceAll(path, "\\", "/")` in `canonicalPath()` for cross-platform path comparison.

#### Scenario: Windows to Linux
- GIVEN backup created on Windows
- WHEN restored on Linux
- THEN paths adapted to `~/.config/opencode/`

#### Scenario: Windows path on Linux
- GIVEN a Windows-style path `C:\Users\foo`
- WHEN `canonicalPath()` runs on Linux
- THEN backslashes are replaced with forward slashes

#### Scenario: Unix path unchanged
- GIVEN a Unix path `/home/foo`
- WHEN `canonicalPath()` runs
- THEN it remains `/home/foo`

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

#### Scenario: setConfigHome helper
- GIVEN any config test
- WHEN `setConfigHome(t, dir)` is called
- THEN the test is isolated from the real user config directory
- AND behavior is correct on Linux, macOS, and Windows

### Requirement: Action DI wiring
The system MUST inject dependencies into `internal/actions/` and `cmd/` instead of calling `os` or `hostname` directly.

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

#### Scenario: RegistryFactory injection
- GIVEN `ListCloudAction` struct
- THEN it accepts `RegistryFactory func() cloud.ProviderRegistry` as a field
- AND tests inject a factory returning mock registries

#### Scenario: CmdDeps struct
- GIVEN a command needs testable dependencies
- THEN `cmdDeps` in `cmd/deps.go` provides `ConfigLoader`, `Stdout`, `Stderr`, `Stdin` fields

#### Scenario: Wrapper pattern
- GIVEN a `runX` function is the cobra `RunE` target
- WHEN it executes
- THEN it delegates to `runXWithDeps(cmd, args, deps)` passing `defaultDeps`

#### Scenario: Test isolation
- GIVEN a test uses injected `CmdDeps`
- WHEN config loading occurs
- THEN it uses the mock loader, never calling `os.UserConfigDir`
- AND output writes to `deps.Stdout`/`deps.Stderr`, not real `os.Stdout`/`os.Stderr`

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

#### Scenario: All-lint-green
- GIVEN `golangci-lint run` executes
- THEN it exits 0 with zero warnings on Ubuntu, macOS, and Windows

#### Scenario: GGA pre-commit with bypass path
- GIVEN a commit is prepared
- WHEN GGA pre-commit validation runs against AGENTS.md
- THEN it passes without `--no-verify` bypass
- OR if GGA fails due to a technical failure (ARG_MAX overflow, provider outage, scope-of-change mismatch), the commit body MUST contain `NO-VERIFY: <reason>` and a follow-up fix commit MUST be created in the same PR

#### Scenario: Docker test pass
- GIVEN `task test:linux` (Docker)
- WHEN it executes
- THEN all tests pass inside the Linux container

#### Scenario: 3-OS CI matrix
- GIVEN GitHub Actions CI with Ubuntu, macOS, Windows
- WHEN `go test ./...` runs
- THEN all three jobs report success

### Requirement: Platform-specific test skipping
The system MUST skip Windows-specific tests on non-Windows via `runtime.GOOS` check.

#### Scenario: Windows-only test on Linux
- GIVEN a test requires Windows registry or `schtasks`
- WHEN `runtime.GOOS != "windows"`
- THEN it calls `t.Skip()` immediately

### Requirement: Lint violation remediation
The system MUST fix golangci-lint violations so `golangci-lint run` exits 0.

#### Scenario: SA5011 nil dereference
- GIVEN `cmd/export_test.go:38`
- WHEN a pointer may be nil
- THEN a nil check precedes dereference

#### Scenario: QF1012 redundant Fprint+Sprintf
- GIVEN `cmd/pick.go:109` and `cmd/wizard.go:283,323`
- WHEN `fmt.Fprint(w, fmt.Sprintf(...))` is used
- THEN it is replaced with `fmt.Fprintf(w, ...)`

#### Scenario: QF1001 De Morgan simplification
- GIVEN `internal/cloud/pack_test.go:130`
- WHEN a boolean expression is over-complex
- THEN it is simplified per staticcheck recommendation

#### Scenario: SA9003 empty branch
- GIVEN `internal/config/migration_test.go:142`
- WHEN an empty if/else branch exists
- THEN it is removed or commented with intent

#### Scenario: SA4023 interface comparison
- GIVEN `internal/schedule/scheduler_unix_test.go:11`
- WHEN a nil interface is compared
- THEN comparison uses concrete type `== nil` or proper interface check

### Requirement: macOS config path isolation
The system MUST use `setConfigHome(t, dir)` helper so tests pass on macOS.

#### Scenario: macOS config resolution
- GIVEN `runtime.GOOS == "darwin"`
- WHEN `setConfigHome(t, dir)` is called
- THEN it sets `HOME` to a temp directory so `os.UserConfigDir()` resolves under that temp tree
- AND `XDG_CONFIG_HOME` is also set for cross-tool consistency

#### Scenario: Linux config resolution
- GIVEN `runtime.GOOS == "linux"`
- WHEN `setConfigHome(t, dir)` is called
- THEN it sets `XDG_CONFIG_HOME` to the temp directory

### Requirement: Action unit test coverage
The system MUST add table-driven unit tests for previously untested action files, covering happy path and error paths.

#### Scenario: login_interactive_test.go
- GIVEN injected stdin and a mock provider
- WHEN `LoginInteractiveAction.Run()` executes
- THEN it returns success or error without real user input

#### Scenario: list_cloud_test.go
- GIVEN a mock `ProviderRegistry` via `RegistryFactory`
- WHEN `ListCloudAction.Run()` executes
- THEN it lists mocked backups and handles empty state

#### Scenario: diff_backups_test.go
- GIVEN two temp directories with backup manifests
- WHEN `DiffBackupsAction.Run()` executes
- THEN it returns a diff or an error for missing files

#### Scenario: verify_backup_test.go
- GIVEN a temp directory with a manifest and checksums
- WHEN `VerifyBackupAction.Run()` executes
- THEN it validates SHA-256 matches or reports mismatch

#### Scenario: pick_backup_test.go
- GIVEN a temp directory with multiple backups
- WHEN `PickBackupAction.Run()` executes
- THEN it selects the correct backup by criteria

#### Scenario: undo_test.go
- GIVEN a temp directory with a restore log
- WHEN `UndoAction.Run()` executes
- THEN it reverts the last restore safely

#### Scenario: schedule_test.go
- GIVEN a temp directory and mock scheduler state
- WHEN `ScheduleAction.Run()` executes
- THEN it schedules or reports scheduling errors

## tui

### Requirement: Toast notification on action completion
The system MUST display a toast notification when a backup or restore action completes successfully or fails.

#### Scenario: Backup success toast
- GIVEN a backup action completes without error
- WHEN the action result message is received by the root model
- THEN `Toast.Show()` SHALL be called with a success message and a positive TTL

#### Scenario: Backup error toast
- GIVEN a backup action returns an error
- WHEN the error message is received by the root model
- THEN `Toast.Show()` SHALL be called with an error description and a positive TTL

#### Scenario: Toast auto-hides
- GIVEN a toast is visible with TTL of 3 seconds
- WHEN 3 seconds elapse
- THEN the toast SHALL hide automatically

### Requirement: Search filters dashboard table rows
The system MUST filter the dashboard table rows based on the active search query.

#### Scenario: Filter with matching rows
- GIVEN the dashboard has 5 backup rows and search is active
- WHEN the user types "conf" in the search input
- THEN the table SHALL display only rows containing "conf" (case-insensitive)

#### Scenario: Filter with no matches
- GIVEN the dashboard has rows and search query is "xyz"
- WHEN the filter is applied
- THEN the table SHALL show an empty state or "no matches" message

#### Scenario: Clear filter
- GIVEN search is active with a query
- WHEN the user presses `Esc` to deactivate search
- THEN the table SHALL restore all original rows

#### Scenario: Empty query shows all rows
- GIVEN search is active with an empty query
- WHEN the filter is evaluated
- THEN all rows SHALL be displayed

### Requirement: Menu cursor 1 (Restore) has a handler
The system MUST respond to enter on menu cursor 1 (Restore) with either a screen transition or user feedback.

#### Scenario: Restore pressed
- GIVEN the cursor is at index 1 (Restore)
- WHEN the user presses enter
- THEN the system SHALL either navigate to a restore screen OR display a "coming soon" toast via `Toast.Show()`

#### Scenario: Profiles pressed
- GIVEN the cursor is at index 4 (Profiles)
- WHEN the user presses enter
- THEN the system SHALL display a "coming soon" toast via `Toast.Show()`

### Requirement: ScreenWizard constant is resolved
The system MUST NOT contain dead screen constants that have no corresponding implementation.

#### Scenario: Wizard constant removed
- GIVEN `ScreenWizard` is removed from the `Screen` enum
- WHEN the code is compiled
- THEN no references to `ScreenWizard` SHALL exist in the codebase

### Requirement: TUI selection routes to cobra actions
The system MUST execute the corresponding backup or restore action after the TUI exits based on the user's menu selection.

#### Scenario: Create backup selected
- GIVEN the user selects "Create backup" (cursor 0) and the TUI exits
- WHEN `defaultRunTUI` receives the `MenuSelection`
- THEN `actions.RunBackup` SHALL be called with the appropriate categories

#### Scenario: Restore selected
- GIVEN the user selects "Restore" (cursor 1) and the TUI exits
- WHEN `defaultRunTUI` receives the `MenuSelection`
- THEN `actions.RunRestore` SHALL be called (or a "coming soon" message if not yet implemented)

#### Scenario: Browse backups selected
- GIVEN the user selects "Browse backups" (cursor 2) and the TUI exits
- WHEN `defaultRunTUI` receives the `MenuSelection`
- THEN no action SHALL be executed (dashboard is an in-TUI screen, not a post-TUI action)

#### Scenario: Quit selected
- GIVEN the user selects "Quit" (cursor 6) and the TUI exits
- WHEN `defaultRunTUI` receives the `MenuSelection`
- THEN no action SHALL be executed and the program SHALL exit cleanly with code 0

#### Scenario: Selection out of bounds
- GIVEN the TUI model has an empty `menuItems` slice
- WHEN `Selection()` is called
- THEN a zero-value `MenuSelection` SHALL be returned and no action SHALL be executed

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
