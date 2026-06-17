# Spec: TUI UX Fixes

## tui-navigation

### Requirement: Arrow key navigation
The system MUST support up/down arrow keys alongside j/k for cursor navigation on all navigable screens.

#### Scenario: Arrow down on main menu
- GIVEN the TUI is on the main menu screen with 7 items
- WHEN the user presses the down arrow key
- THEN the cursor SHALL move to the next item, identical to pressing `j`

#### Scenario: Arrow up on main menu
- GIVEN the cursor is at index 3 on the main menu
- WHEN the user presses the up arrow key
- THEN the cursor SHALL move to index 2, identical to pressing `k`

#### Scenario: Arrow keys on settings screen
- GIVEN the TUI is on the settings screen with 4 options
- WHEN the user presses down arrow then up arrow
- THEN the cursor SHALL move down then back up

#### Scenario: Arrow keys on dashboard
- GIVEN the dashboard table is focused with rows
- WHEN the user presses down arrow
- THEN the key SHALL be forwarded to the table sub-model for row navigation

### Requirement: Wrap-around navigation
The system MUST wrap the cursor from last to first (down) and first to last (up) on the main menu and settings screens.

#### Scenario: Wrap down on menu
- GIVEN the cursor is at the last menu item (index 6, "Quit")
- WHEN the user presses `j` or down arrow
- THEN the cursor SHALL wrap to index 0 ("Create backup")

#### Scenario: Wrap up on menu
- GIVEN the cursor is at index 0 ("Create backup")
- WHEN the user presses `k` or up arrow
- THEN the cursor SHALL wrap to index 6 ("Quit")

#### Scenario: Wrap down on settings
- GIVEN the cursor is at the last settings option (index 3)
- WHEN the user presses `j` or down arrow
- THEN the cursor SHALL wrap to index 0

#### Scenario: Wrap up on settings
- GIVEN the cursor is at index 0 on settings
- WHEN the user presses `k` or up arrow
- THEN the cursor SHALL wrap to index 3

#### Scenario: No wrap-around on dashboard table
- GIVEN the dashboard table cursor is at the last row
- WHEN the user presses `j` or down arrow
- THEN the table SHALL NOT wrap (bubbles/table manages its own cursor)

## tui-help-bar

### Requirement: Persistent help bar on all screens
The system MUST display a contextual help bar footer on every navigable screen, not just the main menu.

#### Scenario: Settings screen help bar
- GIVEN the TUI is on the settings screen
- WHEN View() renders
- THEN a help bar SHALL be displayed with keys: `↑/↓ navigate • enter toggle • q back`

#### Scenario: Dashboard screen help bar
- GIVEN the TUI is on the dashboard screen with data loaded
- WHEN View() renders
- THEN a help bar SHALL be displayed with keys: `↑/↓ navigate • / search • q back`

#### Scenario: Health screen help bar
- GIVEN the TUI is on the health screen in idle state
- WHEN View() renders
- THEN a help bar SHALL be displayed with keys: `enter run • q back`

#### Scenario: Cloud screen help bar
- GIVEN the TUI is on the cloud screen
- WHEN RenderCloudStatus renders
- THEN a help bar SHALL be displayed with keys: `q back`

#### Scenario: Dashboard empty state help bar
- GIVEN the dashboard has no backups
- WHEN View() renders the empty state
- THEN the help bar SHALL still be visible below the "No backups found" message

## tui-dashboard-wiring

### Requirement: ListBackups dependency wired
The system MUST inject a non-nil `ListBackups` function into `tui.Deps` so the dashboard populates with real backup data.

#### Scenario: Dashboard with backups
- GIVEN one or more backups exist on disk
- WHEN the user navigates to the dashboard screen
- THEN the table SHALL display all backup records with ID, Date, Size, Status, and Cloud columns

#### Scenario: Dashboard with no backups
- GIVEN no backups exist on disk
- WHEN the user navigates to the dashboard screen
- THEN the dashboard SHALL display "No backups found" empty state

#### Scenario: Dashboard with ListBackups error
- GIVEN the backup directory is invalid or unreadable
- WHEN the user navigates to the dashboard screen
- THEN the dashboard SHALL display an error message with the underlying error

#### Scenario: ListBackups nil guard
- GIVEN `tui.Deps.ListBackups` is nil (test scenario)
- WHEN the dashboard initializes
- THEN it SHALL return an empty result without panicking

## tui-terminal-minimums

### Requirement: Less aggressive terminal size guard
The system MUST only show "Terminal too small" when the terminal is genuinely too small to render content (below 40 columns or 12 rows).

#### Scenario: Terminal at 40×12
- GIVEN the terminal is exactly 40 columns wide and 12 rows tall
- WHEN the TUI renders
- THEN the normal screen content SHALL be displayed (no "terminal too small" warning)

#### Scenario: Terminal below minimum width
- GIVEN the terminal is 39 columns wide
- WHEN the TUI renders
- THEN "Terminal too small" SHALL be displayed

#### Scenario: Terminal below minimum height
- GIVEN the terminal is 11 rows tall
- WHEN the TUI renders
- THEN "Terminal too small" SHALL be displayed

#### Scenario: Sub-model terminal guard consistency
- GIVEN a sub-screen (settings, dashboard, health) checks terminal dimensions
- WHEN the terminal is at or above 40×12
- THEN the sub-screen SHALL render its normal content, not "terminal too small"
