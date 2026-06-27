# tui-personality Specification

## Purpose

Visual and behavioral "personality" affordances layered onto the existing bak-cli TUI: contextual terminal window title, rotating step indicators, a persistent status bar, a gradient logo, a scrollable dry-run viewport, mouse list navigation, and styled empty states. All additive, render-only, testable as pure `Update`/`View` functions.

## Requirements

### Requirement: Contextual terminal window title

The TUI MUST set the terminal window title to `bak — {Screen}` on every screen. The title MUST reflect the active screen and, when on `ScreenProgress` with a running operation, include the step counter as `bak — Backup {current}/{total}`. The title MUST be set declaratively via the `tea.View.WindowTitle` field (v2 API), NOT via a `tea.SetWindowTitle` command.

#### Scenario: backup screen shows Backup title

- GIVEN the active screen is `ScreenProgress` with a running backup at step 3 of 7
- WHEN `Model.View()` is called
- THEN `View().WindowTitle` MUST equal `bak — Backup 3/7`

#### Scenario: wizard shows Wizard title

- GIVEN the active screen is the wizard screen
- WHEN `Model.View()` is called
- THEN `View().WindowTitle` MUST equal `bak — Wizard`

#### Scenario: returning to menu shows Main Menu title

- GIVEN the user returns to `ScreenMenu`
- WHEN `Model.View()` is called
- THEN `View().WindowTitle` MUST equal `bak — Main Menu`

#### Scenario: restore shows selected backup id

- GIVEN the active screen is `ScreenRestore` with `SelectedID == "abc1234"`
- WHEN `Model.View()` is called
- THEN `View().WindowTitle` MUST contain `abc1234`

### Requirement: Rotating spinner for running step indicators

The progress and health step lists MUST render the running step with the live `spinner.Model` frame (`m.spinner.View()`) instead of a static literal glyph. Completed steps MUST render a colored `✓`. Pending steps MUST render `○`.

#### Scenario: running step shows live spinner frame

- GIVEN `ProgressModel` has a step in `StepRunning` state and `m.spinner` has been advanced N ticks
- WHEN `View()` is called
- THEN the running-step row MUST contain `m.spinner.View()` output (the current rotated frame)

#### Scenario: complete step shows checkmark

- GIVEN a step has transitioned to `StepDone`
- WHEN `View()` is called
- THEN the step row MUST render `✓` with `ProgressDoneStyle`

#### Scenario: pending step shows circle indicator

- GIVEN a step is in `StepPending`
- WHEN `View()` is called
- THEN the step row MUST render `○`

### Requirement: Persistent status bar

All screens MUST render a one-line status bar at the bottom containing version, active preset, and backup path (truncated to terminal width). The status bar MUST hide when terminal width is below 40 columns. It MUST be rendered by a shared stateless function and styled with package-level lipgloss vars (AGENTS.md §styles).

#### Scenario: status bar visible on all screens

- GIVEN the terminal width is >= 40 columns
- WHEN any screen renders
- THEN the bottom line MUST show version, preset, and backup path segments

#### Scenario: status bar adapts to terminal width

- GIVEN the terminal width is 60 columns and the backup path is longer than 40 characters
- WHEN the status bar renders
- THEN the backup path MUST be truncated with an ellipsis to fit the available width

#### Scenario: status bar hidden on narrow terminals

- GIVEN terminal width is 39 columns
- WHEN any screen renders
- THEN the status bar MUST NOT be rendered

### Requirement: Gradient logo with no-color fallback

The ASCII logo MUST render with a Rose Pine multi-stop vertical gradient (Love → Gold → Rose → Pine → Lavender). On terminals with no color support (lipgloss profile = Ascii), the logo MUST fall back to uncolored plain text. The logo MUST remain hidden when terminal width < 40 (existing behavior preserved).

#### Scenario: gradient logo on color terminal

- GIVEN terminal width >= 40 and color profile supports truecolor or 256-color
- WHEN `RenderLogo(width)` is called
- THEN each logo line MUST be rendered with a distinct Rose Pine foreground color from the gradient stops

#### Scenario: plain logo on no-color terminal

- GIVEN the detected lipgloss color profile is Ascii (no color)
- WHEN `RenderLogo(width)` is called
- THEN the logo MUST render without ANSI color codes (monochrome)

#### Scenario: logo hidden on narrow terminal

- GIVEN terminal width < 40
- WHEN `RenderLogo(width)` is called
- THEN the return MUST be the empty string

### Requirement: Scrollable dry-run viewport

The restore dry-run diff MUST render inside a `bubbles/viewport.Model` instead of dumping raw output to screen. The viewport MUST support scroll up/down via arrow keys, `j`/`k`, and `PgUp`/`PgDn`. Pressing `q` MUST return to the backup list.

#### Scenario: viewport shows diff content

- GIVEN `restoreDryRunResultMsg` arrives with non-empty `output`
- WHEN the restore model transitions to `restoreStateDryRun`
- THEN `viewport.SetContent(output)` MUST have been called
- AND `View()` MUST render `viewport.View()` output

#### Scenario: scroll up and down work

- GIVEN the viewport holds 50 lines and the visible height is 10
- WHEN the user presses `PgDn`
- THEN the viewport's scroll position MUST advance by one page
- WHEN the user presses `PgUp`
- THEN the viewport's scroll position MUST retreat by one page

#### Scenario: long diffs scroll without wrapping the help line off-screen

- GIVEN `DryRunOutput` is 80 lines and terminal height is 24
- WHEN `View()` renders
- THEN the diff region MUST occupy a bounded viewport area and the help line MUST remain visible

#### Scenario: q returns to backup list

- GIVEN the model is in `restoreStateDryRun`
- WHEN the user presses `q`
- THEN the state MUST transition to `restoreStateList`
- AND the viewport content MAY be retained or cleared

### Requirement: Mouse navigation on backup lists

The dashboard and restore list screens MUST enable `tea.MouseModeCellMotion` via the `View().MouseMode` field. Mouse wheel MUST scroll the list, and left click MUST select the clicked row. Mouse events MUST be suppressed when the search field is active.

#### Scenario: wheel scrolls list

- GIVEN dashboard is visible, search is inactive, and `tea.MouseWheelMsg{Button: tea.MouseWheelDown}` arrives
- WHEN `Update` processes the message
- THEN the table cursor MUST advance

#### Scenario: click selects item

- GIVEN dashboard list is visible with 5 rows
- WHEN `tea.MouseClickMsg{Button: tea.MouseLeft, Y: 2}` arrives
- THEN the cursor MUST move to the row at the clicked Y coordinate

#### Scenario: mouse suppressed when search active

- GIVEN dashboard search is active (`m.search.IsActive() == true`)
- WHEN any `tea.MouseMsg` arrives
- THEN `Update` MUST return the model unchanged (mouse not processed)

### Requirement: Styled empty states

Screens with no data (no backups on dashboard/restore, no provider on cloud) MUST render a styled empty state with a Rose Pine icon, an italic message, and a hint with the next action. The empty state MUST use a shared stateless `RenderEmptyState(icon, message, hint)` function and package-level lipgloss styles.

#### Scenario: no backups shows empty state with CTA

- GIVEN dashboard is visible and `len(Backups) == 0`
- WHEN `View()` renders the empty branch
- THEN the output MUST contain a Love-colored `∅` icon
- AND an italic message `No backups yet`
- AND a hint `Run 'bak backup' to create one`

#### Scenario: cloud no-provider shows styled empty state

- GIVEN cloud screen has no configured provider
- WHEN `View()` renders
- THEN the output MUST use `RenderEmptyState` (not a bare string) and MUST include a CTA hint