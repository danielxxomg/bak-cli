# Design: Cycle A — Verify & Diff Commands

## Technical Approach

Add two new commands (`bak verify`, `bak diff`) backed by a new `internal/diff` package and a shared `internal/backup.ResolveBackupID` helper. `verify` is a thin wrapper over the existing `manifest.Validate()`. `diff` flattens both manifests into `map[canonicalPath]Item` and compares by SHA-256 hash. Path traversal prevention is extracted from `cmd/restore.go` into the shared helper and reused by all three commands.

## Architecture Decisions

| Decision | Options | Tradeoff | Choice | Rationale |
|---|---|---|---|---|
| Verify error strategy | Fail-fast vs collect-all | Fail-fast = simpler, matches spec "first hash mismatch" | **Fail-fast** on first mismatch | Spec scenario says "exits 1 on the first hash mismatch"; matches existing `manifest.Validate()` semantics |
| Diff algorithm | Map-based O(n) vs nested loop O(n²) | Map = linear, more memory; nested = no allocs | **Map-based** using `map[string]manifest.Item` keyed by canonical SourcePath | Typical backup <500 files; O(n) is trivially fast and simpler to reason about |
| Canonical path normalization | `path.Clean(filepath.ToSlash(p))` vs OS-native | Cross-platform stable vs OS-specific | **`path.Clean(filepath.ToSlash(p))`** | Per AGENTS.md; allows Win↔Linux backup comparison |
| Shared helper location | `internal/backup/resolve.go` vs inline per cmd | Shared = DRY, consistent security; inline = no refactor | **Extract to `internal/backup/resolve.go`** | Same traversal guard needed in 3 commands; single source of truth for security-critical code |
| Diff output ordering | Insertion order vs sorted by path | Sorted = deterministic; insertion = map order (random) | **Sorted by path within each category** | Deterministic output across runs; easier to diff the diff |

## Data Flow

### Verify Flow

```
User ──→ cmd/verify.go ──→ backup.ResolveBackupID(id) ──→ (dir, error)
                                    │
                                    ▼
                          manifest.Load(dir) ──→ *Manifest
                                    │
                                    ▼
                          m.Validate(dir) ──→ nil | error
                                    │
                                    ▼
                          stdout: "✓ verified (N files)"
                          OR stderr: "✗ hash mismatch: <path>"
```

### Diff Flow

```
User ──→ cmd/diff.go ──→ backup.ResolveBackupID(id1) ──→ dirA
                      ──→ backup.ResolveBackupID(id2) ──→ dirB
                             │              │
                             ▼              ▼
                    manifest.Load(dirA)  manifest.Load(dirB)
                             │              │
                             └──────┬───────┘
                                    ▼
                         diff.Compare(mA, mB)
                                    │
                   flatten to map[canonPath]Item (per manifest)
                   union of keys → categorize by presence+hash
                                    │
                                    ▼
                         []DiffEntry (sorted)
                                    │
                                    ▼
                         print grouped by category:
                         Added / Removed / Modified / Unchanged
```

## File Changes

| File | Action | Description |
|---|---|---|
| `internal/diff/diff.go` | Create | `DiffEntry` struct, `Category` type + constants, `Compare(a, b *manifest.Manifest) []DiffEntry` |
| `internal/diff/diff_test.go` | Create | Table-driven tests: all 4 categories, cross-platform paths, identical manifests, empty manifests |
| `internal/backup/resolve.go` | Create | `ResolveBackupID(id string) (dir string, backupsDir string, err error)` — traversal guard + existence check |
| `internal/backup/resolve_test.go` | Create | Table-driven tests: valid ID, missing ID, `../` traversal, nested traversal |
| `cmd/verify.go` | Create | `bak verify <id>` cobra command; calls `ResolveBackupID` + `manifest.Load` + `Validate` |
| `cmd/verify_test.go` | Create | Cobra test: success, corrupted, missing, traversal |
| `cmd/diff.go` | Create | `bak diff <id1> <id2>` cobra command; formats grouped output |
| `cmd/diff_test.go` | Create | Cobra test: all categories, identical, missing, traversal |
| `cmd/restore.go` | Modify | Replace lines 57–73 with call to `backup.ResolveBackupID(backupID)` |

## Interfaces / Contracts

```go
// internal/backup/resolve.go
// ResolveBackupID validates the id, builds the backup dir path,
// enforces path traversal prevention, and checks existence.
func ResolveBackupID(id string) (backupDir string, err error)

// internal/diff/diff.go
type Category string

const (
    CategoryAdded     Category = "Added"
    CategoryRemoved   Category = "Removed"
    CategoryModified  Category = "Modified"
    CategoryUnchanged Category = "Unchanged"
)

// DiffEntry represents one file-level difference between two backups.
type DiffEntry struct {
    SourcePath string   // canonical path (path.Clean + filepath.ToSlash)
    Category   Category
    Adapter    string   // adapter name from the manifest where the item was found
}

// Compare returns the set of differences between manifests a and b.
// Entries are sorted by SourcePath.
func Compare(a, b *manifest.Manifest) []DiffEntry

// canonicalPath normalizes an item's SourcePath for cross-platform matching.
func canonicalPath(p string) string // unexported; path.Clean(filepath.ToSlash(p))
```

## Testing Strategy

| Layer | What | Approach |
|---|---|---|
| Unit | `diff.Compare` — all 4 categories, cross-platform paths, empty manifest, identical manifests | Table-driven in `internal/diff/diff_test.go` |
| Unit | `backup.ResolveBackupID` — valid, missing, traversal (`../`, encoded variants) | Table-driven with `t.TempDir()` |
| Unit | `cmd/verify`, `cmd/diff` — cobra RunE with mocked filesystem | Use `t.TempDir()` to stage fake backups; execute via `rootCmd.SetArgs(...)` |
| Integration | End-to-end verify on real backup + corrupted file | Write real manifest + files, mutate one, assert exit code |
| Coverage | >80% on `internal/diff`, `internal/backup` (resolve.go), new cmd code | `go test -cover ./...` |

## Migration / Rollout

No migration required. All changes are additive new commands + one safe refactor of `cmd/restore.go` to use the shared helper (behavior-preserving).

## Open Questions

- [ ] **Verify non-verbose output**: spec requires "success message including the file count" but doesn't specify per-file output without `--verbose`. Propose: non-verbose = single summary line, verbose = per-file progress. Confirm?
- [ ] **Diff exit code on differences**: spec says "exits 0" when diff runs successfully, even with differences. Confirm diff returns 0 even when Added/Removed/Modified entries exist (i.e., exit code signals error, not delta presence)?
