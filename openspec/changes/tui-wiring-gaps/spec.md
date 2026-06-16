# Delta Specs for tui-wiring-gaps

## tui-components (MODIFIED — toast wiring)

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

## tui-dashboard (MODIFIED — search integration)

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

## tui-main-menu (MODIFIED — menu items and wizard)

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

#### Scenario: Wizard screen implemented

- GIVEN `ScreenWizard` remains in the `Screen` enum
- WHEN the user navigates to the wizard screen
- THEN a wizard screen implementation SHALL exist in `screens/wizard.go` and render without panic

## tui-action-dispatch (NEW)

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
