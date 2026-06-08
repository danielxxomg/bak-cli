# Design: repo-cleanup-docs

## Technical Approach

Pure docs/config/file hygiene — zero code logic changes. Four independent batches, each a separate conventional commit. All verification via grep/filesystem checks (no Go tests needed).

## Architecture Decisions

### Decision: Batch Ordering

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Files first, then docs | Fast but docs reference stale paths | **Rejected** |
| Docs → CHANGELOG → openspec → files | Docs reference correct paths before file moves | **Chosen** |
| All in one commit | Simpler but harder to revert selectively | **Rejected** |

**Rationale**: Docs fixes (Batch 1) correct references to paths/tools that still exist. File cleanup (Batch 4) removes/renames those paths. Ordering ensures no commit breaks internal cross-references.

### Decision: CHANGELOG Restructure Strategy

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Move all [Unreleased] into one version | Loses granularity | **Rejected** |
| Split [Unreleased] by feature affinity into v1.0.0/v1.1.0/etc. | Preserves semantic meaning | **Chosen** |
| Rename [Unreleased] to v1.0.0 wholesale | May group unrelated features | **Rejected** |

**Rationale**: The [Unreleased] section contains feature groups (plugin system, scheduling, wizard, verify/diff) that map to distinct roadmap versions. Split by feature affinity into versioned sections. Also fix [0.3.0] which contains a misplaced second `### Added` block (multi-agent/cloud — should be [0.2.0]).

### Decision: Archive Naming Convention

| Option | Tradeoff | Decision |
|--------|----------|----------|
| `archive/{change-name}/` | Loses date context | **Rejected** |
| `archive/YYYY-MM-DD-{change-name}/` | Consistent with existing archives | **Chosen** |

**Rationale**: Existing archives use `2026-06-04-*`, `2026-06-06-*`, `2026-06-07-*` pattern. Use `2026-06-08-repo-cleanup` for consistency.

## Data Flow

```
Batch 1: Edit .md files in-place (no moves)
Batch 2: Restructure CHANGELOG.md sections (in-place)
Batch 3: Move dirs → archive/, fix config.yaml
Batch 4: git mv, rm -rf scripts/, edit .gga + custom.yaml
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `SECURITY.md` L54 | Modify | `filepath.ToSlash` → `strings.ReplaceAll(path, "\\", "/")` |
| `SECURITY.md` L110 | Modify | "No encryption at rest" → document AES-256-GCM encryption added in v0.3.0 |
| `CONTRIBUTING.md` L19 | Modify | `Go 1.24+` → `Go 1.25+` |
| `CONTRIBUTING.md` L56-60 | Modify | `make lint` → `task lint` |
| `CONTRIBUTING.md` L79-85 | Modify | `make test/test-verbose/test-cover` → `task` equivalents |
| `CONTRIBUTING.md` L128-130 | Modify | `make build` → `task build` |
| `CONTRIBUTING.md` L146-151 | Modify | `make release-snapshot/release` → `task` equivalents |
| `CONTRIBUTING.md` L184 | Modify | `filepath.ToSlash` → `strings.ReplaceAll(path, "\\", "/")` |
| `CONTRIBUTING.md` L308 | Modify | `make all` → `task all` |
| `CONTRIBUTING.md` L357-359 | Modify | `make all/test-cover/lint` → `task` equivalents |
| `CONTRIBUTING.md` L423 | Modify | Remove `scripts/` line from project structure tree |
| `CONTRIBUTING.md` L425 | Modify | `Makefile` → `Taskfile.yml` |
| `README.md` L402 | Modify | `Makefile` → `Taskfile.yml` in directory tree |
| `README.md` L506-525 | Modify | Update roadmap: mark completed items, fix version labels |
| `CHANGELOG.md` | Modify | Split [Unreleased] into versioned sections; fix misplaced [0.3.0] content → [0.2.0] |
| `openspec/config.yaml` L10 | Modify | `Go 1.26+` → `Go 1.25+` |
| `openspec/changes/v0.2.0-multi-agent-cloud/` | Move | → `archive/2026-06-08-repo-cleanup/v0.2.0-multi-agent-cloud/` |
| `openspec/changes/v0.3.0-encryption-profiles/` | Move | → `archive/2026-06-08-repo-cleanup/v0.3.0-encryption-profiles/` |
| `openspec/changes/cycle-a-diff-verify/` | Move | → `archive/2026-06-08-repo-cleanup/cycle-a-diff-verify/` |
| `openspec/changes/cycle-c-plugins-coverage/` | Move | → `archive/2026-06-08-repo-cleanup/cycle-c-plugins-coverage/` |
| `openspec/verify-report-v1.1.0.md` | Move | → `archive/2026-06-08-repo-cleanup/` |
| `openspec/verify-report-v1.1.0-final.md` | Move | → `archive/2026-06-08-repo-cleanup/` |
| `scripts/` | Delete | Empty directory (0 files) |
| `examples/presets/custom.yaml` L12 | Modify | `name: my-full` → `name: custom` |
| `.gga` L11 | Modify | Remove `*_test.go` from EXCLUDE_PATTERNS |
| `cmd/coverage_test.go` | Rename | → `cmd/wiring_test.go` (git mv) |

## Interfaces / Contracts

No new interfaces. This is a docs/config/file-only change.

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Grep checks | No `filepath.ToSlash` in `*.md` | `grep -r "filepath.ToSlash" --include="*.md" .` → 0 matches |
| Grep checks | No `make ` in CONTRIBUTING.md | `grep "make " CONTRIBUTING.md` → 0 matches |
| Grep checks | Go version consistency (1.25) | `grep -r "1\.2[0-46]" --include="*.md" --include="*.yaml" .` → 0 matches |
| Grep checks | CHANGELOG has versioned sections | `grep "^\## \[" CHANGELOG.md` → includes v0.2.0, v1.0.0+ |
| File check | `scripts/` does not exist | `test ! -d scripts/` |
| File check | `cmd/wiring_test.go` exists | `test -f cmd/wiring_test.go` |
| File check | `cmd/coverage_test.go` absent | `test ! -f cmd/coverage_test.go` |
| Grep checks | `.gga` has no `*_test.go` exclusion | `grep "_test.go" .gga` → 0 matches |
| Grep checks | `custom.yaml` name field = `custom` | `grep "name: custom" examples/presets/custom.yaml` → 1 match |
| Dir check | Only `archive/` and `repo-cleanup-docs/` in changes/ | `ls openspec/changes/` → 2 entries |

No Go unit tests — this change has zero code logic modifications.

## Migration / Rollout

No migration required. All changes are file edits, moves, and renames. Each batch is a separate commit and can be reverted independently via `git revert`.

## Open Questions

None.
