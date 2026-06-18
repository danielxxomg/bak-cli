# Delta: error-handling

## MODIFIED Requirements

### Requirement: Error handling discipline
The system MUST handle all returned errors. No `_ =` discards on error returns are permitted. Error messages MUST follow AGENTS.md formatting rules.

(Previously: `profiles.go` discarded errors via `_ = m.setActive(name)` (line 154), `_ = m.deleteProfile(name)` (line 111), and `_ = m.SaveProfile(...)` (line 101). `settings.go` discarded via `_ = m.saveFunc(...)` (line 110).)

#### Scenario: profiles.go handles setActive error

- GIVEN `m.setActive(name)` returns an error
- WHEN the user presses Enter to switch active profile
- THEN the error MUST be displayed in `m.Msg` or surfaced to the user
- AND the `_ =` discard MUST be removed

#### Scenario: profiles.go handles deleteProfile error

- GIVEN `m.deleteProfile(name)` returns an error
- WHEN the user confirms profile deletion
- THEN the error MUST be displayed in `m.Msg`
- AND the `_ =` discard MUST be removed

#### Scenario: profiles.go handles SaveProfile error

- GIVEN `m.SaveProfile(...)` returns an error
- WHEN a wizard-created profile is saved
- THEN the error MUST be displayed in `m.Msg`
- AND the `_ =` discard MUST be removed

#### Scenario: settings.go handles saveFunc error

- GIVEN `m.saveFunc(key, value)` returns an error
- WHEN the user toggles a setting
- THEN the error MUST be displayed (e.g., in `m.msg` or status area)
- AND the `_ =` discard MUST be removed

#### Scenario: Error messages follow AGENTS.md format

- GIVEN an error is displayed to the user
- WHEN the error message is constructed
- THEN it MUST start with lowercase
- AND it MUST include context (what operation failed)
- AND it MUST NOT include sensitive data (tokens, paths with usernames)
