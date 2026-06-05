# Tasks: bak — AI Coding Config Backup CLI

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 2800–3600 (full project) |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | PR 1 → PR 2 → PR 3 → PR 4 → PR 5 → PR 6 |
| Delivery strategy | ask-on-risk |
| Chain strategy | feature-branch-chain |

Decision needed before apply: Resolved
Chained PRs recommended: Yes
Chain strategy: feature-branch-chain (PR #1 of 6 — Foundation)
400-line budget risk: High (mitigated by chained PRs)

### Suggested Work Units

| Unit | Goal | Likely PR | Notes |
|------|------|-----------|-------|
| 1 | Foundation: skeleton, adapter interface, manifest, paths | PR 1 | base: main; ~400 lines incl. tests |
| 2 | Backup engine: OpenCode adapter, presets, secrets, `bak backup` | PR 2 | base: PR 1 branch; ~500 lines |
| 3 | Restore engine: dry-run, overwrite, `bak restore` | PR 3 | base: PR 2 branch; ~400 lines |
| 4 | Git safety: auto-commit, undo, `bak undo` | PR 4 | base: PR 3 branch; ~350 lines |
| 5 | Cloud sync: Gist client, push/pull/login | PR 5 | base: PR 4 branch; ~450 lines |
| 6 | TUI + polish: picker, list, export, goreleaser, README | PR 6 | base: PR 5 branch; ~500 lines |

## Phase 1: Foundation

- [x] 1.1 Init Go module: `go mod init github.com/gentle-programming/bak-cli`, add deps (cobra)
- [x] 1.2 Create `main.go` entry point + `cmd/root.go` cobra root with `--verbose` flag + `cmd/version.go`
- [x] 1.3 Create `internal/adapters/adapter.go`: `Adapter` interface (`Name`, `Detect`, `ListItems`, `Backup`, `Restore`) + `Item` struct
- [x] 1.4 Create `internal/adapters/registry.go`: `Registry` with `Register`, `DetectAll`, `Get`, `GetByName`, `All`, `List`
- [x] 1.5 Create `internal/manifest/manifest.go`: `Manifest` struct matching JSON schema, `New`/`Save`/`Load`/`Validate`/`AddAdapter`
- [x] 1.6 Create `internal/manifest/validate.go`: (merged into manifest.go) per-file SHA-256 hash validation
- [x] 1.7 Create `internal/paths/normalize.go`: `ToCanonical`/`FromCanonical`, `IsUnderHome`, `DetectOS`, `ConfigDir`
- [x] 1.8 Create `internal/config/config.go`: `Config` struct, `Load`/`Save`/`Get`/`Set` for `~/.config/bak/config.json`
- [x] 1.9 Write table-driven tests for paths (Win↔Linux↔Mac), manifest round-trip, registry operations, presets

## Phase 2: Backup Engine

- [x] 2.1 Create `internal/presets/presets.go`: `quick`, `full`, `skills` preset definitions mapping to category lists _(done in Phase 1)_
- [x] 2.2 Create `internal/backup/secrets.go`: regex-based secret detection (`API_KEY=`, `TOKEN=`, etc.), `ScanFile` + `GenerateEnvExample`
- [x] 2.3 Create `internal/adapters/opencode/adapter.go`: implement `Adapter` for OpenCode — detect `~/.config/opencode/`, list items by category (skills, commands, config, mcp, plugins, agents)
- [x] 2.4 Create `internal/backup/engine.go`: orchestrate adapters → copy files → scan secrets → build manifest → write to `~/.bak/backups/<id>/`
- [x] 2.5 Create `cmd/backup.go`: `bak backup [--preset quick|full|skills]` cobra command wiring engine
- [x] 2.6 Write tests: preset resolution, secret detection patterns, OpenCode adapter with temp dir, backup engine round-trip, integration test

## Phase 3: Restore Engine

- [x] 3.1 Create `internal/restore/paths.go`: resolve manifest canonical paths to target OS absolute paths, security check (refuse paths outside home)
- [x] 3.2 Create `internal/restore/dryrun.go`: compute diff between backup items and existing target files, format unified diff output
- [x] 3.3 Create `internal/restore/engine.go`: read manifest → normalize paths → dry-run gate → overwrite files → report results
- [x] 3.4 Create `cmd/restore.go`: `bak restore [--dry-run] <id>` cobra command
- [x] 3.5 Write tests: cross-platform path resolution, dry-run diff accuracy, restore overwrite correctness, security path check

## Phase 4: Git Safety

- [x] 4.1 Create `internal/git/repo.go`: go-git wrapper — `Init(path)`, `Open(path)`, `IsRepo(path)`
- [x] 4.2 Create `internal/git/commit.go`: `AutoCommit(repo, message)` — stage all + commit with timestamp
- [x] 4.3 Create `internal/git/undo.go`: `RevertLast(repo)` — revert HEAD commit
- [x] 4.4 Integrate git safety into restore engine: pre-restore auto-commit, post-restore auto-commit
- [x] 4.5 Create `cmd/undo.go`: `bak undo` — thin wrapper over `git.RevertLast` on `~/.bak/`
- [x] 4.6 Write tests: init repo, auto-commit, undo reverts, restore creates two commits

## Phase 5: Cloud Sync

- [ ] 5.1 Create `internal/cloud/auth.go`: token management — read `GITHUB_TOKEN` env or `~/.bak/config.json`, `bak login` flow
- [ ] 5.2 Create `internal/cloud/gist.go`: GitHub Gist REST client — `Create`, `Update`, `Get`, `List` (private gists, HTTPS only)
- [ ] 5.3 Create `cmd/push.go`: `bak push <id>` — serialize backup dir → push to private gist
- [ ] 5.4 Create `cmd/pull.go`: `bak pull [gist-id]` — fetch gist → reconstruct backup dir → register locally
- [ ] 5.5 Create `cmd/login.go`: `bak login` — prompt for GitHub PAT, save to config
- [ ] 5.6 Write tests: gist CRUD with httptest mock server, push/pull round-trip, auth token resolution

## Phase 6: TUI + Polish

- [ ] 6.1 Create `internal/tui/picker.go`: bubbletea model with checkbox list for categories, lipgloss styling
- [ ] 6.2 Create `cmd/pick.go`: `bak pick` — run TUI picker → selected categories → backup
- [ ] 6.3 Create `cmd/list.go`: `bak list` — scan `~/.bak/backups/`, display table (id, date, preset, file count)
- [ ] 6.4 Create `cmd/export.go`: `bak export <id>` — create tar.gz from backup dir using stdlib `archive/tar` + `compress/gzip`
- [ ] 6.5 Create `cmd/version.go`: `bak version` — print version from goreleaser ldflags
- [ ] 6.6 Create `.goreleaser.yaml`: cross-compile Windows/macOS/Linux, ldflags for version
- [ ] 6.7 Create `.golangci.yaml`: enable `govet`, `errcheck`, `staticcheck`, `gofmt`
- [ ] 6.8 Create `Makefile`: `build`, `test`, `lint`, `clean` targets
- [ ] 6.9 Write README.md: install, usage examples, adapter extension guide
