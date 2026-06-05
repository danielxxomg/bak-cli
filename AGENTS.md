# AGENTS.md — bak-cli Code Review Rules

## Project Identity
- **Name**: bak-cli
- **Binary**: `bak`
- **Purpose**: Backup and restore OpenCode AI coding configurations across machines
- **Language**: Go 1.24+
- **Architecture**: CLI (cobra) with adapter pattern

## Code Review Rules

### Go Idioms
- MUST use `fmt.Errorf("context: %w", err)` for error wrapping — never bare `errors.New` for wrapped errors
- MUST use table-driven tests (`[]struct{ name string; ... }`) for unit tests
- MUST use `filepath.Join` for OS-specific paths, `path.Clean` for canonical paths
- SHOULD prefer interfaces over concrete types for testability
- MUST NOT use `panic` in library code — return errors
- MUST handle ALL returned errors — no `_ =` for error returns

### Security
- MUST validate all paths stay under user home directory (path traversal prevention)
- MUST NOT include secrets/API keys/tokens in backup by default
- MUST redact sensitive patterns (ghp_*, sk-*, sk-ant-*) in any output
- MUST use `os.UserHomeDir()` — never hardcode home paths

### Cross-Platform
- MUST handle Windows (`\`), macOS (`/`), and Linux (`/`) path separators
- MUST use `path.Clean` (not `filepath.Clean`) for canonical path normalization
- MUST test path operations on all three OS representations
- MUST NOT assume case-sensitive filesystems

### CLI Patterns
- MUST use cobra for command structure
- MUST provide `--help` for every command
- MUST return exit code 0 on success, 1 on error
- SHOULD provide `--verbose` flag for debugging
- MUST provide `--dry-run` for any destructive operation (restore)

### Testing
- MUST achieve >80% coverage for new code
- MUST test happy path AND error paths
- MUST test edge cases: empty input, missing files, permission errors
- MUST use `t.TempDir()` for test isolation — never write to real filesystem
- SHOULD test cross-platform path behavior

### Backup/Restore Specifics
- MUST create manifest before copying files (fail-fast on invalid state)
- MUST compute SHA-256 checksums for all backed-up files
- MUST verify checksums on restore
- MUST NOT restore without mandatory dry-run diff
- MUST warn on version mismatch between backup and installed tools

### Git Safety
- MUST auto-commit before and after restore operations
- MUST NOT force-push or rewrite history
- `bak undo` MUST use `git revert` (safe, non-destructive)

### Commits
- MUST follow Conventional Commits: `feat:`, `fix:`, `test:`, `chore:`, `docs:`
- MUST keep commits atomic — one logical change per commit
- MUST NOT include AI attribution (Co-Authored-By) in commit messages
