# Delta for bak-cli

## ADDED Requirements

### Requirement: `bak --version` flag

The system MUST support `bak --version` printing the same version info as `bak version`. `rootCmd.Version` MUST be set.

#### Scenario: Flag prints version

- GIVEN bak is installed
- WHEN `bak --version` runs
- THEN version string MUST be printed
- AND exit code is 0

#### Scenario: Subcommand still works

- GIVEN bak is installed
- WHEN `bak version` runs
- THEN version info MUST still be printed

### Requirement: `bak restore` interactive picker

`bak restore` with no args MUST show an interactive backup picker on TTY. Non-TTY without args MUST error with a helpful message suggesting `bak list`.

#### Scenario: TTY no-arg picker

- GIVEN TTY and multiple backups exist
- WHEN `bak restore` runs with no args
- THEN an interactive picker MUST be shown
- AND the selected backup is used for restore

#### Scenario: Non-TTY no-arg errors

- GIVEN non-TTY environment
- WHEN `bak restore` runs with no args
- THEN an error MUST reference `bak list`

#### Scenario: With arg still works

- GIVEN a valid backup ID
- WHEN `bak restore <id> --dry-run` runs
- THEN dry-run proceeds for that backup

### Requirement: `bak profile create` wizard without name

`bak profile create` with no args and `--interactive` MUST launch the wizard, which collects the profile name.

#### Scenario: Interactive no-arg launches wizard

- GIVEN `bak profile create --interactive` with no name arg
- WHEN the command runs
- THEN the wizard MUST launch and collect the name

#### Scenario: Non-interactive no-arg errors

- GIVEN no `--interactive` flag and no name arg
- WHEN `bak profile create` runs
- THEN an error MUST suggest providing a name or using `--interactive`

#### Scenario: With name arg works

- GIVEN a name argument
- WHEN `bak profile create myprofile` runs
- THEN the profile is created with that name

### Requirement: Main menu footer includes `? help` hint

The main menu footer MUST include a `? help` key hint alongside existing navigation hints.

#### Scenario: Footer shows help hint

- GIVEN the main menu is displayed
- WHEN the footer renders
- THEN it MUST include `? help`

#### Scenario: Other hints preserved

- GIVEN the main menu footer
- THEN `↑/↓ navigate`, `enter select`, and `q quit` MUST still appear

## MODIFIED Requirements

### Requirement: TUI selection routes to cobra actions

The system MUST execute the corresponding backup or restore action after the TUI exits based on the user's menu selection, but ONLY when `MenuSelection.Selected` is `true`.

(Previously: Dispatched on cursor position alone, causing `q` to trigger backup when cursor was at index 0.)

#### Scenario: Create backup selected

- GIVEN the user selects "Create backup" (cursor 0) via Enter and the TUI exits
- WHEN `defaultRunTUI` receives the `MenuSelection`
- THEN `actions.RunBackup` SHALL be called with the appropriate categories

#### Scenario: Restore selected

- GIVEN the user selects "Restore" (cursor 1) via Enter and the TUI exits
- WHEN `defaultRunTUI` receives the `MenuSelection`
- THEN `actions.RunRestore` SHALL be called

#### Scenario: Browse backups selected

- GIVEN the user selects "Browse backups" (cursor 2) via Enter and the TUI exits
- WHEN `defaultRunTUI` receives the `MenuSelection`
- THEN no action SHALL be executed

#### Scenario: Quit via key press

- GIVEN the user presses `q` (cursor at any position)
- WHEN `defaultRunTUI` receives the `MenuSelection`
- THEN `Selected` is `false` and no action SHALL be executed
- AND the program exits cleanly with code 0

#### Scenario: Quit menu item selected

- GIVEN the user selects "Quit" (last cursor) via Enter and the TUI exits
- WHEN `defaultRunTUI` receives the `MenuSelection`
- THEN no action SHALL be executed and the program SHALL exit cleanly with code 0

#### Scenario: Selection out of bounds

- GIVEN the TUI model has an empty `menuItems` slice
- WHEN `Selection()` is called
- THEN a zero-value `MenuSelection` SHALL be returned and no action SHALL be executed
