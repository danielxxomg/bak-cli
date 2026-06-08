# Delta for agent-adapters

## Purpose

Registry now auto-discovers multiple agents, not just OpenCode.

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

#### Scenario: No agents installed

- GIVEN no supported agents installed
- WHEN `bak backup` runs
- THEN warning printed and empty backup created with zero agents

#### Scenario: Priority order

- GIVEN multiple agents installed
- WHEN backup manifest generated
- THEN agents listed in priority order: Claude Code → Cursor → Codex → Windsurf → Kiro → KiloCode → pi.dev → OpenCode

## ADDED Requirements

### Requirement: Adapter detection path validation

The system MUST validate detected agent paths stay within the user home directory.

#### Scenario: Path traversal prevention

- GIVEN malicious symlink in agent config dir pointing to `/etc/passwd`
- WHEN backup runs
- THEN path rejected and agent skipped with error
