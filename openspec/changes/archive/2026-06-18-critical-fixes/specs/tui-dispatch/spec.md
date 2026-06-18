# TUI Dispatch Specification

## Purpose

Gates post-exit action dispatch behind explicit user confirmation, preventing quit actions from triggering side effects.

## Requirements

### Requirement: RouteSelection requires explicit selection

`RouteSelection` MUST only execute post-exit actions when `MenuSelection.Selected` is `true`. `Selected` MUST be set to `true` only when the user confirms a menu choice via the Enter key.

#### Scenario: Enter confirms selection

- GIVEN cursor is at index 0 (Create backup)
- WHEN user presses Enter
- THEN `MenuSelection.Selected` MUST be `true`
- AND `RouteSelection` dispatches the corresponding action

#### Scenario: Quit does not set Selected

- GIVEN cursor is at index 0 (Create backup)
- WHEN user presses `q`
- THEN `MenuSelection.Selected` MUST be `false`
- AND `RouteSelection` MUST NOT dispatch any action

#### Scenario: Esc does not set Selected

- GIVEN cursor is at any position
- WHEN user presses Esc
- THEN `MenuSelection.Selected` MUST be `false`

### Requirement: Quit triggers no post-exit action

Pressing `q` or `Esc` to quit the TUI MUST NOT trigger any backup, restore, or other side-effect action. The program MUST exit with code 0.

#### Scenario: q exits cleanly

- GIVEN TUI is displayed with cursor at index 0
- WHEN user presses `q`
- THEN no backup directory is created
- AND exit code is 0

#### Scenario: Esc exits cleanly

- GIVEN TUI is displayed
- WHEN user presses Esc
- THEN no action is dispatched
- AND exit code is 0

#### Scenario: Quit menu item exits cleanly

- GIVEN cursor is on "Quit" menu item
- WHEN user presses Enter
- THEN no action is dispatched
- AND exit code is 0
