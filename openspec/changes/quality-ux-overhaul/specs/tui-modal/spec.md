# tui-modal Specification

## Purpose

Reusable modal dialog component for the TUI. Supports confirmation dialogs, alert messages, and text input prompts. Used by restore (confirm), profiles (delete confirm), and settings screens.

## Requirements

### Requirement: Confirmation dialog

The system MUST render a centered modal with a title, message, and two action buttons (Confirm / Cancel).

#### Scenario: Confirm action

- GIVEN a modal is displayed with title "Confirm Restore" and message "This will overwrite current config."
- WHEN the user presses Enter on "Confirm"
- THEN the modal SHALL close and invoke the `OnConfirm` callback

#### Scenario: Cancel action

- GIVEN a confirmation modal is displayed
- WHEN the user presses Escape or navigates to "Cancel" and presses Enter
- THEN the modal SHALL close and invoke the `OnCancel` callback (if provided)

### Requirement: Alert dialog

The system MUST render a modal with a message and a single "OK" dismiss button.

#### Scenario: Alert dismiss

- GIVEN an alert modal is displayed with message "Backup created successfully"
- WHEN the user presses Enter or Escape
- THEN the modal SHALL close

### Requirement: Keyboard navigation

The system MUST support keyboard navigation within the modal. Tab/Shift+Tab SHALL cycle between buttons. Enter SHALL activate the focused button. Escape SHALL trigger the cancel/dismiss action.

#### Scenario: Tab cycles buttons

- GIVEN a confirmation modal with "Confirm" focused
- WHEN the user presses Tab
- THEN focus SHALL move to "Cancel"

#### Scenario: Escape cancels

- GIVEN any modal is displayed
- WHEN the user presses Escape
- THEN the modal SHALL close via the cancel path

### Requirement: Layout and styling

The system MUST render the modal centered over the existing screen content with a bordered frame using Rose Pine theme colors from `styles/`.

#### Scenario: Modal over content

- GIVEN the restore screen is active and a confirmation modal opens
- WHEN the modal renders
- THEN the underlying screen SHALL be visible but dimmed behind the modal border

#### Scenario: Narrow terminal

- GIVEN terminal width is 40 columns
- WHEN a modal renders
- THEN the modal SHALL fit within the terminal width with at least 2 columns of padding on each side
