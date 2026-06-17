# Design: Adapter Knowledge Validation

## Architecture

### Decision 1: Table-driven knowledge test

A single table-driven test validates all adapters against expected values. Each adapter row specifies expected `ConfigRelPath` and `CategoryMap`.

**Rationale**: One test catches regressions across all adapters. Easy to extend when adding a new adapter.

### Decision 2: Registry-driven discovery

Replace hardcoded `allAdapters()` with `register.All()`. The registry is the single source of truth for the adapter set.

**Rationale**: Prevents the test from silently missing new adapters. `TestAdapterKnowledge_RegistryCoverage` cross-checks registry vs expectedKnowledge.

## expectedKnowledge Table

| Adapter | ConfigRelPath | CategoryMap |
|---------|--------------|-------------|
| claudecode | `.claude` | `agents`, `plugins`, + base categories |
| cursor | `.cursor` | `mcp` (root file), + base categories |
| codex | `.codex` | `agents` (root AGENTS.md), + base categories |
| windsurf | `.windsurf` | `memories` (rules subpath), `skills`, + base categories |
| kiro | `.kiro` | `agents`, `steering`, `specs`, + base categories |
| kilocode | `.kilocode` | `workflows`, `skills`, + base categories |
| pidev | `.pi/agent` | base categories |

## Fixes Applied

| Adapter | Fix |
|---------|-----|
| claudecode | Added `agents` and `plugins` to CategoryMap |
| cursor | Added `mcp` (root file) to CategoryMap |
| codex | Replaced `instructions` dir with `agents` (root AGENTS.md) |
| windsurf | Fixed rules subpath to `memories`, added `skills` |
| kiro | Replaced `hooks` with `agents`, `steering`, `specs` |
| kilocode | Added `workflows` and `skills` to CategoryMap |
| pidev | Changed ConfigRelPath from `.pi` to `.pi/agent` |

## Files Affected

- `internal/adapters/knowledge_test.go` — new: table-driven knowledge + registry coverage tests
- `internal/adapters/claudecode/claudecode.go` — export identifiers, fix CategoryMap
- `internal/adapters/cursor/cursor.go` — export identifiers, fix CategoryMap
- `internal/adapters/codex/codex.go` — export identifiers, fix CategoryMap
- `internal/adapters/windsurf/windsurf.go` — export identifiers, fix CategoryMap
- `internal/adapters/kiro/kiro.go` — export identifiers, fix CategoryMap
- `internal/adapters/kilocode/kilocode.go` — export identifiers, fix CategoryMap
- `internal/adapters/pidev/pidev.go` — export identifiers, fix ConfigRelPath

## Verification

- `go test ./internal/adapters/...` — all pass
- `go vet ./internal/adapters/...` — clean
- `go test -cover ./internal/adapters/...` — ≥80%
