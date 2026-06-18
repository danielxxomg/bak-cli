# Exploration: Coverage Improvement Feasibility

## Current State

- **cmd/**: 56.6% coverage (target: 65%) — gap of ~8.4%
- **internal/actions/**: 82.9% coverage (target: 90%) — gap of ~7.1%

Total uncovered statements in cmd/: ~43.4%  
Total uncovered statements in internal/actions/: ~17.1%

## Affected Areas

### cmd/ (56.6% → 65%)

| Category | Functions | Lines (est.) | % of gap | Effort | Recommendation |
|----------|-----------|--------------|----------|--------|----------------|
| **Interactive TUI (bubbletea)** | `runLoginInteractiveWithDeps`, `runPick`, `runPickWithDeps`, `runProfileCreateInteractiveWithDeps` | ~80 | ~25% | High | **SKIP** — Per AGENTS.md: "MUST NOT test bubbletea.Program.Run() directly". These require E2E tests with teatest or manual testing. |
| **os.Exit paths** | `Execute` | ~10 | ~3% | N/A | **SKIP** — Per AGENTS.md: "MUST NOT unit-test os.Exit paths — test via integration/E2E only". |
| **TUI rendering (View methods)** | `renderPresetList`, `renderToggleList`, `renderConfirmSummary` | ~50 | ~15% | Medium | **PARTIAL** — These are pure functions that return strings. Testable via `View()` logic per AGENTS.md, but low business value. |
| **Command wrappers (delegates)** | `runPull`, `runPullWithDeps`, `runPush`, `runPushWithDeps`, `runProfileCreate`, `runProfileList`, `runProfileShow`, `runProfileDelete` | ~60 | ~20% | Low | **TEST** — These are thin wrappers that wire dependencies. Easy to test with mocked deps. |
| **Partially covered functions** | `runBackupWithDeps` (62.5%), `runExportWithDeps` (80%), `runListLocal` (75%), `runLoginWithDeps` (71.4%), `runProfileCreateWithDeps` (41.7%), `runProfileListWithDeps` (75%), `runProfileShowWithDeps` (75%), `runProfileDeleteWithDeps` (75%), `runUndoWithDeps` (33.3%) | ~120 | ~37% | Medium | **TEST** — Error paths and edge cases. Focus on error handling branches. |

**cmd/ Summary:**
- **Testable with unit tests**: ~180 lines (command wrappers + partial coverage gaps) → ~55% of gap
- **Only E2E testable (TUI)**: ~80 lines → ~25% of gap
- **os.Exit paths (skip per AGENTS.md)**: ~10 lines → ~3% of gap
- **TUI rendering (low value)**: ~50 lines → ~15% of gap

**Realistic target for cmd/**: 56.6% → 62-64% (testing wrappers + error paths)  
**Effort**: 4-6 hours  
**Feasibility**: Reaching 65% requires TUI tests, which violates AGENTS.md principles. **65% is NOT sensible** without E2E infrastructure.

---

### internal/actions/ (82.9% → 90%)

| Category | Functions | Lines (est.) | % of gap | Effort | Recommendation |
|----------|-----------|--------------|----------|--------|----------------|
| **Testable with unit tests** | `FormatSizeBytes` (50%), `scanBackupForSecrets` (64.7%), `CreateTarGz` (66.7%), `RunExport` (57.7%), `ProfileShow` (78.9%), `ProfileList` (88.9%), `sched` (66.7%) | ~100 | ~40% | Medium | **TEST** — Pure logic and error paths. Table-driven tests. |
| **OS wrappers (error paths)** | `MkdirAll`, `RemoveAll`, `WalkDir`, `WriteFile`, `CopyFile` (66.7-80%) | ~40 | ~16% | Low | **TEST** — Inject failing filesystem mocks to test error handling. |
| **Interactive wizard** | `ProfileCreateInteractive` (0%) | ~15 | ~6% | High | **SKIP** — Requires wizard mock; low business logic value. |
| **Unused/fallback code** | `defaultConfigLoad` (0%), `UserHomeDir` (0%), `Load` (RealConfigLoader, 0%) | ~20 | ~8% | N/A | **SKIP** — Fallback/bridge code not exercised in normal flow. |
| **Hard-to-trigger error paths** | `Run` (backup 85%, diff 87.9%, pick 62.8%, restore 68.9%, pull 80%, push 82.5%) | ~80 | ~32% | High | **PARTIAL** — Some error paths require specific filesystem states or network failures. |

**internal/actions/ Summary:**
- **Testable with unit tests**: ~100 lines → ~40% of gap
- **Requires mocking (OS errors)**: ~40 lines → ~16% of gap
- **Hard-to-trigger error paths**: ~80 lines → ~32% of gap
- **Interactive/unused code**: ~35 lines → ~14% of gap

**Realistic target for internal/actions/**: 82.9% → 88-90%  
**Effort**: 6-8 hours  
**Feasibility**: 90% is **SENSIBLE** with focused effort on testable logic and error paths.

---

## Approaches

### Approach 1: Partial Coverage Push (Recommended)
**Description**: Focus on high-value, testable code. Skip TUI and os.Exit paths per AGENTS.md.

**cmd/ actions**:
- Test command wrappers (`runPullWithDeps`, `runPushWithDeps`, etc.) — 2 hours
- Test error paths in partially covered functions — 2 hours
- **Result**: 56.6% → ~62-64%

**internal/actions/ actions**:
- Test pure logic functions (`FormatSizeBytes`, `scanBackupForSecrets`, etc.) — 3 hours
- Test OS wrapper error paths with mocks — 2 hours
- Test error paths in `Run` methods — 3 hours
- **Result**: 82.9% → ~88-90%

**Total effort**: 10-12 hours  
**Risk**: Low — follows existing patterns and AGENTS.md rules

---

### Approach 2: Full Coverage Push (Not Recommended)
**Description**: Attempt to reach 65% cmd/ and 90% internal/actions/ including TUI tests.

**Additional work**:
- Build E2E test infrastructure with teatest — 8-12 hours
- Write TUI interaction tests — 6-8 hours
- Test os.Exit paths via integration tests — 4-6 hours

**Total effort**: 28-38 hours  
**Risk**: High — violates AGENTS.md principles, low ROI on TUI tests

---

### Approach 3: Skip cmd/, Focus on internal/actions/
**Description**: Accept cmd/ at current level, push internal/actions/ to 90%.

**Effort**: 6-8 hours  
**Risk**: Low  
**Result**: cmd/ stays at 56.6%, internal/actions/ → 88-90%

---

## Recommendation

**VERDICT: PARTIAL — Sensible for internal/actions/, NOT sensible for cmd/**

### internal/actions/ (82.9% → 90%): ✅ SENSIBLE
- **Effort**: 6-8 hours
- **Risk**: Low
- **Approach**: Focus on pure logic functions, error paths, and OS wrapper mocking
- **Justification**: High business value, follows existing test patterns, meaningful tests

### cmd/ (56.6% → 65%): ❌ NOT SENSIBLE
- **Effort**: 10-12 hours for 62-64%, 28-38 hours for 65%
- **Risk**: High (requires E2E infrastructure)
- **Approach**: Test command wrappers and error paths only (→ 62-64%)
- **Justification**: Reaching 65% requires TUI tests that violate AGENTS.md. The remaining 1-3% is not worth the architectural compromise.

### Overall Recommendation: **PURSUE PARTIAL**

1. **internal/actions/**: Push to 88-90% (6-8 hours) — **HIGH PRIORITY**
2. **cmd/**: Push to 62-64% (4-6 hours) — **MEDIUM PRIORITY**
3. **Accept** that cmd/ will remain below 65% unless E2E infrastructure is built

**Total estimated effort**: 10-14 hours  
**Expected result**: cmd/ → 62-64%, internal/actions/ → 88-90%

---

## Risks

1. **AGENTS.md compliance**: Pushing cmd/ to 65% requires TUI tests that violate "MUST NOT test bubbletea.Program.Run() directly"
2. **Diminishing returns**: Last 5-10% coverage requires disproportionate effort
3. **Test quality vs. quantity**: Risk of writing meaningless tests just to hit numbers
4. **Maintenance burden**: More tests = more maintenance, especially brittle TUI tests

---

## Ready for Proposal

**YES** — but with scoped targets:
- **internal/actions/**: Propose 90% target (sensible, high value)
- **cmd/**: Propose 62-64% target (sensible, respects AGENTS.md)
- **Do NOT propose 65% for cmd/** without E2E infrastructure discussion

The orchestrator should tell the user:
> "Coverage improvement is sensible for internal/actions/ (82.9% → 90%, 6-8 hours). For cmd/, we can realistically reach 62-64% (4-6 hours) without violating AGENTS.md rules. Reaching 65% for cmd/ requires E2E test infrastructure and violates the principle of not testing TUI directly. Recommend pursuing partial targets: 90% for internal/actions/, 62-64% for cmd/."
