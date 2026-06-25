# Delta for tui-dispatch

## ADDED Requirements

### Requirement: Model.Update submodel dispatch

`Model.Update` MUST use a `subModel` interface (`Update(tea.Msg) (tea.Model, tea.Cmd)`) and a `map[Screen]subModel` for message dispatch instead of per-screen type-assert + reassign forwarding. A `forwardTo(screen Screen, msg tea.Msg) (tea.Model, tea.Cmd, bool)` helper MUST handle the generic forwarding pattern. `Model.handleKey` MUST use the same dispatch map.

(Previously: 21 copies of a 5-line type-assert-reassign block across `Update`, `handleKey`, and `View` — one per screen per message type.)

#### Scenario: key event routed to correct sub-model

- GIVEN the TUI is on the dashboard screen
- WHEN a `tea.KeyPressMsg` arrives
- THEN the message is forwarded to the dashboard sub-model via the dispatch map
- AND the dashboard sub-model's `Update` is called exactly once

#### Scenario: WindowSizeMsg routed to all active sub-models

- GIVEN a `tea.WindowSizeMsg` arrives
- WHEN `Model.Update` processes it
- THEN the active sub-model for the current screen receives the message via `forwardTo`
- AND the model stores updated width/height

#### Scenario: screen-specific messages handled by sub-model

- GIVEN a `ProgressStepMsg` arrives during backup
- WHEN `Model.Update` processes it
- THEN the progress handling logic executes directly (not forwarded to a sub-model)
- AND the progress bar updates

#### Scenario: unknown screen does not panic

- GIVEN `m.screen` is set to an unrecognized value
- WHEN a forwardable message arrives
- THEN `forwardTo` returns `(nil, nil, false)`
- AND `Model.Update` returns `(m, nil)` without panic

#### Scenario: lazy-init populates sub-model map

- GIVEN a `screenChangeMsg` for a screen not yet initialized
- WHEN `Model.Update` processes it
- THEN the sub-model for that screen is created and stored in the map
- AND subsequent messages route to the cached instance

### Requirement: Screen type unexported

The `Screen` type MUST be unexported (`type screen int`). Constants referenced outside the `tui` package MUST remain exported. No `cmd/` package MUST reference the exported `Screen` type.

#### Scenario: grep confirms no cmd/ references to exported Screen type

- GIVEN the refactored codebase
- WHEN `grep -r 'tui\.Screen' cmd/` executes
- THEN zero matches are found

#### Scenario: tui package compiles with unexported screen

- GIVEN `type screen int` in `internal/tui`
- WHEN `go build ./...` executes
- THEN compilation succeeds with zero errors

### Requirement: styles.RenderTooSmall helper

`internal/tui/styles` MUST provide `RenderTooSmall(width, height int) string` that returns the "Terminal too small" message. All duplicate guards in `model.go`, `screens/dashboard.go`, and `screens/health.go` MUST use this helper.

#### Scenario: RenderTooSmall produces correct message

- GIVEN `RenderTooSmall(15, 5)` is called
- WHEN it renders
- THEN it returns a string containing "Terminal too small (15x5)"

#### Scenario: all too-small guards use shared helper

- GIVEN the refactored codebase
- WHEN grepping for "Terminal too small" in `internal/tui/`
- THEN the literal appears only in `styles/` package
- AND all screen files call `styles.RenderTooSmall`
