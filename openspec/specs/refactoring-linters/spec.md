# refactoring-linters Specification

## Purpose

Enable refactoring linters (maintidx, dupl) in golangci-lint to catch code duplication and maintainability regressions. Documents the deferral of complexity linters (gocognit, funlen, nestif) to a follow-up change.

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

### REQ-RL-003: gocognit/funlen/nestif DEFERRED

The linters `gocognit`, `funlen`, and `nestif` MUST NOT be enabled in this change. They are deferred to a follow-up change (`qa-refactor-analysis`) due to 44 existing violations including 3 architectural SEVERE cases (cognitive complexity ≥70) that require code extraction, not configuration changes.

#### Scenario: linters are NOT enabled in this change

- GIVEN the `.golangci.yml` configuration after this change
- WHEN inspecting the enabled linters list
- THEN `gocognit`, `funlen`, and `nestif` MUST NOT appear in the enabled linters
- AND the proposal documents the deferral rationale
