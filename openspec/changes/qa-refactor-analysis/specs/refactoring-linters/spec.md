# Delta for refactoring-linters

## MODIFIED Requirements

### Requirement: gocognit/funlen/nestif ENABLED (replaces REQ-RL-003 DEFERRED)

The linters `gocognit`, `funlen`, and `nestif` MUST be enabled in `.golangci.yml` with ratcheted thresholds. Test files (`_test.go`) MUST be excluded from all three linters. The linter config MUST be committed in the same PR as the refactors that resolve violations.

(Previously: These three linters were DEFERRED per REQ-RL-003 due to 31 existing violations including 3 SEVERE cases. This change resolves all violations and enables the linters.)

#### Scenario: gocognit enabled at threshold 35

- GIVEN `.golangci.yml` with `gocognit.min-complexity: 35`
- WHEN `golangci-lint run` executes
- THEN functions with cognitive complexity <35 pass
- AND functions with complexity >=35 fail with a gocognit violation

#### Scenario: funlen enabled at lines 80 / statements 50

- GIVEN `.golangci.yml` with `funlen.lines: 80` and `funlen.statements: 50`
- WHEN `golangci-lint run` executes
- THEN functions with <80 lines AND <50 statements pass
- AND functions with >=80 lines OR >=50 statements fail

#### Scenario: nestif enabled at threshold 6

- GIVEN `.golangci.yml` with `nestif.min-complexity: 6`
- WHEN `golangci-lint run` executes
- THEN nesting depth <6 passes
- AND nesting depth >=6 fails

#### Scenario: test files exempt from all three linters

- GIVEN a `_test.go` file with cognitive complexity 50, 90 lines, and nesting depth 10
- WHEN `golangci-lint run` executes
- THEN zero violations are reported for that test file

#### Scenario: zero violations on refactored non-test code

- GIVEN all Tier 1-3 refactors applied
- WHEN `golangci-lint run ./...` executes
- THEN exit code is 0 with zero gocognit/funlen/nestif violations on non-test files

## ADDED Requirements

### Requirement: maintidx nolint removal

The three `//nolint:maintidx` annotations on SEVERE functions (`backup.go:58`, `engine.go:62`, `model.go:127`) MUST be removed. After refactoring, these functions MUST fall below the maintidx threshold without suppression.

#### Scenario: golangci-lint reports 0 maintidx violations without nolint

- GIVEN the three `//nolint:maintidx` comments removed
- WHEN `golangci-lint run` executes
- THEN zero maintidx violations are reported for those functions

### Requirement: Inherent-complexity nolint policy

Functions with inherent domain complexity that cannot be refactored below linter thresholds MUST use explicit `//nolint:<linter> // <reason>` annotations. Only Tier 4 items qualify: `tarGZDir`, `untarGzDir` (gocognit), and `RenderShortcuts` (funlen).

#### Scenario: tarGZDir has nolint with reason

- GIVEN `internal/cloud/pack.go` `tarGZDir` function
- WHEN inspected for linter suppression
- THEN it has `//nolint:gocognit // inherent: tar/gzip walk is fixed algorithm`

#### Scenario: no new nolint annotations beyond Tier 4

- GIVEN the refactored codebase
- WHEN grepping for `//nolint:gocognit`, `//nolint:funlen`, `//nolint:nestif`
- THEN only Tier 4 items have nolint annotations
- AND each annotation includes a reason comment
