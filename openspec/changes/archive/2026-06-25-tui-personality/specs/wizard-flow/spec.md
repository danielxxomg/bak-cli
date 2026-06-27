# Delta for wizard-flow

## ADDED Requirements

### Requirement: Paste support in wizard text inputs

The wizard text input fields (profile name, and any free-text wizard step) MUST accept bracketed paste via the v2 `tea.PasteMsg` message. The model MUST append `msg.Content` to the active input buffer. `DisableBracketedPasteMode` MUST remain `false` (the default) so the terminal can deliver paste events. Plain keystrokes MUST continue to work alongside paste.

#### Scenario: paste inserts pasted text

- GIVEN the wizard is on the profile-name step with an empty input
- WHEN a `tea.PasteMsg{Content: "work-laptop"}` arrives
- THEN the input buffer MUST equal `"work-laptop"`

#### Scenario: paste appends to existing text

- GIVEN the input already contains `"work-"`
- WHEN a `tea.PasteMsg{Content: "laptop"}` arrives
- THEN the input buffer MUST equal `"work-laptop"`

#### Scenario: regular keys still work after paste

- GIVEN paste has populated the input with `"work-laptop"`
- WHEN the user presses Backspace
- THEN the input buffer MUST become `"work-lapto"`

## MODIFIED Requirements

### Requirement: Profile creation via wizard

The system MUST launch the 5-step interactive wizard when the user presses 'n' on the profiles screen. The wizard result MUST create a real profile, not a hardcoded stub. The wizard text inputs MUST accept bracketed paste (per "Paste support in wizard text inputs").

(Previously: wizard inputs only accepted character-at-a-time keystrokes; multi-character paste was not handled.)

#### Scenario: tuiRunWizard launches real wizardModel

- GIVEN the user presses 'n' on the profiles screen
- WHEN `tuiRunWizard` is invoked
- THEN it MUST launch the `wizardModel` via `tea.NewProgram`
- AND the wizard MUST present all 5 steps (name, provider, preset, adapters, confirm)

#### Scenario: Wizard result creates real profile

- GIVEN the user completes all 5 wizard steps
- WHEN the wizard finishes
- THEN the returned `ProfileInfo` MUST contain user-selected values (not hardcoded defaults)
- AND the profile MUST be saved via `tuiSaveProfile`

#### Scenario: Wizard cancel returns to profiles

- GIVEN the user is in the wizard
- WHEN the user presses 'q' or Esc to cancel
- THEN `tuiRunWizard` MUST return an error or zero-value `ProfileInfo`
- AND the profiles screen MUST NOT add a new profile