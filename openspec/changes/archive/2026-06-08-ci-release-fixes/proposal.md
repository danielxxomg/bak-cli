# Proposal: ci-release-fixes

## Intent

CI pipeline has Go version mismatches and cross-platform binary naming bugs that cause build failures on non-Windows runners. Three jobs (`security`, `goreleaser`, `build`) pin Go 1.24 while `go.mod` requires 1.25. The build verification step runs `./bak.exe` on Unix where the binary is named `bak`. The Taskfile hardcodes `bak.exe` as the binary name, breaking local builds on Linux/macOS.

## Scope

### In Scope
- Fix Go version in `security`, `goreleaser`, and `build` CI jobs to `1.25`
- Fix binary verification step to use `./bak` on Unix runners
- Fix Taskfile.yml to use OS-appropriate binary name

### Out of Scope
- GoReleaser release configuration (`.goreleaser.yaml`)
- Adding new CI jobs or runners
- Changing build flags or ldflags

## Capabilities

### New Capabilities
- `ci-consistency`: CI Go version alignment and cross-platform binary naming correctness

### Modified Capabilities
None

## Approach

Bump all `go-version` values in `ci.yml` from `'1.24'` to `'1.25'` to match `go.mod`. Fix the Unix verify step to run `./bak version` instead of `./bak.exe version`. Make Taskfile binary name conditional using Go's `GOOS` or Taskfile's OS variable.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `.github/workflows/ci.yml` | Modified | Go version in 3 jobs + Unix verify step |
| `Taskfile.yml` | Modified | Binary name variable (line 5) |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| GoReleaser config also pins Go 1.24 | Low | Checked — goreleaser uses CI-provided Go, no separate pin |
| Taskfile conditional syntax unsupported | Low | Taskfile v3 supports `{{.OS}}` variable natively |

## Rollback Plan

Revert the single commit — all changes are in 2 YAML files with no code dependencies.

## Dependencies

None

## Success Criteria

- [ ] All CI jobs use `go-version: '1.25'`
- [ ] CI build verification passes on Ubuntu, macOS, and Windows runners
- [ ] `task build` produces correct binary name on each OS
