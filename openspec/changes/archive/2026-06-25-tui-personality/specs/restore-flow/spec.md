# Delta for restore-flow

## ADDED Requirements

### Requirement: Dry-run output routed through interactive viewport

The restore dry-run flow MUST render its diff preview through the TUI interactive viewport (per `tui-interactive-preview` / `tui-personality` REQ-TP-005) instead of emitting a raw string dump. The viewport MUST be the sole presentation surface for dry-run content on `restoreStateDryRun`.

#### Scenario: dry-run diff shows in viewport

- GIVEN a backup is selected and the user confirms dry-run preview
- WHEN `restoreDryRunResultMsg` arrives with non-empty `output`
- THEN the diff MUST appear inside a `bubbles/viewport.Model`
- AND the output MUST NOT be appended as a raw string to the screen body

#### Scenario: user scrolls then returns

- GIVEN the viewport is displaying a multi-page diff
- WHEN the user presses `PgDn` then `q`
- THEN the viewport MUST scroll one page down
- AND on `q` the screen MUST return to the backup list (not exit the TUI)

## MODIFIED Requirements

### Requirement: Dry-run gate

The system MUST show diff before applying changes. The TUI restore flow MUST call the real `actions.RestoreAction` instead of returning hardcoded strings.

(Previously: dry-run preview emitted the raw diff string directly to the screen body via `renderDryRun`. The diff is now delivered through a scrollable `bubbles/viewport` so long diffs no longer wrap or push the help line off-screen.)

#### Scenario: tuiRunRestore calls real RestoreAction

- GIVEN `tuiRunRestore` is invoked with a valid backup ID and `dryRun=true`
- WHEN the function executes
- THEN it MUST construct an `actions.RestoreAction` with injected dependencies
- AND call `RestoreAction.Run()` with the dry-run flag
- AND return the actual diff output from the action

#### Scenario: Dry-run shows real diff output in viewport

- GIVEN a backup exists with modified files
- WHEN the user previews the backup in the TUI restore screen
- THEN the dry-run output MUST show actual file-level differences inside the viewport
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