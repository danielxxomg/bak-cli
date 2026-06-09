# Tasks: Adapter Knowledge Validation

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~160 (test ~120, fixes ~40) |
| 400-line budget risk | Low |
| Chained PRs recommended | No |
| Suggested split | Single PR |
| Delivery strategy | single-pr |
| Chain strategy | size-exception |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: Low

## Phase 1: Foundation — Knowledge Test (RED)

- [ ] 1.1 Export `AdapterName`, `ConfigRelPath`, `CategoryMap` from each adapter package (claudecode, cursor, codex, windsurf, kiro, kilocode, pidev)
- [ ] 1.2 Create `internal/adapters/knowledge_test.go` — table-driven test validating configRelPath and categoryMap against documented values from design.md

## Phase 2: Fix Adapters (GREEN)

- [ ] 2.1 Fix claudecode — add `agents` and `plugins` to CategoryMap
- [ ] 2.2 Fix cursor — add `mcp` (root file) to CategoryMap
- [ ] 2.3 Fix codex — replace `instructions` dir with `agents` (root AGENTS.md)
- [ ] 2.4 Fix windsurf — fix rules subpath to `memories`, add `skills`
- [ ] 2.5 Fix kiro — replace `hooks` with `agents`, `steering`, `specs`
- [ ] 2.6 Fix kilocode — add `workflows` and `skills` to CategoryMap
- [ ] 2.7 Fix pidev — change ConfigRelPath from `.pi` to `.pi/agent`

## Phase 3: Verify

- [x] 3.1 Run `go test ./internal/adapters/...` — all pass
- [x] 3.2 Run `go vet ./internal/adapters/...` — clean
- [x] 3.3 Run `go test -cover ./internal/adapters/...` — ≥80%

## Phase 4: Verify Remediation

- [x] 4.1 Validate `SubPath` and `IsDir` in `CategoryMap` — extended expectedKnowledge with per-category struct, updated test assertions
- [x] 4.2 Replace hardcoded `allAdapters()` with `register.All()` — registry-driven adapter discovery, auto-includes new adapters
- [x] 4.3 Add `TestAdapterKnowledge_RegistryCoverage` — cross-checks registry vs expectedKnowledge/adapterRegistry tables
- [x] 4.4 Run `go test ./...` — 1192 passed
