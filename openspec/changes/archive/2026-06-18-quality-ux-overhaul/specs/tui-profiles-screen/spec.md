# tui-profiles-screen Specification

## Purpose

Profiles management screen in the TUI. A profile is a named set of tool configurations (provider, preset, adapters, categories) used to control what gets backed up. Reuses existing CLI `wizardModel` from `cmd/wizard.go` for creation flow.

## Requirements

### Requirement: Profile list display

The system MUST display a table of existing profiles with name, provider, preset, and active indicator.

#### Scenario: Populate from config

- GIVEN `Deps.ListProfiles` returns 2 profiles
- WHEN the user navigates to the Profiles screen (menu cursor 4)
- THEN a table SHALL render with profile name, provider, preset columns and a marker on the active profile

#### Scenario: No profiles exist

- GIVEN no profiles are configured
- WHEN the user navigates to the Profiles screen
- THEN the screen SHALL display "No profiles yet. Press 'n' to create one."

### Requirement: Create profile via wizard

The system MUST launch the 5-step wizard (Provider → Preset → Adapters → Categories → Confirm) when the user presses 'n' on the Profiles screen.

#### Scenario: Complete wizard

- GIVEN the user presses 'n' on the Profiles screen
- WHEN the user completes all 5 wizard steps and confirms
- THEN the new profile SHALL be persisted via `Deps.SaveProfile` and appear in the profile list

#### Scenario: Cancel wizard

- GIVEN the user is on wizard step 3
- WHEN the user presses Escape
- THEN the wizard SHALL abort and return to the profile list without saving

### Requirement: Switch active profile

The system MUST allow the user to set a profile as active by pressing Enter on it.

#### Scenario: Switch profile

- GIVEN 3 profiles exist and profile "work" is active
- WHEN the user navigates to profile "personal" and presses Enter
- THEN "personal" SHALL become the active profile and the list SHALL update the active marker

### Requirement: Delete profile with confirmation

The system MUST show a confirmation modal before deleting a profile. The active profile MUST NOT be deletable.

#### Scenario: Delete non-active profile

- GIVEN the user selects a non-active profile and presses 'd'
- WHEN the user confirms deletion in the modal
- THEN the profile SHALL be removed via `Deps.DeleteProfile` and the list SHALL refresh

#### Scenario: Cannot delete active profile

- GIVEN the user selects the active profile and presses 'd'
- THEN the system SHALL display a toast "Cannot delete the active profile" and take no further action
