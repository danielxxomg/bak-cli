# Spec: Generic Adapter Base Struct

## Requirements

### Requirement: GenericAdapter base struct
The system MUST provide a `GenericAdapter` struct in `internal/adapters/generic.go` with fields: `AdapterName string`, `ConfigRelPath string`, `Categories map[string]CategoryDir`, `DetectErrContext string`, `ScanOpts ScanOptions`, `RootConfigFiles map[string]string`, and `StatFn func(string) (os.FileInfo, error)`.

#### Scenario: Construction
- GIVEN adapter constants
- WHEN `GenericAdapter{...}` is constructed
- THEN it requires no additional initialization

#### Scenario: CategoryDir exported
- GIVEN `adapters` package
- THEN `CategoryDir` type is exported with `SubPath string` and `IsDir bool`

### Requirement: GenericAdapter implements Adapter
`GenericAdapter` MUST satisfy the `adapters.Adapter` interface.

#### Scenario: Compile-time check
- GIVEN `GenericAdapter` defined
- THEN `var _ adapters.Adapter = (*GenericAdapter)(nil)` compiles

#### Scenario: Detect behavior
- GIVEN a home directory with `~/.codex/`
- WHEN `GenericAdapter.Detect(homeDir)` is called
- THEN it returns `(true, path, nil)` for existing directories
- AND `(false, path, nil)` for missing directories
- AND wraps errors with per-adapter context

#### Scenario: ListItems behavior
- GIVEN `~/.codex/instructions/` exists
- WHEN `ListItems(homeDir, []string{"instructions","config"})` runs
- THEN it returns items for directory scan and root-level files
- AND items include `Category`, `SourcePath`, `RelPath`, `IsDir`, `Hash`, `Size`

#### Scenario: Backup behavior
- GIVEN home and backup directories
- WHEN `Backup(homeDir, backupDir, items)` runs
- THEN files copy to `backupDir/<adapterName>/<relPath>`
- AND directories create via `os.MkdirAll`

#### Scenario: Restore behavior
- GIVEN backup and home directories
- WHEN `Restore(backupDir, homeDir, items)` runs
- THEN files copy from `backupDir/<adapterName>/<relPath>` to `homeDir/<configRelPath>/<relPath>`
- AND directories create via `os.MkdirAll`

### Requirement: Adapter migration
The 7 adapters (codex, cursor, kiro, kilocode, pidev, windsurf, claudecode) MUST delegate to `GenericAdapter`.

#### Scenario: Thin wrapper
- GIVEN `internal/adapters/codex/adapter.go`
- THEN it defines constants, constructs `GenericAdapter`, and delegates all methods
- AND package body is ≤30 lines

#### Scenario: All 7 refactored
- GIVEN the 7 adapter packages
- THEN each uses `GenericAdapter` for `Detect`, `ListItems`, `Backup`, `Restore`

### Requirement: Behavioral preservation
The refactor MUST produce zero behavioral changes.

#### Scenario: Test compatibility
- GIVEN `go test ./...`
- WHEN executed after refactor
- THEN zero new failures occur
- AND no test files are modified

#### Scenario: Error context preservation
- GIVEN `Detect` on missing directory
- WHEN error messages are compared
- THEN they remain identical per-adapter

### Requirement: Registration preservation
The adapter registration pattern MUST remain unchanged.

#### Scenario: RegisterAll unchanged
- GIVEN `register/register.go`
- THEN it still imports and registers `&codex.Adapter{}`, etc.
- AND no new registration logic is added

### Requirement: AGENTS.md compliance
The GenericAdapter implementation MUST comply with AGENTS.md code review rules.

#### Scenario: No filepath.ToSlash
- GIVEN path computation in GenericAdapter
- THEN it MUST NOT use `filepath.ToSlash`
- AND MUST use `path.Clean(strings.ReplaceAll(path, "\\", "/"))`

#### Scenario: Error wrapping
- GIVEN any error returned
- THEN it MUST be wrapped with `fmt.Errorf("context: %w", err)`
- AND context MUST start with lowercase
