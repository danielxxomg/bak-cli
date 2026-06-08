# Tasks: Cycle A — Verify & Diff Commands

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~450–550 (9 files: 4 new source, 4 new test, 1 refactor) |
| 400-line budget risk | Medium |
| Chained PRs recommended | Yes |
| Suggested split | PR 1 (foundation) → PR 2 (commands + refactor) |
| Delivery strategy | ask-on-risk |
| Chain strategy | pending |

Decision needed before apply: Yes
Chained PRs recommended: Yes
Chain strategy: pending
400-line budget risk: Medium

### Suggested Work Units

| Unit | Goal | Likely PR | Notes |
|------|------|-----------|-------|
| 1 | Foundation: resolve helper + diff package with tests | PR 1 | ~270 lines; pure infrastructure, no cmd changes |
| 2 | Commands + refactor: verify, diff, restore refactor, docs | PR 2 | ~250 lines; depends on PR 1; wires everything together |

## Phase 1: Foundation — Shared Resolve Helper

- [x] 1.1 Create `internal/backup/resolve.go` with `ResolveBackupID(id string) (backupDir string, err error)` — uses `BakDir()`, builds `backupsDir`, enforces `path.Clean(filepath.ToSlash())` traversal guard, checks `os.Stat` existence
- [x] 1.2 Create `internal/backup/resolve_test.go` — table-driven: valid ID, missing ID, `../` traversal, nested `../../etc` traversal, using `t.TempDir()` to stage fake backup dirs

## Phase 2: Foundation — Diff Package

- [x] 2.1 Create `internal/diff/diff.go` — define `Category` type with constants (`Added`, `Removed`, `Modified`, `Unchanged`), `DiffEntry` struct (`SourcePath`, `Category`, `Adapter`), unexported `canonicalPath(p string) string` using `path.Clean(filepath.ToSlash(p))`
- [x] 2.2 Add `Compare(a, b *manifest.Manifest) []DiffEntry` to `internal/diff/diff.go` — flatten both manifests into `map[canonicalPath]Item`, iterate union of keys, categorize by presence + hash comparison, sort result by `SourcePath`
- [x] 2.3 Create `internal/diff/diff_test.go` — table-driven: all 4 categories, cross-platform paths (Win `\` vs Linux `/`), identical manifests → all Unchanged, empty manifests → empty result

## Phase 3: Verify Command

- [x] 3.1 Create `cmd/verify.go` — cobra command `bak verify <id>`, calls `backup.ResolveBackupID`, `manifest.Load`, `m.Validate(dir)`; exit 0 + `"✓ backup <id> verified (N files)"` on success; exit 1 on first hash mismatch; `--verbose` flag for per-file progress
- [x] 3.2 Create `cmd/verify_test.go` — test: success path (stage valid backup in `t.TempDir()`), corrupted file (mutate hash), missing backup ID, traversal blocked (`../`)

## Phase 4: Diff Command

- [x] 4.1 Create `cmd/diff.go` — cobra command `bak diff <id1> <id2>`, resolves both IDs via `ResolveBackupID`, loads both manifests, calls `diff.Compare`, prints grouped output under Added/Removed/Modified/Unchanged headings; always exits 0 on success
- [x] 4.2 Create `cmd/diff_test.go` — test: all categories present, identical backups, missing backup ID, traversal blocked

## Phase 5: Refactor restore.go

- [x] 5.1 Modify `cmd/restore.go` — replace lines 57–73 (inline BakDir + traversal guard + existence check) with single call to `backup.ResolveBackupID(backupID)`; verify `go test ./cmd/...` still passes for restore tests

## Phase 6: Verification & Docs

- [x] 6.1 Run `go test -cover ./...` — assert >80% coverage on `internal/diff`, `internal/backup` (resolve.go), and new cmd files
- [x] 6.2 Run `go vet ./...` and `go build ./...` — zero warnings
- [x] 6.3 Update `README.md` — add `bak verify` and `bak diff` usage examples with flags
