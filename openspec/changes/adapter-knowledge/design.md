# Design: Adapter Knowledge Validation

## Technical Approach

Add a single `knowledge_test.go` file in `internal/adapters/` containing table-driven tests that assert adapter constants against hardcoded expected values from documentation research. Then fix each adapter's constants to make tests pass (TDD red→green).

## Architecture Decisions

### Decision: Single test file vs per-adapter tests

**Choice**: Single `internal/adapters/knowledge_test.go` with one big table
**Alternatives considered**: One `knowledge_test.go` per adapter sub-package
**Rationale**: The constants (`ConfigRelPath`, `Categories`) are accessible via the `Adapter` interface through `register.All()`. A single table makes it trivial to see all expected values at a glance and ensures no adapter is forgotten. Per-package tests would require exporting constants or adding test-only accessor methods.

### Decision: Test data source — hardcoded vs generated

**Choice**: Hardcoded expected values from documentation research
**Alternatives considered**: Scraping tool docs at test time, loading from YAML fixture
**Rationale**: Tool config layouts change rarely. Hardcoded values are explicit, reviewable, and fail loudly when they drift. YAML fixtures add indirection for no benefit at this scale.

### Decision: Handling tools with dual-scope config (global + project)

**Choice**: Only validate user-level (global) config paths
**Alternatives considered**: Also test project-level config detection
**Rationale**: bak-cli v1 scope is user-level config backup. Project-level configs are repo-specific and out of scope. Tests document this boundary explicitly.

### Decision: How to access adapter internals for testing

**Choice**: Instantiate each adapter via `register.All()`, then type-assert to access `GenericAdapter` fields
**Alternatives considered**: Export constants from each adapter package, add `ConfigRelPath()` method to interface
**Rationale**: The `GenericAdapter` struct fields are already exported. Type-asserting to `*adapters.GenericAdapter` works for 7 of 8 adapters (opencode has its own implementation). For opencode, test its known values directly.

## Data Flow

```
register.All()
    │
    ├── &claudecode.Adapter{} ──→ base.Detect() ──→ base.ConfigRelPath
    ├── &cursor.Adapter{}     ──→ base.Detect() ──→ base.ConfigRelPath
    ├── ...                   ──→ ...
    └── &opencode.Adapter{}   ──→ (custom impl, test separately)

knowledge_test.go:
    TestAdapterKnowledge ──→ table of {name, expectedPath, expectedCats}
        │
        ├── for each registered adapter:
        │       assert adapter.ConfigRelPath == expectedPath
        │       assert adapter.CategoryMap keys ⊇ expectedCats
        └── fail if any adapter missing from table
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/adapters/knowledge_test.go` | Create | Table-driven knowledge validation tests |
| `internal/adapters/claudecode/adapter.go` | Modify | Add `agents`, `plugins` to categoryMap |
| `internal/adapters/cursor/adapter.go` | Modify | Add `mcp` to categoryMap |
| `internal/adapters/codex/adapter.go` | Modify | Replace `instructions` dir with `agents` (root AGENTS.md) |
| `internal/adapters/windsurf/adapter.go` | Modify | Fix `rules` subpath to `memories`, add `skills` |
| `internal/adapters/kiro/adapter.go` | Modify | Replace `hooks` with `agents`, `steering`, `specs` |
| `internal/adapters/kilocode/adapter.go` | Modify | Add `workflows`, `skills` to categoryMap |
| `internal/adapters/pidev/adapter.go` | Modify | Fix configRelPath to `.pi/agent` |

## Interfaces / Contracts

### Knowledge test table structure

```go
type adapterKnowledge struct {
    name         string
    configRelPath string
    categories   map[string]struct{ subPath string; isDir bool }
    source       string // doc URL or "manual verification"
}

var expectedKnowledge = []adapterKnowledge{
    {
        name:         "claude-code",
        configRelPath: ".claude",
        categories: map[string]struct{ subPath string; isDir bool }{
            "config":   {"", false},
            "skills":   {"skills", true},
            "commands": {"commands", true},
            "agents":   {"agents", true},
            "plugins":  {"plugins", true},
        },
        source: "https://docs.anthropic.com/en/docs/claude-code/overview",
    },
    // ... one entry per adapter
}
```

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | Adapter constants match docs | Table-driven test in `knowledge_test.go` |
| Unit | Existing adapter tests still pass | Run `go test ./internal/adapters/...` |
| Integration | Backup/restore with corrected paths | Existing generic_test.go covers mechanism |

## Commit Strategy

Two atomic commits:
1. `test: add adapter knowledge validation tests` — new `knowledge_test.go` (tests fail)
2. `fix: correct adapter constants to match documented config layouts` — fix all adapters (tests pass)

## Open Questions

- [ ] KiloCode global config is at `~/.config/kilo/` but CLI runtime is at `~/.kilocode/`. Which does bak-cli back up? → Decision: `~/.kilocode/` (CLI runtime, consistent with current adapter)
- [ ] Should opencode's custom implementation also be covered by the knowledge test? → Decision: Yes, test its constants directly since it doesn't use GenericAdapter
