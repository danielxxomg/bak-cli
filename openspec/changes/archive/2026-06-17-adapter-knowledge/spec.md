# Spec: Adapter Knowledge Validation

## Requirements

### REQ-AK-001: Export Adapter Identifiers

Each adapter package MUST export `AdapterName` (string), `ConfigRelPath` (string), and `CategoryMap` (map).

**Scenario**: All 7 adapters export required identifiers
- **Given** the adapter packages claudecode, cursor, codex, windsurf, kiro, kilocode, pidev
- **When** importing each package
- **Then** `AdapterName`, `ConfigRelPath`, and `CategoryMap` are accessible exported symbols

### REQ-AK-002: Knowledge Test Validates Against Design

A table-driven test MUST validate each adapter's `ConfigRelPath` and `CategoryMap` against documented expected values from design.md.

**Scenario**: Knowledge test passes with correct adapter metadata
- **Given** expected values documented in design.md
- **When** running `TestAdapterKnowledge` in `internal/adapters/knowledge_test.go`
- **Then** every adapter's `ConfigRelPath` matches the expected path
- **And** every adapter's `CategoryMap` matches the expected categories, `SubPath`, and `IsDir`

### REQ-AK-003: Fix Adapter Discrepancies

Each identified discrepancy MUST be corrected in the adapter source.

**Scenario**: claudecode adapter has correct CategoryMap
- **Given** claudecode adapter
- **When** inspecting `CategoryMap`
- **Then** it includes `agents` and `plugins` entries

**Scenario**: cursor adapter has correct CategoryMap
- **Given** cursor adapter
- **When** inspecting `CategoryMap`
- **Then** it includes `mcp` (root file) entry

**Scenario**: codex adapter has correct CategoryMap
- **Given** codex adapter
- **When** inspecting `CategoryMap`
- **Then** it uses `agents` (root AGENTS.md) instead of `instructions` dir

**Scenario**: windsurf adapter has correct CategoryMap
- **Given** windsurf adapter
- **When** inspecting `CategoryMap`
- **Then** rules subpath is `memories` and `skills` is present

**Scenario**: kiro adapter has correct CategoryMap
- **Given** kiro adapter
- **When** inspecting `CategoryMap`
- **Then** `hooks` is replaced with `agents`, `steering`, `specs`

**Scenario**: kilocode adapter has correct CategoryMap
- **Given** kilocode adapter
- **When** inspecting `CategoryMap`
- **Then** `workflows` and `skills` are present

**Scenario**: pidev adapter has correct ConfigRelPath
- **Given** pidev adapter
- **When** reading `ConfigRelPath`
- **Then** it is `.pi/agent` (not `.pi`)

### REQ-AK-004: Registry-Driven Adapter Discovery

Adapter enumeration MUST use `register.All()` instead of a hardcoded list, so new adapters are auto-included.

**Scenario**: Adapter list is registry-driven
- **Given** a new adapter is added to the registry
- **When** running knowledge tests
- **Then** the new adapter is automatically included in validation via `register.All()`

**Scenario**: Registry coverage is cross-checked
- **Given** `register.All()` returns a set of adapters
- **When** running `TestAdapterKnowledge_RegistryCoverage`
- **Then** every adapter in the registry is present in the expectedKnowledge table

## Dependencies

- `internal/adapters/register/` — provides `All()` for registry-driven discovery
