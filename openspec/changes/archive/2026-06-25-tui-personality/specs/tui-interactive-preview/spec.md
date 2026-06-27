# tui-interactive-preview Specification

## Purpose

Interactive scrollable preview affordance for the restore dry-run diff. Companion to `tui-personality` REQ-TP-005: defines the interactive command surface of the dry-run viewport (key bindings, quit, scroll), while `tui-personality/REQ-TP-005` defines its presence in the restore flow.

## Requirements

### Requirement: Scrollable dry-run preview viewport

The restore command's dry-run output MUST render in an interactive `bubbles/viewport` embedded in the TUI, NOT printed to stdout. The viewport MUST accept `↑`/`↓`, `j`/`k`, `PgUp`/`PgDn`, and `g`/`G` scroll keys, and MUST quit back to the previous screen on `q`. The viewport content MUST be the real diff output produced by `RunRestore(id, true)` (per `restore-flow`).

#### Scenario: viewport initialized with diff content

- GIVEN `restoreDryRunResultMsg{output: "diff --git ..."}` arrives
- WHEN the restore model handles the message
- THEN the embedded `viewport.Model` MUST have its content set to the full diff string
- AND the state MUST transition to `restoreStateDryRun`

#### Scenario: up/down and j/k scroll the viewport

- GIVEN the viewport holds more lines than its visible height
- WHEN the user presses `j` or `↓`
- THEN the viewport MUST scroll down one line
- WHEN the user presses `k` or `↑`
- THEN the viewport MUST scroll up one line

#### Scenario: q returns to previous screen

- GIVEN the model is in `restoreStateDryRun`
- WHEN the user presses `q`
- THEN a `ScreenBackMsg` or transition to `restoreStateList` MUST occur
- AND the viewport MUST NOT print its content to stdout

#### Scenario: content fits terminal without scroll

- GIVEN the diff is 5 lines and the viewport height is 20
- WHEN `View()` renders
- THEN the full diff MUST be visible with no scroll offset
- AND the scroll-percent indicator (if shown) MUST read 100%