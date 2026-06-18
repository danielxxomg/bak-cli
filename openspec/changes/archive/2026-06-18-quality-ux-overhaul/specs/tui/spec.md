# Delta for tui

## ADDED Requirements

### Requirement: Centralized terminal size guard

The system MUST provide a single `styles.IsTooSmall(w, h int) bool` helper with unified constants (`MinWidth=40`, `MinHeight=12`). All sub-screens MUST use this helper instead of re-implementing the check.

#### Scenario: Consistent minimums

- GIVEN 6 sub-screens currently implement their own size check
- WHEN any sub-screen receives a `tea.WindowSizeMsg`
- THEN it SHALL call `styles.IsTooSmall(w, h)` — no local constants

#### Scenario: Quit from too-small view

- GIVEN the terminal is below minimum size
- WHEN the "Terminal too small" message is displayed
- THEN pressing 'q' SHALL quit the TUI cleanly

### Requirement: Help overlay on every screen

The system MUST show keybindings when the user presses `?` on any screen.

#### Scenario: Help from sub-screen

- GIVEN the user is on the Settings screen
- WHEN the user presses `?`
- THEN a help overlay SHALL display the keybindings for the current screen

#### Scenario: Dismiss help

- GIVEN the help overlay is visible
- WHEN the user presses `?` again or Escape
- THEN the overlay SHALL close and return to the previous screen

## MODIFIED Requirements

### Requirement: Menu cursor 1 (Restore) has a handler

The system MUST respond to enter on menu cursor 1 (Restore) by navigating to the Restore screen (`ScreenRestore`) where the user can select a backup and preview a dry-run diff.

(Previously: displayed a "coming soon" toast via `Toast.Show()` with no screen transition)

#### Scenario: Restore pressed

- GIVEN the cursor is at index 1 (Restore)
- WHEN the user presses enter
- THEN the system SHALL navigate to `ScreenRestore` and display the backup list

#### Scenario: Profiles pressed

- GIVEN the cursor is at index 4 (Profiles)
- WHEN the user presses enter
- THEN the system SHALL navigate to `ScreenProfiles` and display the profile list

### Requirement: Toast notification on action completion

The system MUST display a toast notification positioned at the bottom-right of the screen using `lipgloss.Place` when a backup or restore action completes successfully or fails. The toast MUST have a visible border and background.

(Previously: toast was appended to the bottom of content as a plain newline-prefixed string with no positioning)

#### Scenario: Backup success toast

- GIVEN a backup action completes without error
- WHEN the action result message is received by the root model
- THEN `Toast.Show()` SHALL be called with a success message and a positive TTL
- AND the toast SHALL render at the bottom-right corner with a border

#### Scenario: Backup error toast

- GIVEN a backup action returns an error
- WHEN the error message is received by the root model
- THEN `Toast.Show()` SHALL be called with an error description and a positive TTL
- AND the toast SHALL render at the bottom-right corner with a border

#### Scenario: Toast auto-hides

- GIVEN a toast is visible with TTL of 3 seconds
- WHEN 3 seconds elapse
- THEN the toast SHALL hide automatically

#### Scenario: Narrow terminal toast

- GIVEN terminal width is less than 50 columns
- WHEN a toast is triggered
- THEN the toast SHALL render inline at the bottom without overlapping existing content
