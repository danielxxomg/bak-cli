# Proposal: coverage-quick-wins

## Intent

Close coverage gaps on 7 trivially-testable functions that were at 0% coverage. No production code changes — tests only.

## Scope

### In Scope
- Add table-driven tests for pure functions at 0% coverage
- Verify existing coverage for functions that already had tests

### Out of Scope
- Production code changes
- New test infrastructure
- Coverage for inherently-untestable code (bubbletea UI, os.Exit paths)

## Approach

Use existing test patterns from the project (table-driven, t.TempDir, setConfigHome). Strict TDD: RED → GREEN → REFACTOR for each function.

## Results

- 5 of 7 target functions already had tests (discovered during implementation)
- 2 new test functions added: `TestParseCSV` (7 cases), `TestProfileValidateForCreation` (4 cases)
- Total tests: 1113 → 1128
