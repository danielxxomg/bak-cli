# Apply Progress: generic-adapter

**Status**: Phase 2 in progress — 2 of 8 commits complete.

## Phase 1: Foundation (Complete)

- [x] 1.1 RED — `internal/adapters/generic_test.go` with 24 table-driven tests
- [x] 1.2 GREEN — `internal/adapters/generic.go`: GenericAdapter, CategoryDir, 5 interface methods
- [x] 1.3 VERIFY — 1152/1152 tests pass, go vet clean
- [x] 1.4 COMMIT — `refactor: add GenericAdapter base struct`

## Phase 2: Adapter Migrations

### 2.1–2.2: Codex (Complete)
- [x] 2.1 Migrate codex — 14/14 tests pass unmodified, 1152/1152 zero regressions
- [x] 2.2 COMMIT — `refactor: migrate codex adapter to GenericAdapter`

### Remaining
- [ ] 2.3–2.14: kiro, kilocode, pidev, windsurf, cursor, claudecode
- [ ] 3.1–3.5: Final verification

## TDD Cycle Evidence

| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|------|-----------|-------|------------|-----|-------|-------------|----------|
| 1.1 | `adapters/generic_test.go` | Unit | N/A (new) | ✅ Written | ✅ Passed | ✅ 24 cases | ✅ Clean |
| 1.2 | `adapters/generic.go` | Unit | N/A | — | ✅ Passed | ✅ All scenarios | ✅ GGA fixes |
| 2.1 | `codex/adapter_test.go` | Unit | ✅ 14/14 | ➖ Approval (refactor) | ✅ 14/14 | ➖ All existing | ✅ Clean |
