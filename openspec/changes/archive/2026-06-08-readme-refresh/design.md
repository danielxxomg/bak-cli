# Design: README Refresh

## Decisions

### D1: Badge Selection and Sources

**Decision**: Add 3 new badges — Go version, Platform, Tests/CI.

| Badge | Source | URL Pattern |
|-------|--------|-------------|
| Go version | shields.io | `img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go` |
| Platform | shields.io | `img.shields.io/badge/Platform-macOS%20%7C%20Linux%20%7C%20Windows-blue` |
| Tests/CI | GitHub Actions | `github.com/actions/workflows/.../badge.svg` (if workflow exists) or shields.io |

**Rationale**: shields.io badges are stable, don't require CI integration, and auto-render on GitHub. Go Report Card and License badges already exist.

### D2: Installation Hierarchy

**Decision**: Homebrew (macOS/Linux) and Scoop (Windows) as recommended. Everything else in `<details>`.

```
## Installation

### macOS / Linux (Recommended)
brew install --cask danielxxomg/tap/bak

### Windows (Recommended)
scoop bucket add ... && scoop install bak

<details>
<summary>Alternative install methods</summary>

#### Debian/Ubuntu ...
#### RHEL/Fedora ...
#### Go install ...
#### From Source ...

</details>
```

**Rationale**: Package managers are the fastest path. Source/Go install are developer-only.

### D3: Platform Table Placement

**Decision**: Place platform table immediately after badges, before Features.

```
## Supported Platforms

| Platform | Install Method | Package |
|----------|---------------|---------|
| macOS | Homebrew | `brew install --cask` |
| Linux | Homebrew, deb, rpm | Multiple |
| Windows | Scoop | `scoop install` |
```

**Rationale**: Users check platform compatibility before reading features.

### D4: Contributing Condensation

**Decision**: Reduce Contributing to 3 lines + link. Move adapter interface example to CONTRIBUTING.md.

```
## Contributing

Contributions welcome! See [CONTRIBUTING.md](CONTRIBUTING.md) for the full guide.

Quick start: fork → branch → commit → push → PR.
```

**Rationale**: CONTRIBUTING.md already exists. Duplicating steps in README adds noise.

### D5: Next Steps Content

**Decision**: 4 items linking to advanced features:

1. `bak wizard` — Interactive setup wizard
2. `bak schedule` — Automated backup scheduling
3. Custom presets — `~/.config/bak/presets/`
4. Custom adapters — `~/.config/bak/adapters/`

**Rationale**: Gives users a clear path beyond `bak backup`.

### D6: Brand Assets Collapsible

**Decision**: Wrap in `<details><summary>Brand Assets</summary>...</details>`.

**Rationale**: Brand assets are rarely needed by most readers. Collapsing reduces scroll depth.

## Section Order (Final)

1. Banner image
2. Badges (6+)
3. One-liner description
4. Supported Platforms (NEW — table)
5. Features
6. Installation (recommended + collapsible alternatives)
7. Quick Start
8. Commands
9. Configuration (unchanged)
10. Architecture (unchanged)
11. Safety Guarantees
12. Contributing (condensed + link)
13. Next Steps (NEW)
14. Roadmap
15. Brand Assets (collapsible)
16. License

## No Code Changes

This change is documentation-only. Zero Go code modifications. Zero test impact.
