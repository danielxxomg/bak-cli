# Delta for engine-test-fix

## ADDED Requirements

### REQ-TEST-001: engine_test.go MUST use table-driven tests
**Priority**: must

All test functions in `internal/backup/engine_test.go` MUST use the table-driven test pattern.

**Scenario**: Test function structure
- GIVEN `internal/backup/engine_test.go`
- WHEN test functions are defined
- THEN each test MUST use `tests := []struct{ name string; ... }{...}` pattern
- AND iterate with `for _, tt := range tests { t.Run(tt.name, func(t *testing.T) { ... }) }`

**Acceptance criteria**:
- [ ] All 14 test functions converted to table-driven pattern
- [ ] Each test case struct has a `name string` field
- [ ] `t.Run(tt.name, ...)` wraps each sub-test
- [ ] `t.Helper()` used in shared setup if applicable

**Scenario**: Error path preservation
- GIVEN existing tests cover error paths (missing files, permission errors, invalid manifests)
- WHEN tests are refactored to table-driven
- THEN all error path test cases MUST be preserved as named sub-tests

**Acceptance criteria**:
- [ ] No test cases removed during refactor
- [ ] Error assertions remain identical

---

### REQ-TEST-002: Test coverage MUST NOT decrease
**Priority**: must

The refactor from individual test functions to table-driven MUST NOT reduce code coverage.

**Scenario**: Coverage before and after
- GIVEN `internal/backup/engine_test.go` pre-refactor coverage baseline
- WHEN tests are refactored to table-driven
- THEN `go test -cover ./internal/backup/` MUST report coverage >= pre-refactor baseline

**Acceptance criteria**:
- [ ] Coverage percentage for `internal/backup/` is >= pre-refactor value
- [ ] All existing assertions preserved in table-driven sub-tests
- [ ] Both happy path and error path cases present in table
