# Contributing to bak-cli

Thank you for your interest in contributing to **bak** — the CLI that backs up, restores, and syncs your OpenCode AI coding configuration. This guide will help you get started.

## Table of Contents

- [Development Environment](#development-environment)
- [Running Tests](#running-tests)
- [Building](#building)
- [Code Style](#code-style)
- [Adding a New Adapter](#adding-a-new-adapter)
- [Commit Conventions](#commit-conventions)
- [Pull Request Process](#pull-request-process)

## Development Environment

### Prerequisites

- **Go 1.25+** — [download](https://go.dev/dl/)
- **Git** — [download](https://git-scm.com/)
- **golangci-lint** (optional, for linting) — [install](https://golangci-lint.run/welcome/install/)

### Setup

```bash
# Clone the repository
git clone https://github.com/danielxxomg/bak-cli.git
cd bak-cli

# Download dependencies
go mod download

# Verify everything works
go vet ./...
go build -o bak .
```

The project uses Go modules. All dependencies are declared in `go.mod`. The module path is `github.com/danielxxomg/bak-cli`.

### Recommended: golangci-lint

```bash
# Install (macOS)
brew install golangci-lint

# Install (Linux/macOS with Go)
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Install (Windows)
choco install golangci-lint

# Run
golangci-lint run
```

Or use the Task target:

```bash
task lint
```

## Running Tests

```bash
# Run all tests
go test ./...

# Verbose output
go test -v ./...

# With coverage
go test -cover ./...

# With coverage profile (for HTML report)
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

Or use Task targets:

```bash
task test           # go test ./...
task test-verbose   # go test -v ./...
task cover          # go test -cover ./...
```

### Test Requirements

- **Coverage target**: >80% for new code
- **Test isolation**: Use `t.TempDir()` — never write to the real filesystem
- **Table-driven tests**: Prefer `[]struct{ name string; ... }` for unit tests
- **Happy path and error paths**: Both must be covered
- **Edge cases**: Empty input, missing files, permission errors
- **Cross-platform paths**: Test Windows (`\`), macOS, and Linux (`/`) separators

### Example Test Structure

```go
func TestBackupPreset(t *testing.T) {
    tests := []struct {
        name    string
        preset  string
        wantErr bool
    }{
        {"valid quick preset", "quick", false},
        {"valid full preset", "full", false},
        {"invalid preset", "unknown", true},
        {"empty preset", "", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // ...
        })
    }
}
```

## Building

```bash
# Simple build
go build -o bak .

# With version info (matching release builds)
go build -ldflags "-s -w -X github.com/danielxxomg/bak-cli/cmd.Version=dev -X github.com/danielxxomg/bak-cli/cmd.Commit=$(git rev-parse --short HEAD) -X github.com/danielxxomg/bak-cli/cmd.Date=$(date -u +%Y-%m-%dT%H:%M:%SZ)" -o bak .

# Using Task (recommended)
task build
```

### Build Verification

After building, run the smoke test:

```bash
./bak version
./bak --help
```

### Release Builds

Releases are built with [GoReleaser](https://goreleaser.com/). Configuration is in `.goreleaser.yaml`.

```bash
# Dry-run (snapshot)
task goreleaser-verify

# Full release
task goreleaser-verify
```

The release matrix covers:

| OS | Arch |
|----|------|
| Linux | amd64, arm64 |
| macOS (darwin) | amd64, arm64 |
| Windows | amd64, arm64 |

## Code Style

The project follows the conventions documented in **[AGENTS.md](AGENTS.md)**. Key points:

### Go Idioms

- Use `fmt.Errorf("context: %w", err)` for error wrapping — never bare `errors.New` for wrapped errors
- Prefer interfaces over concrete types for testability
- Never use `panic` in library code — return errors
- Handle ALL returned errors — no `_ =` for error returns
- Use `filepath.Join` for OS-specific paths, `path.Clean` for canonical paths

### Error Messages

- Start with lowercase: `"backup dir: %w"` not `"Backup dir: %w"`
- Include operation context (what was being done when it failed)
- Never include sensitive data (tokens, paths with usernames)

### Security

- Validate all paths stay under user home directory (path traversal prevention)
- Never include secrets/API keys/tokens in backups
- Use `os.UserHomeDir()` — never hardcode home paths
- Use `path.Clean` + `strings.ReplaceAll(path, "\\", "/")` for canonical path comparison

## Adding a New Adapter

The project uses an **adapter pattern** to support multiple AI coding tools. Currently, 8 adapters are implemented (Claude Code, Cursor, Codex, Windsurf, Kiro, KiloCode, pi.dev, and OpenCode), and the interface is designed for extension.

### Step 1: Understand the Interface

Every adapter must implement the `adapters.Adapter` interface:

```go
// internal/adapters/adapter.go
type Adapter interface {
    Name() string
    Detect(homeDir string) (installed bool, configDir string, err error)
    ListItems(homeDir string, categories []string) ([]Item, error)
    Backup(homeDir string, backupDir string, items []Item) error
    Restore(backupDir string, homeDir string, items []Item) error
}
```

### Step 2: Create the Adapter Package

Create a new package under `internal/adapters/`:

```
internal/adapters/
├── adapter.go          # Adapter interface + Item struct
├── registry.go         # Registry (DetectAll, Register, etc.)
├── registry_test.go
├── opencode/           # Reference implementation
│   ├── adapter.go
│   └── adapter_test.go
└── youradapter/        # ← Your new adapter here
    ├── adapter.go
    └── adapter_test.go
```

### Step 3: Implement the Adapter

Use the existing OpenCode adapter (`internal/adapters/opencode/adapter.go`) as a reference. Each method:

| Method | Purpose |
|--------|---------|
| `Name()` | Returns a unique identifier (e.g., `"claude-code"`, `"cursor"`) |
| `Detect()` | Checks if the tool is installed (look for its config directory) |
| `ListItems()` | Enumerates files/dirs for requested categories with SHA-256 hashes |
| `Backup()` | Copies items from source to backup directory |
| `Restore()` | Copies items from backup directory back to source |

**Categories** your adapter should map to:

| Category | Example content |
|----------|----------------|
| `skills` | Agent skill files |
| `commands` | Custom command definitions |
| `config` | Root-level config files (JSON, AGENTS.md, etc.) |
| `mcp` | MCP server configurations |
| `plugins` | Plugin/extension files |
| `agents` | Agent definitions |

**Compile-time interface check** (include this in your adapter file):

```go
var _ adapters.Adapter = (*Adapter)(nil)
```

### Step 4: Register in the CLI

Open `cmd/backup.go` and register your adapter:

```go
import (
    // ... existing imports ...
    youradapter "github.com/danielxxomg/bak-cli/internal/adapters/youradapter"
)

func runBackup(cmd *cobra.Command, args []string) error {
    // ...

    reg := adapters.NewRegistry()
    if err := reg.Register(&opencodeadapter.Adapter{}); err != nil {
        return fmt.Errorf("register opencode adapter: %w", err)
    }
    // Add your adapter registration:
    if err := reg.Register(&youradapter.Adapter{}); err != nil {
        return fmt.Errorf("register youradapter adapter: %w", err)
    }

    // ...
}
```

### Step 5: Write Tests

Every adapter must have comprehensive tests:

```go
func TestAdapterName(t *testing.T) {
    a := &Adapter{}
    if a.Name() != "expected-name" {
        t.Errorf("Name() = %q, want %q", a.Name(), "expected-name")
    }
}

func TestDetect_NotFound(t *testing.T) {
    a := &Adapter{}
    installed, _, err := a.Detect(t.TempDir())
    if err != nil {
        t.Fatal(err)
    }
    if installed {
        t.Error("expected not installed in empty dir")
    }
}
```

Check the Opencode adapter test (`internal/adapters/opencode/adapter_test.go`) for patterns to follow.

### Step 6: Run the Full Pipeline

```bash
go test ./internal/adapters/...
go test ./...
task ci    # vet + test + build
```

## Commit Conventions

This project follows **[Conventional Commits](https://www.conventionalcommits.org/)**.

### Format

```
<type>: <short description>

<optional body>
<optional footer>
```

### Types

| Type | Usage |
|------|-------|
| `feat:` | New feature |
| `fix:` | Bug fix |
| `test:` | Adding or modifying tests |
| `chore:` | Maintenance (deps, build, tooling) |
| `docs:` | Documentation changes |
| `refactor:` | Code restructuring (no behavior change) |
| `perf:` | Performance improvement |

### Rules

- Commits must be **atomic** — one logical change per commit
- Use lowercase, imperative mood: `feat: add --force flag to restore` not `feat: Added --force flag`
- Never include AI attribution (`Co-Authored-By`) in commit messages

### Examples

```bash
feat: add support for Claude Code adapter
fix: prevent path traversal in restore command
test: add table-driven tests for preset validation
chore: upgrade go-git to v5.19.1
docs: document adapter interface in CONTRIBUTING.md
refactor: extract path sanitization into paths package
```

## Pull Request Process

### Before Opening

1. **Run the full pipeline**: `task ci` (vet → test → build)
2. **Check coverage**: `task cover` — ensure >80% for new code
3. **Lint**: `task lint` (if golangci-lint is installed)
4. **Self-review**: Go through the PR checklist in the [PR template](.github/pull_request_template.md)
5. **Update docs**: If your change affects the CLI interface, update `README.md`

### Creating the PR

1. Create a feature branch from `main`:
   ```bash
   git checkout -b feat/my-feature
   ```
2. Make your changes with [conventional commits](#commit-conventions)
3. Push and open a PR against `main`
4. Fill out the PR template completely

### PR Template Checklist

The PR template requires you to confirm:

- [ ] Code follows AGENTS.md standards
- [ ] Self-review completed
- [ ] Tests added for the change
- [ ] `go test ./...` passes
- [ ] `go vet ./...` passes
- [ ] Documentation updated if needed

### Review Process

- All PRs require at least one approving review
- CI must pass (tests + vet + build)
- The reviewer will check against [AGENTS.md](AGENTS.md) conventions
- Address feedback in new commits — squash at merge

### After Merge

- Delete your feature branch
- Celebrate! 🎉

## Project Structure

```
bak-cli/
├── cmd/                    # CLI commands (cobra)
│   ├── root.go             # Root command + Execute()
│   ├── backup.go           # bak backup
│   ├── restore.go          # bak restore
│   ├── push.go             # bak push
│   ├── pull.go             # bak pull
│   ├── pick.go             # bak pick (interactive TUI)
│   ├── wizard.go           # bak wizard (guided setup TUI)
│   ├── export.go           # bak export
│   ├── list.go             # bak list
│   ├── login.go            # bak login
│   ├── undo.go             # bak undo
│   ├── verify.go           # bak verify
│   ├── diff.go             # bak diff
│   ├── schedule.go         # bak schedule
│   └── version.go          # bak version
├── internal/
│   ├── adapters/           # Agent adapter interface + registry + GenericAdapter
│   │   ├── generic.go      # GenericAdapter base struct
│   │   ├── knowledge_test.go # Adapter knowledge validation
│   │   ├── claudecode/     # Claude Code adapter
│   │   ├── codex/          # OpenAI Codex adapter
│   │   ├── cursor/         # Cursor adapter
│   │   ├── kiro/           # Kiro adapter
│   │   ├── kilocode/       # KiloCode adapter
│   │   ├── opencode/       # OpenCode adapter
│   │   ├── pidev/          # pi.dev adapter
│   │   ├── windsurf/       # Windsurf adapter
│   │   └── register/       # Adapter registration
│   ├── actions/            # Business logic (cobra-free, io.Writer injection)
│   ├── backup/             # Backup engine + secret detection
│   ├── restore/            # Restore engine + dry-run + git safety
│   ├── manifest/           # Manifest schema + validation
│   ├── cloud/              # Cloud providers (Gist, GitHub Repo, Gitea, rclone)
│   │   ├── content_types.go # Shared content API types + helpers
│   │   └── httputil.go     # Shared HTTP helpers
│   ├── paths/              # Cross-platform path normalization (Slash, CanonicalPath)
│   ├── git/                # Git operations (go-git)
│   ├── config/             # Configuration management
│   ├── crypto/             # AES-256-GCM encryption
│   ├── diff/               # Backup diff engine
│   ├── presets/            # Preset definitions + loader
│   └── schedule/           # Scheduled backup (cron)
├── tests/
│   └── e2e/                # End-to-end testscript tests
├── main.go                 # Entry point
├── Taskfile.yml            # Development workflow targets
├── .goreleaser.yaml        # Cross-platform release config
├── AGENTS.md               # Code review rules (GGA enforced)
├── go.mod                  # Go module definition
└── go.sum                  # Dependency checksums
```

## Need Help?

- Open a [GitHub Issue](https://github.com/danielxxomg/bak-cli/issues) with the `question` label
- Check existing issues for similar questions
- Read [AGENTS.md](AGENTS.md) for the full code review standards
