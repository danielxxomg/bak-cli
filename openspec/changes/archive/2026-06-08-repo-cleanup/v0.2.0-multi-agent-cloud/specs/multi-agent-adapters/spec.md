# Delta for multi-agent-adapters

## Purpose

Adapter implementations for Claude Code, Cursor, Codex, Windsurf, Kiro, KiloCode, and pi.dev.

## Requirements

### Requirement: Adapter interface compliance

Each new agent adapter MUST implement the existing `Adapter` interface.

#### Scenario: Claude Code adapter

- GIVEN Claude Code config exists at `~/.claude-code/`
- WHEN `bak backup` runs
- THEN adapter detected and config backed up with SHA-256 checksums

#### Scenario: Cursor adapter

- GIVEN Cursor config exists at `~/.cursor/`
- WHEN `bak backup` runs
- THEN adapter detected and config backed up

#### Scenario: Agent not installed

- GIVEN agent config directory absent
- WHEN `bak backup` runs
- THEN adapter skipped without error

### Requirement: Multi-agent backup

The system MUST backup all installed agents in a single run.

#### Scenario: Multiple agents installed

- GIVEN OpenCode, Claude Code, and Cursor are installed
- WHEN `bak backup --preset full` runs
- THEN manifest contains all three agents with correct paths

### Requirement: Detection priority

Discovery MUST follow priority order: Claude Code → Cursor → Codex → Windsurf → Kiro → KiloCode → pi.dev.

#### Scenario: Priority order

- GIVEN multiple agents installed
- WHEN manifest generated
- THEN agents listed in priority order

## MODIFIED Requirements

### Requirement: Multi-agent adapter registry

The system MUST have extensible adapter interface with OpenCode first-class. The registry MUST auto-discover all installed agents, not just OpenCode.
(Previously: Registry only discovered OpenCode; other agents were ignored.)

#### Scenario: OpenCode discovery

- GIVEN OpenCode installed
- WHEN adapter queried
- THEN correct config paths returned for host OS

#### Scenario: Graceful skip

- GIVEN unregistered agent config exists
- WHEN backup runs
- THEN config ignored without error

#### Scenario: Auto-discovery

- GIVEN Claude Code and Cursor installed
- WHEN registry initialized
- THEN both adapters registered automatically
