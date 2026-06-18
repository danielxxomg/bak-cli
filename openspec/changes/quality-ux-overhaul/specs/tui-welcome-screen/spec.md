# tui-welcome-screen Specification

## Purpose

First-run welcome screen displayed when no bak configuration exists. Guides the user through initial setup and navigates to the main menu. Uses existing `screens/welcome.go` (`RenderWelcome`, `ShouldShowWelcome`) which are currently dead code (never called from `model.go`).

## Requirements

### Requirement: First-run detection

The system MUST show the Welcome screen when `Deps.ConfigExists` returns false during TUI initialization.

#### Scenario: No config exists

- GIVEN `~/.config/bak/config.json` does not exist
- WHEN `bak` launches with no arguments in a TTY
- THEN the Welcome screen SHALL be displayed as the initial screen

#### Scenario: Config already exists

- GIVEN `~/.config/bak/config.json` exists
- WHEN `bak` launches
- THEN the main menu SHALL be displayed (Welcome screen skipped)

### Requirement: Welcome content

The system MUST display the bak tagline, a brief description of what the tool does, and prompt the user to press Enter to continue.

#### Scenario: Welcome renders

- GIVEN the Welcome screen is active
- WHEN the screen renders
- THEN it SHALL show the ASCII logo, "Pack your AI coding setup. Move anywhere.", and "Press Enter to get started"

### Requirement: Navigate to main menu

The system MUST transition to the main menu when the user presses Enter on the Welcome screen.

#### Scenario: Enter pressed

- GIVEN the Welcome screen is displayed
- WHEN the user presses Enter
- THEN the screen SHALL transition to `ScreenMenu`

#### Scenario: Quit from welcome

- GIVEN the Welcome screen is displayed
- WHEN the user presses 'q'
- THEN the TUI SHALL quit cleanly with exit code 0
