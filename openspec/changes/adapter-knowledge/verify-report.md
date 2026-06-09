## Verification Report

- Change: `adapter-knowledge`
- Project: `bak-cli`
- Mode: Full artifacts (specs + design + tasks)
- Verdict: **FAIL**

### Completeness

| Dimension | Status | Evidence |
|---|---|---|
| Tasks checked in `tasks.md` | PASS | All tasks 1.1–3.3 are marked complete |
| Runtime tests executed | PASS | `go test ./...` passed; `go test -run TestAdapterKnowledge -v ./internal/adapters` passed |
| Static analysis executed | PASS | `go vet ./...` clean |
| Coverage executed | PASS | `go test -cover ./internal/adapters/...` shows `internal/adapters` 85.9%, adapter subpackages 100%, `opencode` 82.5% |

### Build / Test Evidence

| Command | Result | Evidence |
|---|---|---|
| `go test ./...` | PASS | `1191 passed in 26 packages` |
| `go vet ./...` | PASS | `Go vet: No issues found` |
| `go test -cover ./internal/adapters/...` | PASS | `internal/adapters` 85.9%, `opencode` 82.5%, all other adapter subpackages 100.0% |
| `go test -run TestAdapterKnowledge -v ./internal/adapters` | PASS | `TestAdapterKnowledge_ConfigPaths`, `TestAdapterKnowledge_Categories`, `TestAdapterKnowledge_NoExtraAdapters` all passed |

### Checklist Verification

| Check | Status | Evidence |
|---|---|---|
| `internal/adapters/knowledge_test.go` exists | PASS | File present |
| Tests all 7 target adapters | PASS | Covers claudecode, cursor, codex, windsurf, kiro, kilocode, pidev |
| Claude Code has `agents`, `plugins` | PASS | `internal/adapters/claudecode/adapter.go:17-23` |
| Cursor has `mcp` | PASS | `internal/adapters/cursor/adapter.go:17-21` |
| Codex has `agents` not `instructions` | PASS | `internal/adapters/codex/adapter.go:17-20` |
| Windsurf has `skills`, `memories` | PASS | `internal/adapters/windsurf/adapter.go:17-21` |
| Kiro has `agents`, `steering`, `specs` not `hooks` | PASS | `internal/adapters/kiro/adapter.go:17-22` |
| KiloCode has `workflows`, `skills` | PASS | `internal/adapters/kilocode/adapter.go:17-22` |
| PiDev `ConfigRelPath` is `.pi/agent` | PASS | `internal/adapters/pidev/adapter.go:13-20` |
| Category maps match documented tool categories | PASS | Source inspection matches design/spec for all 7 adapters |

### Spec Compliance Matrix

| Requirement / Scenario | Status | Evidence |
|---|---|---|
| Config path validation | PASS | `TestAdapterKnowledge_ConfigPaths` passed |
| Scenario: all adapters have correct config paths | PASS | Verbose test run passed for all 7 adapters |
| Scenario: new adapter added without knowledge entry fails | FAILING | `knowledge_test.go` does **not** use `register.All()`; `allAdapters()` is hardcoded, so a newly registered adapter can be missed unless test code is also edited |
| Category coverage validation | PASS | `TestAdapterKnowledge_Categories` passed for all 7 adapters |
| Scenario: category keys match documentation | PASS | Verbose test run passed |
| Scenario: missing category detected | PASS | Current test would fail if an expected category key is absent from one of the 7 hardcoded adapters |
| Category subpath validation | FAILING | No test asserts `SubPath` or `IsDir` values in `knowledge_test.go` |
| Scenario: subpath matches real directory layout | FAILING | Missing runtime test coverage for subpaths like Windsurf `rules -> memories` |
| Scenario: root-file category has empty `SubPath` and `IsDir=false` | FAILING | Missing runtime test coverage for root-file categories like Cursor `mcp` and Codex `agents` |

### Correctness Table

| Area | Status | Notes |
|---|---|---|
| Adapter constants in source | PASS | The 7 requested adapters match the documented values |
| Knowledge test coverage of categories | PASS | Category keys are asserted |
| Knowledge test coverage of subpaths / file-vs-dir semantics | FAIL | Spec requires this, but tests do not assert it |
| Detection of newly registered adapters | FAIL | Design/spec expected registry-driven validation; implementation is hardcoded |

### Design Coherence

| Design Decision | Status | Evidence |
|---|---|---|
| Single `knowledge_test.go` file | PASS | Implemented |
| Table-driven validation | PASS | Implemented |
| Instantiate via `register.All()` / runtime registry | WARNING | Design says registry-based coverage; implementation uses direct package constants in `allAdapters()` |
| Validate config paths and category map against documented values | PARTIAL | Paths and keys validated; subpaths / `IsDir` semantics missing |

### CRITICAL

- `internal/adapters/knowledge_test.go` does not verify `CategoryMap` `SubPath` and `IsDir` semantics, so required spec scenarios for directory layout and root-file categories are untested.
- `internal/adapters/knowledge_test.go` does not build its runtime adapter list from `register.All()`. Because `allAdapters()` is hardcoded, the required scenario "new adapter added without knowledge entry" is not actually protected by runtime verification.

### WARNING

- The implementation diverges from the design decision that knowledge validation should be registry-driven. This is not just style — it is why the missing-adapter scenario currently is not enforced.
- The requested golang-pro skill path under `.config/opencode/skills/golang-pro/` does not exist in this environment; verification used the installed skill at `.agents/skills/golang-pro/SKILL.md` instead.

### SUGGESTION

- Extend `expectedKnowledge` to store full expected category metadata (`SubPath`, `IsDir`) and assert exact values, not only key presence.
- Replace `allAdapters()` hardcoding with registry-based enumeration via `register.All()` so new adapter registrations automatically participate in knowledge validation.

### PASS

- All requested command checks passed: `go test ./...`, `go vet ./...`, and adapter coverage.
- All 7 requested adapter source constants currently match the documented values from the spec/design.