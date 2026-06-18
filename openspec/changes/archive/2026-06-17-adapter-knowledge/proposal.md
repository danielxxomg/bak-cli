# Proposal: Adapter Knowledge Validation

## Intent

Validate that each of the 7 adapters (claudecode, cursor, codex, windsurf, kiro, kilocode, pidev) correctly exposes `ConfigRelPath` and `CategoryMap` matching documented design values. Fix discrepancies found and add a registry-driven knowledge test to auto-detect new adapters.

## Scope

### In Scope

- Export `AdapterName`, `ConfigRelPath`, `CategoryMap` from each adapter package
- Table-driven knowledge test validating all adapters against expected values
- Fix 7 adapter discrepancies in CategoryMap and ConfigRelPath
- Registry-driven adapter discovery (`register.All()`) to auto-include new adapters
- Verify build, tests, vet pass with ≥80% coverage

### Out of Scope

- Adding new adapters
- Changing adapter behavior beyond config metadata

## Capabilities

### New Capabilities

- Adapter knowledge validation via `internal/adapters/knowledge_test.go`

### Modified Capabilities

- 7 adapter packages: corrected `CategoryMap` entries and `ConfigRelPath` values

## Approach

1. Export identifiers from all 7 adapters
2. Create knowledge test with expected values
3. Fix discrepancies found by the test
4. Replace hardcoded `allAdapters()` with `register.All()` for registry-driven discovery
5. Add `TestAdapterKnowledge_RegistryCoverage` cross-check

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Test fails on new adapters not in expected table | Low | RegistryCoverage test catches this — maintainer adds row |

## Success Criteria

- [ ] All 13 tasks complete
- [ ] `go test ./internal/adapters/...` passes
- [ ] `go vet ./internal/adapters/...` clean
- [ ] Coverage ≥80% for `internal/adapters/`
