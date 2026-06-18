# Delta: restore-flow

## MODIFIED Requirements

### Requirement: Dry-run gate
The system MUST show diff before applying changes. The TUI restore flow MUST call the real `actions.RestoreAction` instead of returning hardcoded strings.

(Previously: `tuiRunRestore` returned `"dry-run: no changes detected"` and `"restored successfully"` without executing any real restore logic.)

#### Scenario: tuiRunRestore calls real RestoreAction

- GIVEN `tuiRunRestore` is invoked with a valid backup ID and `dryRun=true`
- WHEN the function executes
- THEN it MUST construct an `actions.RestoreAction` with injected dependencies
- AND call `RestoreAction.Run()` with the dry-run flag
- AND return the actual diff output from the action

#### Scenario: Dry-run shows real diff output

- GIVEN a backup exists with modified files
- WHEN the user previews the backup in the TUI restore screen
- THEN the dry-run output MUST show actual file-level differences
- AND the output MUST NOT be the hardcoded string `"dry-run: no changes detected"`

#### Scenario: Confirm executes real restore

- GIVEN the user confirms restore after dry-run
- WHEN `tuiRunRestore` is called with `dryRun=false`
- THEN it MUST execute `actions.RestoreAction.Run()` which copies files and verifies checksums
- AND the output MUST NOT be the hardcoded string `"restored successfully"`

#### Scenario: Errors surface to user

- GIVEN `RestoreAction.Run()` returns an error
- WHEN `tuiRunRestore` receives the error
- THEN the error MUST be returned to the caller (not swallowed)
- AND the restore screen MUST display the actual error message
