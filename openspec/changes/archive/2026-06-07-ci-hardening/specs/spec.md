# Delta for engineering-quality

## ADDED Requirements

### Requirement: Lint violation remediation
The system MUST fix 7 golangci-lint violations so `golangci-lint run` exits 0.

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

#### Scenario: E2E fixture path expectation
- GIVEN `testdata/e2e/profile_create_list.txtar`
- WHEN the test runs on macOS
- THEN expected config paths match `$HOME/Library/Application Support/bak/`

### Requirement: Cloud list action testability
The system MUST inject `RegistryFactory` into `list_cloud.go` for unit testing without real cloud providers.

#### Scenario: Injected registry
- GIVEN `ListCloudAction` with `RegistryFactory func() cloud.ProviderRegistry`
- WHEN `Run()` is called in a test
- THEN the factory returns a mock registry — no network calls

#### Scenario: Default registry
- GIVEN `ListCloudAction` with nil `RegistryFactory`
- WHEN `Run()` is called
- THEN it uses the default real registry

### Requirement: Action unit test coverage
The system MUST add table-driven unit tests for 7 previously untested action files.

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

## MODIFIED Requirements

### Requirement: CI pipeline fixes
The system MUST fix build and lint issues blocking CI.
(Previously: only lint version pinning and build tag compliance)

#### Scenario: All-lint-green
- GIVEN `golangci-lint run` executes
- THEN it exits 0 with zero warnings on Ubuntu, macOS, and Windows

#### Scenario: GGA pre-commit
- GIVEN a commit is prepared
- WHEN GGA pre-commit validation runs against AGENTS.md
- THEN it passes without `--no-verify` bypass

#### Scenario: Docker test pass
- GIVEN `task test:linux` (Docker)
- WHEN it executes
- THEN all tests pass inside the Linux container

#### Scenario: 3-OS CI matrix
- GIVEN GitHub Actions CI with Ubuntu, macOS, Windows
- WHEN `go test ./...` runs
- THEN all three jobs report success

### Requirement: Action DI wiring
The system MUST inject dependencies into `internal/actions/` instead of calling `os` or `hostname` directly.
(Previously: ProviderFactory, FileSystem, HostnameFunc injection)

#### Scenario: RegistryFactory injection
- GIVEN `ListCloudAction` struct
- THEN it accepts `RegistryFactory func() cloud.ProviderRegistry` as a field
- AND tests inject a factory returning mock registries

### Requirement: Adapter testability
The system MUST test `ConfigAdapter` methods using `t.TempDir()` and injected paths.
(Previously: LoadYAMLAdapters and Register testability)

#### Scenario: setConfigHome helper
- GIVEN any config test
- WHEN `setConfigHome(t, dir)` is called
- THEN the test is isolated from the real user config directory
- AND behavior is correct on Linux, macOS, and Windows

## REMOVED Requirements

None.

## RENAMED Requirements

None.
