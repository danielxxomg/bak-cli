# Proposal: Cycle A — Verify & Diff Commands

## Intent

Users have no way to verify backup integrity after creation or cloud round-trip, and no way to compare two backups to see what changed between them. `bak verify` gives confidence that a backup is not corrupted. `bak diff` answers "what changed between my last backup and this one?" — critical before deleting old backups or auditing config drift.

## Scope

### In Scope
- `bak verify <id>` — SHA-256 checksum verification via existing `manifest.Validate()`
- `bak diff <id1> <id2>` — file-level comparison between two backups (added/removed/modified/unchanged)
- `internal/diff` package — new package for backup-vs-backup comparison logic
- Shared backup ID resolution and path traversal prevention (reuse `restore.go` pattern)

### Out of Scope
- Content-level unified diff output (line-by-line) — deferred to Cycle B
- Diff between backup and current disk state (already covered by `restore --dry-run`)
- Diff output formats (JSON, machine-readable) — text-only for Cycle A
- Encrypted backup decryption for diff (compare encrypted files as-is)

## Capabilities

### New Capabilities
- `backup-verify`: verify backup integrity by re-hashing all files against manifest checksums, with clear pass/fail output
- `backup-diff`: compare two backup manifests to identify added, removed, modified, and unchanged files by canonical SourcePath

### Modified Capabilities
None — no existing spec-level behavior changes.

## Approach

**verify:** Thin wrapper in `cmd/verify.go`. Resolves backup dir via `backup.BakDir()` + ID, loads manifest, calls `m.Validate(backupDir)`. Exit 0 on success, exit 1 on first hash mismatch. Output: `✓ backup <id> verified (N files)` or `✗ verification failed: <details>`.

**diff:** New `internal/diff` package. `DiffEntry` struct: `{SourcePath, Category, Adapter}`. `Category` enum: `Added`, `Removed`, `Modified`, `Unchanged`. Algorithm: flatten both manifests into `map[canonicalPath]Item`, iterate union of keys, compare `Hash` field. Path normalization: `path.Clean(filepath.ToSlash(item.SourcePath))` for canonical comparison.

**Shared:** Extract backup ID resolution + path traversal check into `internal/backup/resolve.go` (`ResolveBackupDir(id) → (dir, error)`). Both `verify` and `diff` commands use this. Reuses the `path.Clean` + `strings.HasPrefix` pattern from `cmd/restore.go`.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `cmd/verify.go` | New | `bak verify` command, thin wrapper around Validate |
| `cmd/diff.go` | New | `bak diff` command, output formatting |
| `internal/diff/` | New | DiffEntry, Compare() logic, path normalization |
| `internal/backup/resolve.go` | New | Shared backup ID resolution + traversal guard |
| `cmd/restore.go` | Modified | Refactor to use shared `backup.ResolveBackupDir()` |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Path normalization mismatch across OS | Low | `path.Clean` + `filepath.ToSlash` per AGENTS.md; table-driven tests with Win/Mac/Linux paths |
| Large manifests slow diff | Low | O(n) map-based comparison; typical backup <500 files |
| Encrypted backup diff shows all files as modified | Med | Document behavior; encrypted files compared by hash (correct — detects corruption) |

## Rollback Plan

1. New commands are additive — removing `cmd/verify.go`, `cmd/diff.go`, `internal/diff/` reverts functionality
2. `internal/backup/resolve.go` extraction: revert `cmd/restore.go` to inline resolution if shared helper causes issues
3. No config or manifest schema changes — zero migration risk

## Dependencies

- None — all logic uses existing stdlib and `internal/manifest` package

## Success Criteria

- [ ] `bak verify <id>` passes on valid backup, fails with clear error on corrupted file
- [ ] `bak diff <id1> <id2>` correctly categorizes files as added/removed/modified/unchanged
- [ ] Path traversal blocked: `bak verify ../etc` returns error
- [ ] Cross-platform path normalization tested (Windows ↔ Linux paths)
- [ ] `go test ./...` passes with >80% coverage on `internal/diff` and new cmd code
