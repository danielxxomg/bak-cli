# Apply Progress: generic-adapter

**Status**: COMPLETE — All 27/27 tasks done. 11 commits on main.

## Phase 1: Foundation (Complete)

- [x] 1.1 RED — `internal/adapters/generic_test.go` with 24 table-driven tests
- [x] 1.2 GREEN — `internal/adapters/generic.go`: GenericAdapter, CategoryDir, 5 interface methods
- [x] 1.3 VERIFY — 1152/1152 tests pass, go vet clean
- [x] 1.4 COMMIT — `refactor: add GenericAdapter base struct`

## Phase 2: Adapter Migrations (Complete)

- [x] 2.1 Migrate codex — 14/14 tests pass unmodified, 1152/1152 zero regressions
- [x] 2.2 COMMIT — `refactor: migrate codex adapter to GenericAdapter`
- [x] 2.3 Migrate kiro — tests pass, zero regressions
- [x] 2.4 COMMIT — `refactor: migrate kiro adapter to GenericAdapter`
- [x] 2.5 Migrate kilocode — tests pass, zero regressions
- [x] 2.6 COMMIT — `refactor: migrate kilocode adapter to GenericAdapter`
- [x] 2.7 Migrate pidev — tests pass, zero regressions
- [x] 2.8 COMMIT — `refactor: migrate pidev adapter to GenericAdapter`
- [x] 2.9 Migrate windsurf — tests pass, zero regressions
- [x] 2.10 COMMIT — `refactor: migrate windsurf adapter to GenericAdapter`
- [x] 2.11 Migrate cursor — tests pass, zero regressions
- [x] 2.12 COMMIT — `refactor: migrate cursor adapter to GenericAdapter`
- [x] 2.13 Migrate claudecode — tests pass, zero regressions
- [x] 2.14 COMMIT — `refactor: migrate claudecode adapter to GenericAdapter`

## Phase 3: Final Verification (Complete)

- [x] 3.1 `go test ./...` — 1152 passed, 0 failures
- [x] 3.2 `go vet ./...` — clean
- [x] 3.3 `register/register.go` unchanged — still uses `&codex.Adapter{}` etc.
- [x] 3.4 No test files modified (only `generic_test.go` is new)
- [x] 3.5 Net line reduction: 1057 lines removed (target ≥700)

## TDD Cycle Evidence

| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|------|-----------|-------|------------|-----|-------|-------------|----------|
| 1.1 | `adapters/generic_test.go` | Unit | N/A (new) | ✅ Written | ✅ Passed | ✅ 24 cases | ✅ Clean |
| 1.2 | `adapters/generic.go` | Unit | N/A | — | ✅ Passed | ✅ All scenarios | ✅ GGA fixes |
| 2.1–2.13 | Existing adapter tests | Unit | ✅ All pass | ➖ Refactor | ✅ All pass | ✅ Zero changes | ✅ Clean |
