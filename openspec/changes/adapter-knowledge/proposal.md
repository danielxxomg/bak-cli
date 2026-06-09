# Proposal: Adapter Knowledge Validation

## Intent

Tests prove mechanism (file copy, hash, walk) but NOT domain knowledge. No test validates that each adapter's `configRelPath` matches the REAL tool config path, or that `categoryMap` covers all documented categories. Research found concrete errors: Kiro has wrong path (uses `~/.kiro` but config is project-level `.kiro/`), Claude Code misses `agents/` and `plugins/`, Cursor misses `mcp.json`, Windsurf misses `skills/` and `global_workflows/`, Codex `instructions/` dir doesn't exist (uses `AGENTS.md` at root), KiloCode global path is `~/.config/kilo/` not `~/.kilocode/`, PiDev uses `.pi/agent/` not `.pi/`.

## Scope

### In Scope
- Knowledge validation tests: table-driven test proving each adapter's constants match documented values
- Fix incorrect adapters: kiro, claude-code, cursor, codex, windsurf, kilocode, pidev
- Each fix includes updating `configRelPath` and/or `categoryMap` to match research

### Out of Scope
- Project-level config backup (only user-level `~/` configs in scope for bak-cli v1)
- YAML adapter changes
- New adapter additions

## Capabilities

### New Capabilities
- `adapter-knowledge`: Tests validating each adapter's configRelPath and categoryMap against documented tool config structures

### Modified Capabilities
- `agent-adapters`: Adapter constants (configRelPath, categoryMap) corrected to match real tool config layouts

## Approach

Single table-driven test file `internal/adapters/knowledge_test.go` with expected values hardcoded from documentation research. Each adapter gets a test case asserting `configRelPath` and full `categoryMap` key set. Fix adapters to match, then tests pass.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/adapters/knowledge_test.go` | New | Table-driven knowledge validation tests |
| `internal/adapters/claudecode/adapter.go` | Modified | Add `agents/`, `plugins/` categories |
| `internal/adapters/cursor/adapter.go` | Modified | Add `mcp` category (root file) |
| `internal/adapters/codex/adapter.go` | Modified | Replace `instructions/` dir with root-file `AGENTS.md` handling |
| `internal/adapters/windsurf/adapter.go` | Modified | Add `skills/`, fix `rules` → `memories/` subpath |
| `internal/adapters/kiro/adapter.go` | Modified | Replace `hooks` with `agents/`, `steering/`, `specs/` |
| `internal/adapters/kilocode/adapter.go` | Modified | Fix path, add `workflows/`, `skills/` |
| `internal/adapters/pidev/adapter.go` | Modified | Fix path `.pi` → `.pi/agent`, fix `agents` category |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Tool docs outdated or incomplete | Med | Mark uncertain values with comments; tests document source |
| Kiro/KiloCode dual-scope (global+project) confusing | Med | bak-cli only backs up global (user-home) configs per v1 scope |
| Fixing adapters breaks existing backups | Low | Category additions are additive; path fixes need migration note |

## Rollback Plan

Revert adapter constant changes via git. Knowledge tests are new files — delete them. No data migration needed since tests are additive.

## Dependencies

- None external — uses only stdlib `testing`

## Success Criteria

- [ ] `go test ./internal/adapters/...` passes with knowledge tests
- [ ] Every adapter's `configRelPath` matches documented path
- [ ] Every adapter's `categoryMap` covers all documented backup-worthy categories
- [ ] Coverage ≥80% for `internal/adapters/` package
