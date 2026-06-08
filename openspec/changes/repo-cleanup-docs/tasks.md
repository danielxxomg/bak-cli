# Tasks: repo-cleanup-docs

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 150-200 |
| 400-line budget risk | Low |
| Chained PRs recommended | No |
| Suggested split | Single PR |
| Delivery strategy | single-pr |
| Chain strategy | N/A |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: N/A
400-line budget risk: Low

## Phase 1: Documentation Accuracy

- [ ] 1.1 Edit `SECURITY.md` line ~54: replace `filepath.ToSlash` recommendation with `strings.ReplaceAll(path, "\\", "/")`
- [ ] 1.2 Edit `SECURITY.md` line ~110: replace "No encryption at rest" with documentation of AES-256-GCM encryption added in v0.3.0
- [ ] 1.3 Edit `CONTRIBUTING.md` line ~19: change `Go 1.24+` to `Go 1.25+`
- [ ] 1.4 Edit `CONTRIBUTING.md` lines ~56-60, ~79-85, ~128-130, ~146-151, ~308, ~357-359: replace all `make` commands with `task` equivalents (15+ references)
- [ ] 1.5 Edit `CONTRIBUTING.md` line ~184: replace `filepath.ToSlash` with `strings.ReplaceAll(path, "\\", "/")`
- [ ] 1.6 Edit `CONTRIBUTING.md` line ~423: remove `scripts/` line from project structure tree
- [ ] 1.7 Edit `CONTRIBUTING.md` line ~425: change `Makefile` to `Taskfile.yml`
- [ ] 1.8 Edit `README.md` line ~402: change `Makefile` to `Taskfile.yml` in architecture directory tree
- [ ] 1.9 Edit `README.md` lines ~506-525: update roadmap to mark v0.2.0 and v0.3.0 as released, fix version labels

## Phase 2: CHANGELOG Restructure

- [ ] 2.1 Read current `CHANGELOG.md` and identify feature groups in `[Unreleased]` section
- [ ] 2.2 Split `[Unreleased]` content into versioned sections: `[1.3.0]`, `[1.2.0]`, `[1.1.0]`, `[1.0.0]` with dates from git tags
- [ ] 2.3 Fix misplaced content in `[0.3.0]`: move second `### Added` block (multi-agent backup, cloud providers) to `[0.2.0]` section
- [ ] 2.4 Keep genuinely unreleased items under `[Unreleased]`

## Phase 3: openspec Housekeeping

- [ ] 3.1 Edit `openspec/config.yaml` line 10: change `Go 1.26+` to `Go 1.25+`
- [ ] 3.2 Create archive directory: `openspec/changes/archive/2026-06-08-repo-cleanup/`
- [ ] 3.3 Move `openspec/changes/v0.2.0-multi-agent-cloud/` to `archive/2026-06-08-repo-cleanup/v0.2.0-multi-agent-cloud/`
- [ ] 3.4 Move `openspec/changes/v0.3.0-encryption-profiles/` to `archive/2026-06-08-repo-cleanup/v0.3.0-encryption-profiles/`
- [ ] 3.5 Move `openspec/changes/cycle-a-diff-verify/` to `archive/2026-06-08-repo-cleanup/cycle-a-diff-verify/`
- [ ] 3.6 Move `openspec/changes/cycle-c-plugins-coverage/` to `archive/2026-06-08-repo-cleanup/cycle-c-plugins-coverage/`
- [ ] 3.7 Move `openspec/verify-report-v1.1.0.md` to `archive/2026-06-08-repo-cleanup/`
- [ ] 3.8 Move `openspec/verify-report-v1.1.0-final.md` to `archive/2026-06-08-repo-cleanup/`

## Phase 4: File Cleanup

- [ ] 4.1 Delete empty `scripts/` directory: `Remove-Item -Recurse -Force scripts/`
- [ ] 4.2 Edit `examples/presets/custom.yaml` line ~12: change `name: my-full` to `name: custom`
- [ ] 4.3 Edit `.gga` line 11: remove `*_test.go` from EXCLUDE_PATTERNS
- [ ] 4.4 Rename test file: `git mv cmd/coverage_test.go cmd/wiring_test.go`

## Phase 5: Verification

- [ ] 5.1 Grep check: `grep -r "filepath.ToSlash" --include="*.md" .` returns 0 matches
- [ ] 5.2 Grep check: `grep "make " CONTRIBUTING.md` returns 0 matches
- [ ] 5.3 Grep check: `grep -r "1\.2[0-46]" --include="*.md" --include="*.yaml" .` returns 0 matches (all Go versions are 1.25)
- [ ] 5.4 Grep check: `grep "^\## \[" CHANGELOG.md` includes `[0.2.0]`, `[0.3.0]`, `[1.0.0]`, `[1.1.0]`, `[1.2.0]`, `[1.3.0]`
- [ ] 5.5 File check: `scripts/` directory does not exist
- [ ] 5.6 File check: `cmd/wiring_test.go` exists and `cmd/coverage_test.go` does not
- [ ] 5.7 Grep check: `grep "_test.go" .gga` returns 0 matches
- [ ] 5.8 Grep check: `grep "name: custom" examples/presets/custom.yaml` returns 1 match
- [ ] 5.9 Directory check: `openspec/changes/` contains only `archive/` and `repo-cleanup-docs/`
