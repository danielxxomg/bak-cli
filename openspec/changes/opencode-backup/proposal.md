# Proposal: bak — AI Coding Config Backup CLI

## Intent

No tool backs up AI coding configs holistically. **bak** is a Go CLI that packs, restores, and syncs configs for OpenCode and future AI coding agents across machines with safety guarantees.

## Scope

### In Scope
- Backup with presets: `quick` (configs), `full` (everything), `skills` (skills only)
- Restore with mandatory dry-run diff before applying
- GitHub Gist cloud sync (private gists, token auth)
- Cross-platform path normalization (Windows ↔ macOS ↔ Linux)
- Interactive TUI picker (`bak pick`) via bubbletea
- Adapter architecture — OpenCode first-class, extensible to other agents
- Secrets exclusion by default + `.env.example` template generation

### Out of Scope
- Encryption at rest (v2), non-OpenCode adapters (v2), session/auth backup, GUI

## Capabilities

### New Capabilities
- `backup-engine`: Preset-based backup, selective categories, secrets exclusion, `.env.example` generation
- `restore-engine`: Restore from ID/path, mandatory dry-run diff, overwrite semantics
- `cloud-sync`: GitHub Gist push/pull, token-based auth
- `path-normalization`: Canonical path storage, OS detection, cross-platform translation
- `agent-adapters`: Adapter interface + registry, OpenCode first adapter, extensible
- `interactive-picker`: Bubbletea TUI checkboxes for selective backup
- `manifest`: Directory format with `manifest.json` (metadata, checksums, OS source), `bak export` → tar.gz

### Modified Capabilities
None (greenfield)

## Approach

- **Stack**: Go 1.24+, cobra, bubbletea/lipgloss, stdlib tar/gzip, goreleaser
- **Architecture**: Adapter pattern per agent → registry → backup engine → manifest → output. Restore: manifest → normalize paths → dry-run gate → apply
- **Format**: Directory (`bak/manifest.json` + `bak/opencode/`), git-friendly. `bak export` → tar.gz
- **Restore**: Overwrite (not merge). Dry-run is mandatory gate

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `cmd/` | New | CLI: backup, restore, push, pull, list, pick, export |
| `internal/adapters/` | New | Adapter interface, registry, OpenCode adapter |
| `internal/backup/` | New | Backup engine, preset resolution, manifest creation |
| `internal/restore/` | New | Restore engine, dry-run diff, path normalization |
| `internal/cloud/` | New | GitHub Gist client |
| `internal/paths/` | New | Cross-platform path normalization |
| `internal/presets/` | New | Preset definitions (quick, full, skills) |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Absolute paths in MCP configs break cross-platform restore | High | Path normalization canonicalizes all paths in manifest |
| OpenCode merge semantics complex (array concat, dedup) | Med | v1 overwrite only; merge deferred |
| Config overlap between tools sharing OpenCode dirs | Med | Manifest tracks file hashes; restore warns on conflicts |

## Rollback Plan

Dry-run is mandatory gate. Timestamped history in `~/.bak/history/`. Restore known-good backup from clean machine.

## Dependencies

- Go 1.24+, GitHub PAT (`GITHUB_TOKEN` env var or `bak login`)

## Success Criteria

- [ ] `bak backup` creates valid manifest-backed backup in < 2s
- [ ] `bak restore --dry-run` diff matches actual restore changes
- [ ] Push/pull round-trips through GitHub Gist without data loss
- [ ] Windows backup restores correctly on macOS/Linux (paths normalized)
- [ ] Secrets excluded from all presets by default
