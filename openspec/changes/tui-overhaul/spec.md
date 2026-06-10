# Delta Specs for tui-overhaul

## tui-theme (NEW)

### Requirement: Rose Pine theme system

The system MUST define a Rose Pine palette with 11 semantic colors and an ASCII art logo with 5-band gradient.

#### Scenario: Palette available

- GIVEN `internal/tui/styles/theme.go` is compiled
- WHEN a lipgloss style references `ColorLavender`
- THEN rendered output contains the `#c4a7e7` ANSI sequence

#### Scenario: Logo on narrow terminal

- GIVEN terminal width is less than 40 columns
- WHEN main menu renders the logo
- THEN the logo is truncated or hidden to prevent overflow

## tui-components (NEW)

### Requirement: Reusable component library

The system MUST provide shared menu, checkbox, frame, and help bar renderers using package-level lipgloss styles.

#### Scenario: Menu navigation

- GIVEN a menu with 4 items and cursor at index 0
- WHEN `tea.KeyMsg{Runes: []rune{'j'}}` is sent to `Update()`
- THEN cursor moves to index 1 and `▸` indicator follows

#### Scenario: Checkbox toggle

- GIVEN a checkbox list with first item unchecked
- WHEN `tea.KeyMsg{Type: tea.KeySpace}` is sent
- THEN the item is marked checked and its style switches to `ColorGreen`

## tui-main-menu (NEW)

### Requirement: Interactive main menu

The system MUST launch a TUI main menu when `bak` is run with zero arguments in a TTY, and route input to the active screen.

#### Scenario: TTY launch

- GIVEN `len(args) == 0` and `isTTY() == true`
- WHEN `rootCmd` executes
- THEN `tea.NewProgram` is called with the root model and alt-screen is active

#### Scenario: Help flag preserved

- GIVEN `--help` is passed to `bak`
- WHEN `rootCmd` parses flags
- THEN cobra help text is printed and TUI is NOT launched

## tui-dashboard (NEW)

### Requirement: Backup dashboard

The system MUST display local backups in a responsive table with columns ID, Date, Size, Status, and Cloud.

#### Scenario: Populated table

- GIVEN two local backups exist
- WHEN the dashboard screen renders
- THEN the table shows both rows with correct column values

#### Scenario: Empty state

- GIVEN no local backups exist
- WHEN the dashboard screen renders
- THEN an empty-state message is shown instead of a table

## tui-search (NEW)

### Requirement: Fuzzy search

The system MUST activate a search input when `/` is pressed and filter the active list by query.

#### Scenario: Search and filter

- GIVEN a list with 5 items and search input focused
- WHEN query "conf" is entered
- THEN only items matching "conf" remain visible

#### Scenario: No matches

- GIVEN search query "xyz"
- WHEN the filter is applied
- THEN a "no matches" message is shown

## tui-settings (NEW)

### Requirement: Interactive settings screen

The system MUST render a settings screen with categories, cloud provider, and theme options, allowing selection and toggling.

#### Scenario: Navigate and toggle

- GIVEN the settings screen is active
- WHEN `j`/`k` moves cursor and `Enter` toggles the selected option
- THEN the option state changes and visual feedback is rendered

#### Scenario: Back to menu

- GIVEN the settings screen is active
- WHEN `q` or `Esc` is pressed
- THEN the screen returns to the main menu

## bak-cli (ADDED)

### Requirement: No-args TUI launch

The system MUST add a `RunE` to `rootCmd` that launches the TUI main menu when no arguments are provided and the output is a TTY.

#### Scenario: Interactive mode

- GIVEN `bak` is invoked with no arguments in a TTY
- WHEN `rootCmd` executes
- THEN it returns the result of `tea.NewProgram(model).Run()`

#### Scenario: Non-interactive fallback

- GIVEN `bak` is invoked with no arguments in a non-TTY
- WHEN `rootCmd` executes
- THEN it falls back to standard cobra help output

### Requirement: Window size responsiveness

The system MUST handle `tea.WindowSizeMsg` in all TUI models so that bordered content adapts to terminal dimensions.

#### Scenario: Resize update

- GIVEN a TUI model is active
- WHEN `tea.WindowSizeMsg{Width: 120, Height: 40}` is received
- THEN the model stores the new dimensions and `View()` uses them

#### Scenario: Minimum size guard

- GIVEN terminal dimensions are below 20x10
- WHEN `View()` renders
- THEN a "terminal too small" message is shown instead of the normal layout
