# Design: ci-release-fixes

## Decision 1: Go Version — Use `'1.25'` Everywhere

**Context**: Three CI jobs (`security`, `goreleaser`, `build`) pin `go-version: '1.24'` while `go.mod` requires `go 1.25.0`. The `lint`, `test`, and `coverage` jobs already use `'1.25'`.

**Decision**: Set all jobs to `go-version: '1.25'`.

**Alternatives considered**:
- Pin to `'1.24'` everywhere → rejected: contradicts `go.mod`, would fail module resolution
- Use `go-version-file: go.mod` → rejected: `setup-go` action supports this but it's less explicit and harder to override per-job if needed later

**Rationale**: Single source of truth is `go.mod`. All CI jobs must match. `'1.25'` is the correct value today.

## Decision 2: Binary Naming — Conditional in Taskfile, Fix CI Verify Steps

**Context**: `Taskfile.yml` hardcodes `BINARY: bak.exe`. The CI Unix verify step runs `./bak.exe` which doesn't exist on Linux/macOS.

**Decision**:
- **Taskfile**: Use Taskfile's built-in `{{.OS}}` variable to set binary name conditionally:
  ```yaml
  vars:
    BINARY:
      sh: echo '{{if eq .OS "windows"}}bak.exe{{else}}bak{{end}}'
  ```
- **CI verify steps**: Fix the Unix step to run `./bak version` (remove `.exe`). The Windows step already correctly runs `.\bak.exe version`.

**Alternatives considered**:
- Always build as `bak.exe` on all platforms → rejected: non-standard, confusing for Unix users
- Use `go build -o bak` on Unix and `go build -o bak.exe` on Windows in separate Taskfile tasks → rejected: more duplication, the conditional variable approach is cleaner

**Rationale**: Taskfile v3 provides `{{.OS}}` which returns `windows`, `linux`, `darwin`, etc. A single conditional expression handles all platforms. The CI fix is a one-line change in the Unix step.
