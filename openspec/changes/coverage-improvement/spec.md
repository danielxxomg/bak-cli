# Delta for bak-cli (Coverage Improvement)

## ADDED Requirements

### Requirement: cmd/ wrapper tests

The system MUST test `runBackup`, `runLogin`, `runPick`, `runPull`, `runPush` delegation via `*WithDeps` with injected mock `CmdDeps`.

#### Scenario: Delegation and error propagation

- GIVEN a cobra command with mock `CmdDeps` where `ConfigLoader` returns an error
- WHEN `runBackup` executes
- THEN it calls `runBackupWithDeps` and returns the wrapped error

#### Scenario: Push with mock factory

- GIVEN `runPushWithDeps` receives a mock `ProviderFactory`
- WHEN it executes
- THEN it creates the provider via the factory and returns success

### Requirement: cmd/ TUI guards

The system MUST verify non-interactive paths for `runLoginWithDeps` and `runPickWithDeps` when `isTTY` is false.

#### Scenario: Non-TTY error

- GIVEN `isTTY` returns false
- WHEN `runLoginWithDeps` or `runPickWithDeps` executes
- THEN it returns a non-interactive error

### Requirement: cmd/ bubbletea model tests

The system MUST test `Update()` and `View()` for each TUI model without `Program.Run()`.

#### Scenario: Picker toggle

- GIVEN a picker model with selectable items
- WHEN `Update()` receives a space key
- THEN the item toggles and `View()` renders the updated state

#### Scenario: Login model input

- GIVEN a login model focused on the token field
- WHEN `Update()` receives a character key
- THEN the token buffer appends it and `View()` masks the input

#### Scenario: Wizard navigation

- GIVEN a wizard model on step 1
- WHEN `Update()` receives a down arrow
- THEN the selection moves to step 2 and `View()` highlights it

### Requirement: actions/ error path tests

The system MUST add unit tests for uncovered error paths in `internal/actions/`.

#### Scenario: saveManifest write failure

- GIVEN a `FileSystem` where `WriteFile` returns an error
- WHEN `saveManifest` is called
- THEN it returns the wrapped error

#### Scenario: scanBackupForSecrets fixture error

- GIVEN a backup directory with a secret pattern file
- WHEN `scanBackupForSecrets` runs
- THEN it returns the scan error

#### Scenario: RunExport create error

- GIVEN a `FileSystem` where `Create` fails for the export path
- WHEN `RunExport` executes
- THEN it returns the create error

#### Scenario: CreateTarGz gzip close error

- GIVEN a `gzip.Writer` that fails on `Close`
- WHEN `CreateTarGz` finishes
- THEN it returns the close error

### Requirement: actions/ FormatSizeBytes edge cases

The system MUST test `FormatSizeBytes` boundary values.

#### Scenario: Boundary values

- GIVEN inputs `0`, `1024`, and `1073741824`
- WHEN `FormatSizeBytes` runs
- THEN it returns `"0 B"`, `"1.0 KB"`, and `"1.0 GB"` respectively

#### Scenario: Large value

- GIVEN input `1099511627776`
- WHEN `FormatSizeBytes` runs
- THEN it returns `"1.0 TB"`

### Requirement: E2E export and undo coverage

The system MUST add E2E testscript tests for `export` and `undo`.

#### Scenario: Export roundtrip

- GIVEN a backup created with `bak backup`
- WHEN `bak export <id>` runs
- THEN a `.tar.gz` containing `manifest.json` is created

#### Scenario: Undo after restore

- GIVEN a restored backup with `--force`
- WHEN `bak undo` runs
- THEN the working tree matches the pre-restore state

### Requirement: AGENTS.md testing compliance

The system MUST NOT add unit tests that call `bubbletea.Program.Run()` or `os.Exit`.

#### Scenario: Compliance review

- GIVEN new test files in `cmd/` and `actions/`
- WHEN they are reviewed
- THEN no test calls `Program.Run()` or `os.Exit` directly
