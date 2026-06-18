# Delta: welcome-content

## MODIFIED Requirements

### Requirement: Welcome screen content
The first-run welcome screen MUST display the ASCII logo, the tagline, and a prompt to continue.

(Previously: `RenderWelcome` showed `"Welcome to bak!"` heading with generic first-time text. No ASCII logo, no tagline, and prompt wording was `"Press enter to continue, or q to quit."` instead of the spec-mandated content.)

#### Scenario: Welcome shows ASCII logo

- GIVEN the user is on the welcome screen (first run)
- WHEN `RenderWelcome` is called
- THEN the output MUST include the ASCII logo from `styles.RenderLogo()`

#### Scenario: Welcome shows tagline

- GIVEN the user is on the welcome screen
- WHEN `RenderWelcome` is called
- THEN the output MUST contain the exact text `"Pack your AI coding setup. Move anywhere."`

#### Scenario: Enter navigates to main menu

- GIVEN the user is on the welcome screen
- WHEN the user presses Enter
- THEN the TUI MUST transition to `ScreenMenu`
- AND the main menu MUST be displayed
