# tui-restore-screen Specification

## Purpose

Restore flow in the TUI: select a backup from a list, preview the dry-run diff, confirm via modal, and execute the restore. Integrates with `internal/restore/` engine and `internal/diff/` for preview.

## Requirements

### Requirement: Backup list display

The system MUST display a scrollable list of available backups when the user navigates to the Restore screen. Each entry MUST show backup ID, date, preset, and source hostname.

#### Scenario: Populate from disk

- GIVEN `Deps.ListBackups` returns 3 backups
- WHEN the user presses Enter on "Restore" (menu cursor 1)
- THEN the Restore screen SHALL render a table with 3 rows sorted by date descending

#### Scenario: Empty state

- GIVEN no backups exist on disk
- WHEN the user navigates to the Restore screen
- THEN the screen SHALL display "No backups found. Create one first." and offer navigation back to the main menu

### Requirement: Dry-run diff preview

The system MUST show a dry-run diff preview before executing any restore. This is a mandatory gate per AGENTS.md: "MUST NOT restore without mandatory dry-run diff."

#### Scenario: Preview selected backup

- GIVEN the user has selected a backup row in the restore list
- WHEN the user presses Enter on the selected row
- THEN the system SHALL call `Deps.RunRestore` with `DryRun: true` and display the resulting diff output

#### Scenario: Diff shows file changes

- GIVEN a backup with 5 files that differ from current config
- WHEN the dry-run preview renders
- THEN the diff SHALL list each file with added/modified/removed indicators

### Requirement: Confirm before execute

The system MUST present a confirmation modal after the dry-run preview and before executing the actual restore.

#### Scenario: User confirms

- GIVEN the dry-run diff is displayed
- WHEN the user presses Enter on "Confirm" in the modal
- THEN the system SHALL call `Deps.RunRestore` with `DryRun: false` and execute the restore

#### Scenario: User cancels

- GIVEN the dry-run diff is displayed
- WHEN the user presses Escape or selects "Cancel"
- THEN the system SHALL return to the backup list without modifying any files

### Requirement: Restore execution with feedback

The system SHALL display progress during restore execution and show a success or error toast on completion.

#### Scenario: Successful restore

- GIVEN the user confirmed the restore
- WHEN `Deps.RunRestore` completes without error
- THEN a success toast SHALL appear with "Restored successfully" and the screen SHALL return to the main menu

#### Scenario: Restore error

- GIVEN the user confirmed the restore
- WHEN `Deps.RunRestore` returns an error
- THEN an error toast SHALL appear with the error description and the screen SHALL remain on the restore screen
