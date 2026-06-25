# refactoring-linters Specification

## Purpose

Enable refactoring linters (maintidx, dupl, gocognit, funlen, nestif) in golangci-lint to catch code duplication, maintainability regressions, and cognitive complexity. All violations in non-test code must be resolved through refactoring; only Tier 4 algorithmic functions may carry nolint annotations.

## Requirements

### REQ-RL-001: maintidx Linter Enabled

golangci-lint MUST run the `maintidx` linter with a threshold of under 20 (functions with maintainability index ≥20 are flagged). Test files (`_test.go`) MUST be excluded from this linter.

#### Scenario: function with maintainability index <20 passes

- GIVEN a Go function with maintidx score below 20
- WHEN `golangci-lint run` executes
- THEN no maintidx violation is reported for that function

#### Scenario: function with maintainability index ≥20 fails

- GIVEN a Go function with maintidx score ≥20
- WHEN `golangci-lint run` executes
- THEN golangci-lint reports a lint error for that function

#### Scenario: test files are exempted

- GIVEN a test file (`_test.go`) containing functions with maintidx ≥20
- WHEN `golangci-lint run` executes
- THEN no maintidx violation is reported for test file functions

---

### REQ-RL-002: dupl Linter Enabled

golangci-lint MUST run the `dupl` linter with a token threshold of 80. Test files (`_test.go`) MUST be excluded from this linter.

#### Scenario: code with <80 token duplication passes

- GIVEN two code blocks with fewer than 80 identical tokens
- WHEN `golangci-lint run` executes
- THEN no dupl violation is reported

#### Scenario: code with ≥80 token duplication fails

- GIVEN two code blocks with 80 or more identical tokens
- WHEN `golangci-lint run` executes
- THEN golangci-lint reports a dupl violation listing both locations

#### Scenario: test files are exempted

- GIVEN test files (`_test.go`) containing duplicated code blocks ≥80 tokens
- WHEN `golangci-lint run` executes
- THEN no dupl violation is reported for test file code

---

### REQ-RL-003: gocognit/funlen/nestif ENABLED

The linters `gocognit`, `funlen`, and `nestif` MUST be enabled in `.golangci.yml` with appropriate thresholds. Test files (`_test.go`) MUST be exempt from all three linters.

#### Scenario: gocognit enabled at threshold 35

- GIVEN the golangci-lint configuration
- WHEN `golangci-lint run` is executed
- THEN `gocognit` reports violations for functions with cognitive complexity > 35
- AND test files (`*_test.go`) are exempt

#### Scenario: funlen enabled at 80 statements / 50 lines

- GIVEN the golangci-lint configuration
- WHEN `golangci-lint run` is executed
- THEN `funlen` reports violations for functions exceeding 80 statements or 50 lines
- AND test files are exempt

#### Scenario: nestif enabled at threshold 6

- GIVEN the golangci-lint configuration
- WHEN `golangci-lint run` is executed
- THEN `nestif` reports violations for nesting depth > 6
- AND test files are exempt

#### Scenario: zero violations on refactored non-test code

- GIVEN all refactoring phases complete
- WHEN `golangci-lint run ./...` is executed
- THEN exit code is 0
- AND no gocognit/funlen/nestif violations are reported on non-test code

---

### REQ-RL-004: nolint Annotations Tier 4 Only

After refactoring, only functions with inherent algorithmic complexity (Tier 4) may carry `//nolint` annotations for gocognit/funlen/nestif. Each annotation MUST include a reason comment.

#### Scenario: tarGZDir has nolint with reason

- GIVEN `internal/cloud/pack.go` contains `tarGZDir`
- WHEN the function exceeds gocognit threshold
- THEN a `//nolint:gocognit` annotation with reason comment is present

#### Scenario: untarGzDir has nolint with reason

- GIVEN `internal/cloud/pack.go` contains `untarGzDir`
- WHEN the function exceeds gocognit threshold
- THEN a `//nolint:gocognit` annotation with reason comment is present

#### Scenario: RenderShortcuts has nolint with reason

- GIVEN `internal/tui/screens/shortcuts.go` contains `RenderShortcuts`
- WHEN the function exceeds funlen threshold
- THEN a `//nolint:funlen` annotation with reason comment is present

#### Scenario: no new nolint beyond Tier 4

- GIVEN all refactoring phases complete
- WHEN `grep -r '//nolint:gocognit\|//nolint:funlen\|//nolint:nestif'` is executed
- THEN only Tier 4 items have nolint annotations
- AND each annotation includes a reason comment

---

### REQ-RL-005: maintidx Violations Eliminated

All `maintidx` violations in non-test code MUST be resolved through refactoring. No `//nolint:maintidx` annotations are permitted.

#### Scenario: golangci-lint reports 0 maintidx violations

- GIVEN all refactoring phases complete
- WHEN `golangci-lint run ./...` is executed
- THEN zero `maintidx` violations are reported

#### Scenario: no nolint:maintidx annotations exist

- GIVEN all refactoring phases complete
- WHEN `grep -r '//nolint:maintidx'` is executed on non-test Go files
- THEN zero matches are found

---

### REQ-RL-006: dupl Violations Eliminated

All `dupl` violations in non-test code MUST be resolved through extraction of shared helpers. No `//nolint:dupl` annotations are permitted.

#### Scenario: golangci-lint reports 0 dupl violations

- GIVEN all refactoring phases complete
- WHEN `golangci-lint run ./...` is executed
- THEN zero `dupl` violations are reported

#### Scenario: no nolint:dupl annotations exist

- GIVEN all refactoring phases complete
- WHEN `grep -r '//nolint:dupl'` is executed on non-test Go files
- THEN zero matches are found

---

### REQ-RL-007: gocognit/funlen/nestif Violations Eliminated

All `gocognit`, `funlen`, and `nestif` violations in non-test code (beyond Tier 4 nolints) MUST be resolved through refactoring.

#### Scenario: golangci-lint reports 0 gocognit violations

- GIVEN all refactoring phases complete and gocognit enabled
- WHEN `golangci-lint run ./...` is executed
- THEN zero `gocognit` violations are reported on non-test code (excluding Tier 4 nolints)

#### Scenario: golangci-lint reports 0 funlen violations

- GIVEN all refactoring phases complete and funlen enabled
- WHEN `golangci-lint run ./...` is executed
- THEN zero `funlen` violations are reported on non-test code (excluding Tier 4 nolints)

#### Scenario: golangci-lint reports 0 nestif violations

- GIVEN all refactoring phases complete and nestif enabled
- WHEN `golangci-lint run ./...` is executed
- THEN zero `nestif` violations are reported on non-test code
