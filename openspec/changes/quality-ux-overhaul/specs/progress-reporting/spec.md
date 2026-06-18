# progress-reporting Specification

## Purpose

Progress callbacks during backup, restore, push, and pull operations. Currently no progress is emitted (`engine.go:52-230` has zero callbacks). The TUI progress screen (`screens/progress.go`) exists but never receives `ProgressStepMsg` in production.

## Requirements

### Requirement: Engine progress callback

The system MUST accept an optional `progressFn` callback on `backup.Engine` and `RestoreAction`. The callback signature MUST be `func(currentFile string, filesDone int, filesTotal int)`.

#### Scenario: Backup emits progress

- GIVEN a backup with 20 files and a non-nil `progressFn`
- WHEN `Engine.Run()` executes
- THEN `progressFn` SHALL be called once per file with the file name, incrementing `filesDone`, and correct `filesTotal`

#### Scenario: Nil-safe callback

- GIVEN `progressFn` is nil
- WHEN `Engine.Run()` executes
- THEN the backup SHALL complete normally without panic (nil guard before each invocation)

### Requirement: Action-level progress

The system MUST propagate `progressFn` from `BackupAction.Run()` and `RestoreAction.Run()` down to the engine.

#### Scenario: BackupAction forwards progress

- GIVEN `BackupAction` has a non-nil `ProgressFn`
- WHEN `BackupAction.Run()` executes
- THEN each engine progress callback SHALL be forwarded to `BackupAction.ProgressFn`

#### Scenario: RestoreAction forwards progress

- GIVEN `RestoreAction` has a non-nil `ProgressFn`
- WHEN `RestoreAction.Run()` executes
- THEN each file restored SHALL trigger a progress callback

### Requirement: TUI progress bridge

The system MUST bridge engine progress callbacks to TUI `ProgressStepMsg` via a goroutine and channel.

#### Scenario: Progress bar updates

- GIVEN the TUI progress screen is active and a backup is running
- WHEN the engine reports file 5 of 20
- THEN `ProgressStepMsg{Step: "file 5/20", Current: 5, Total: 20}` SHALL be sent to the TUI model
- AND the progress bar SHALL show 25%

#### Scenario: Completion message

- GIVEN a backup is running in the TUI
- WHEN the engine completes all files
- THEN `ProgressDoneMsg{}` SHALL be sent and the spinner SHALL stop

### Requirement: Cloud push/pull progress

The system SHOULD emit progress during cloud push and pull operations.

#### Scenario: Push progress

- GIVEN a push operation with 10 files to upload
- WHEN the push executes with a non-nil `progressFn`
- THEN progress SHALL be reported per file uploaded
