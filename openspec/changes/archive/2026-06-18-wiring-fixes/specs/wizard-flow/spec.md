# Delta: wizard-flow

## MODIFIED Requirements

### Requirement: Profile creation via wizard
The system MUST launch the 5-step interactive wizard when the user presses 'n' on the profiles screen. The wizard result MUST create a real profile, not a hardcoded stub.

(Previously: `tuiRunWizard` returned a hardcoded `ProfileInfo{Name:"default", Provider:"github", Preset:"quick"}` without launching the wizard.)

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
