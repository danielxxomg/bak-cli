# Proposal: v2-migration — bubbletea & lipgloss v1 → v2

## Intent

Migrate bubbletea and lipgloss from v1 (`github.com/charmbracelet/*`) to v2 (`charm.land/*/v2`). This is a **mechanical dependency upgrade** — no feature changes, no behavior changes. The v2 migration unblocks the future `tui-overhaul` change by moving to the current upstream API surface.

The tui-overhaul exploration explicitly deferred this: _"Can reconsider v2 migration as a separate change later."_ This is that change.

## Scope

### In Scope
- Update `go.mod` / `go.sum`: replace `github.com/charmbracelet/bubbletea v1.3.10` → `charm.land/bubbletea/v2 v2.0.7`, `github.com/charmbracelet/lipgloss v1.1.0` → `charm.land/lipgloss/v2 v2.0.3`
- Update imports in 6 files (`cmd/pick.go`, `cmd/pick_test.go`, `cmd/wizard.go`, `cmd/wizard_test.go`, `cmd/login.go`, `cmd/profile.go`)
- Adapt `View()` return type: `string` → `tea.View` (wrap with `tea.NewView()`)
- Adapt key handling: `tea.KeyMsg` → `tea.KeyPressMsg`, `msg.Type` switch → `msg.String()` switch in `wizard.go`
- Rewrite test key constructors: `tea.KeyMsg{Type:..., Runes:...}` → `tea.KeyPressMsg{Code:..., Text:...}`
- Run `go mod tidy`, verify `go build`, `go test`, `go vet`, `golangci-lint`

### Out of Scope
- Adding `bubbles/v2` dependency (not currently used — defer to tui-overhaul)
- Adopting v2 declarative View features (AltScreen, MouseMode via `tea.View` struct fields)
- Key release handling (`tea.KeyReleaseMsg`)
- Any visual/UX changes to the TUI
- AGENTS.md rule updates for v2 patterns (defer — see decisions)

## Capabilities

### New Capabilities
None — this is a dependency version bump with API adaptation. No new user-facing or spec-level behavior.

### Modified Capabilities
None — existing capabilities (interactive picker, wizard) behave identically after migration.

## Approach

**Single PR, single logical commit.** All changes are coupled — the code does not compile if imports and go.mod are out of sync. Per AGENTS.md: _"MUST keep commits atomic — one logical change per commit."_ The atomic unit here is the entire migration.

**Implementation order** (from exploration):
1. `go get charm.land/bubbletea/v2@v2.0.7 charm.land/lipgloss/v2@v2.0.3`
2. `go doc charm.land/bubbletea/v2.KeyPressMsg` — verify struct fields before writing code
3. Update `cmd/login.go`, `cmd/profile.go` (import-only — quick wins)
4. Update `cmd/pick.go` (key type assertion + View return type)
5. Update `cmd/wizard.go` (most complex — `msg.Type` switch → `msg.String()` switch + View)
6. Update `cmd/pick_test.go`, `cmd/wizard_test.go` (KeyMsg constructors)
7. `go mod tidy` + `go build ./...` + `go test ./cmd/...` + `go vet ./...`

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `go.mod` / `go.sum` | Modified | Replace v1 deps with v2, indirect deps may change |
| `cmd/pick.go` | Modified | Import paths, `KeyMsg`→`KeyPressMsg`, `View()` return `tea.View` |
| `cmd/pick_test.go` | Modified | Import path, 5× `KeyMsg` constructors → `KeyPressMsg` |
| `cmd/wizard.go` | Modified | Import paths, `msg.Type` switch → `msg.String()` switch, `View()` return `tea.View` |
| `cmd/wizard_test.go` | Modified | Import path, 5× `KeyMsg` constructors → `KeyPressMsg` |
| `cmd/login.go` | Modified | Import path only |
| `cmd/profile.go` | Modified | Import path only |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| `KeyPressMsg` struct fields differ from exploration assumptions | Medium | Run `go doc` immediately after `go get`; verify before writing code |
| `tea.KeyEsc`/`tea.KeyEnter` constants removed in v2 | Low | Fall back to string matching (`"esc"`, `"enter"`) — canonical in v2 |
| `tea.View` struct accessor for test assertions | Medium | Verify with `go doc tea.View`; likely has `.String()` or equivalent |
| Indirect dependency conflicts (termenv → colorprofile) | Low | `go mod tidy` resolves; pin versions if needed |

## Rollback Plan

`git revert <commit-sha>` — single commit, clean revert. No data migration, no schema changes, no config changes. The old v1 dependencies remain in the Go module proxy and can be re-resolved instantly.

## Dependencies

- Go 1.25+ (already required by project)
- `charm.land/bubbletea/v2 v2.0.7` (verified available)
- `charm.land/lipgloss/v2 v2.0.3` (verified available)

## Key Decisions

### 1. Commit strategy: **Single commit**
All changes are coupled — code does not compile with mixed v1/v2 imports. Splitting into deps→code→tests would produce 2 broken intermediate states. One atomic commit keeps `git bisect` clean.

### 2. bubbles dependency: **Defer to tui-overhaul**
Project does not currently use `bubbles` (v1 or v2). Adding `charm.land/bubbles/v2` now would be an unused dependency, violating AGENTS.md: _"MUST NOT add dependencies for trivial functionality."_ Add it when tui-overhaul actually needs bubbles components.

### 3. AGENTS.md updates: **Defer**
Current AGENTS.md rules are framework-agnostic (test TUI models, don't test `Program.Run()`). v2-specific rules (e.g., "use `KeyPressMsg` not `KeyMsg`") would be premature — they constrain code that doesn't exist yet. Revisit during tui-overhaul if new patterns emerge.

## Success Criteria

- [ ] `go build ./...` succeeds with zero errors
- [ ] `go test ./cmd/...` — all existing tests pass (no test logic changes beyond API adaptation)
- [ ] `go vet ./...` — clean
- [ ] `golangci-lint run` — clean (if configured in CI)
- [ ] No behavioral changes: `bak pick`, `bak profile create --interactive`, `bak login --interactive` work identically
- [ ] `go.mod` contains `charm.land/bubbletea/v2` and `charm.land/lipgloss/v2`, no `github.com/charmbracelet/bubbletea` or `github.com/charmbracelet/lipgloss`
