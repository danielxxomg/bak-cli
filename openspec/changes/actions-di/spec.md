# Delta for actions-di

## ADDED Requirements

### Requirement: BackupAction cobra decoupling

`BackupAction.Run` MUST NOT accept `*cobra.Command`. The signature SHALL be `Run() error`.

#### Scenario: Backup execution without cobra

- GIVEN a `BackupAction` with injected `Stdout` and `Stderr`
- WHEN `Run()` is called
- THEN it executes without `*cobra.Command` parameter
- AND output writes to injected writers

### Requirement: PushAction cobra decoupling

`PushAction.Run` MUST NOT accept `*cobra.Command`. The signature SHALL be `Run(args []string) error`.

#### Scenario: Push execution without cobra

- GIVEN a `PushAction` with injected `Stdout` and `Stderr`
- WHEN `Run(args []string)` is called with a backup ID
- THEN it executes without `*cobra.Command` parameter
- AND output writes to injected writers

### Requirement: RestoreAction cobra decoupling

`RestoreAction.Run` MUST NOT accept `*cobra.Command`. The signature SHALL be `Run() error`. Output MUST write to injected `io.Writer` fields.

#### Scenario: Restore execution without cobra

- GIVEN a `RestoreAction` with injected `Stdout` and `Stderr`
- WHEN `Run()` is called
- THEN it executes without `*cobra.Command` parameter
- AND output writes to injected writers instead of `cmd.OutOrStdout()`

### Requirement: cmd/ caller adaptation

All `cmd/` files constructing actions MUST pass `deps.Stdout` and `deps.Stderr`.

#### Scenario: Command wiring

- GIVEN `cmd/backup.go`, `cmd/push.go`, `cmd/restore.go`
- WHEN they instantiate their respective actions
- THEN `deps.Stdout` and `deps.Stderr` are assigned to action struct fields

### Requirement: Test preservation

Action tests MUST pass with new signatures and injected writers.

#### Scenario: Backup test with buffer

- GIVEN a `BackupAction` test
- WHEN `Run()` is called with `bytes.Buffer` writers
- THEN the test passes without `*cobra.Command`

#### Scenario: Push test with buffer

- GIVEN a `PushAction` test
- WHEN `Run(args)` is called with `bytes.Buffer` writers
- THEN the test passes without `*cobra.Command`

#### Scenario: Restore test with buffer

- GIVEN a `RestoreAction` test
- WHEN `Run()` is called with `bytes.Buffer` writers
- THEN the test passes without `*cobra.Command`

### Requirement: AGENTS.md architecture boundary

`internal/actions/` MUST NOT import `github.com/spf13/cobra`.

#### Scenario: Import check

- GIVEN `internal/actions/backup.go`, `push.go`, `restore.go`
- WHEN imports are analyzed
- THEN `github.com/spf13/cobra` is absent
