# Exploration: repo-cleanup-docs

## Audit Date
2026-06-08

## Summary
Full repo quality audit of bak-cli. Found stale documentation, incorrect references, misnamed files, and openspec housekeeping debt.

## Findings

### Documentation Errors (Security/Correctness)
| File | Line | Issue | Severity |
|------|------|-------|----------|
| SECURITY.md | L54 | Recommends `filepath.ToSlash` — FORBIDDEN by AGENTS.md | High |
| CONTRIBUTING.md | L184 | Same forbidden function reference | High |
| CONTRIBUTING.md | L19 | Says "Go 1.24+" but go.mod requires 1.25 | Medium |
| CONTRIBUTING.md | L110+ | 15+ references to `make` commands; project uses Taskfile.yml | Medium |
| CONTRIBUTING.md | L423 | Mentions `scripts/` directory that doesn't exist | Low |
| README.md | L402 | Architecture diagram shows Makefile (doesn't exist) | Medium |
| README.md | L506-525 | Roadmap shows v0.2.0/v0.3.0 as "current" — we're at v1.3.0 | Medium |
| SECURITY.md | L110 | Says "No encryption at rest" but v0.3.0 added it | High |

### CHANGELOG Issues
- v1.0.0 through v1.3.0 content sits in `[Unreleased]` — never versioned into proper sections.

### openspec Housekeeping
| Item | Issue |
|------|-------|
| config.yaml L10 | Says "Go 1.26+" — doesn't exist, go.mod says 1.25 |
| 4 stale changes | v0.2.0-multi-agent-cloud, v0.3.0-encryption-profiles, cycle-a-diff-verify, cycle-c-plugins-coverage — not archived |
| 2 verify reports | verify-report-v1.1.0.md, verify-report-v1.1.0-final.md floating at openspec root |

### File Cleanup
| Item | Issue |
|------|-------|
| `scripts/` | Empty directory — should be deleted |
| `examples/presets/custom.yaml` | Contains `name: my-full` — filename mismatch |
| `.gga` L11 | Excludes `*_test.go` from review — should include tests |
| `cmd/coverage_test.go` | Misleading name; tests wiring, not coverage — rename to `wiring_test.go` |

## Conclusion
All findings are documentation accuracy, file naming, and openspec housekeeping. No code logic changes required. Safe to batch into a single cleanup change.
