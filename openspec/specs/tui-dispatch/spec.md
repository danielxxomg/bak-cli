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

### Requirement: Model.Update submodel dispatch

`Model.Update` MUST use a `subModel` interface and a `map[screen]subModel` for routing key events to the active screen's model. The map MUST be lazily initialized on first screen change.

#### Scenario: key event routed to correct sub-model

- GIVEN current screen is dashboard
- WHEN a key event arrives at `Model.Update`
- THEN the event is forwarded to the dashboard sub-model
- AND the dashboard sub-model returns its updated state

#### Scenario: unknown screen does not panic

- GIVEN `m.subs` map does not contain the current screen
- WHEN a key event arrives
- THEN the event is handled gracefully without panic
- AND the model returns unchanged state

#### Scenario: lazy-init populates map on screen change

- GIVEN `m.subs` is nil at startup
- WHEN a `screenChange` message sets the active screen
- THEN `m.subs` is initialized with all known screen sub-models

#### Scenario: WindowSizeMsg routed to active sub-model

- GIVEN a `tea.WindowSizeMsg` arrives
- WHEN `Model.Update` processes it
- THEN the message is forwarded to the active sub-model
- AND the sub-model stores width and height

#### Scenario: ProgressStepMsg handled directly

- GIVEN a `ProgressStepMsg` arrives
- WHEN `Model.Update` processes it
- THEN the message is handled directly by `Model.Update` (not forwarded)
- AND the progress state is updated

### Requirement: Screen extraction

Each TUI screen MUST have its own file in `internal/tui/screens/`. Screen models MUST implement the `subModel` interface (`Init()`, `Update()`, `View()`).

#### Scenario: dashboard has own file

- GIVEN `internal/tui/screens/dashboard.go` exists
- WHEN the TUI displays the dashboard screen
- THEN `DashboardModel.Update` and `DashboardModel.View` handle all dashboard logic

#### Scenario: health has own file

- GIVEN `internal/tui/screens/health.go` exists
- WHEN the TUI displays the health screen
- THEN `HealthModel.Update` and `HealthModel.View` handle all health logic

#### Scenario: screen models implement subModel interface

- GIVEN each screen model in `internal/tui/screens/`
- WHEN compiled
- THEN each model satisfies the `subModel` interface

### Requirement: Render consolidation

The `Model.View` method MUST delegate rendering to a `renderScreen` helper function. `Model.View` MUST NOT contain inline screen rendering logic.

#### Scenario: View delegates to renderScreen

- GIVEN `Model.View` is called
- WHEN the model renders the current screen
- THEN rendering logic is delegated to `renderScreen`
- AND `Model.View` contains only dispatch logic

#### Scenario: renderScreen handles all screen types

- GIVEN `renderScreen` receives a screen type and model state
- WHEN rendering any screen
- THEN the correct screen-specific render function is called

### Requirement: styles.RenderTooSmall

The "Terminal too small" message MUST be rendered by a shared `styles.RenderTooSmall(width, height int) string` function. No screen file may contain an inline "Terminal too small" string literal.

#### Scenario: RenderTooSmall produces correct message

- GIVEN `styles.RenderTooSmall(15, 5)` is called
- WHEN the function executes
- THEN the returned string contains "Terminal too small (15x5)"

#### Scenario: all too-small guards use shared helper

- GIVEN all screen files in `internal/tui/screens/`
- WHEN `grep "Terminal too small" internal/tui/` is executed
- THEN the literal appears only in `styles/` package
- AND all screen files call `styles.RenderTooSmall`
