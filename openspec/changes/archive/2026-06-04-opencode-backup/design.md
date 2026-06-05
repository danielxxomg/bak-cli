# Design: bak — AI Coding Config Backup CLI

## Technical Approach

Go CLI using adapter pattern per agent tool. Each adapter knows how to detect, list, backup, and restore configs for its target. A backup engine orchestrates adapters, builds a manifest, and writes to a git-tracked directory. Restore reverses the flow with mandatory dry-run diff and git safety net. GitHub Gist provides cloud sync v1.

## Architecture Decisions

| Decision | Options Considered | Choice | Rationale |
|----------|-------------------|--------|-----------|
| Adapter granularity | Per-file vs per-agent vs per-category | **Per-agent** (OpenCode, Claude Code, etc.) | Matches Gentle AI's proven pattern; each agent has unique path layouts and config formats |
| Backup storage | tar.gz, flat dir, git repo | **Git repo** (`~/.bak/`) | chezmoi/yadm inspiration; free history, diff, undo via `git revert`. No snapshot garbage |
| Restore safety | Trust user, always overwrite, dry-run gate | **Mandatory dry-run → auto-commit → apply → auto-commit** | Spec requires dry-run gate. Git commits before/after enable `bak undo` |
| Manifest format | YAML, TOML, JSON | **JSON** (`manifest.json`) | Git-friendly diffs, stdlib support, inspectable. Matches Gentle AI precedent |
| Merge vs overwrite | Deep merge (OpenCode semantics), overwrite | **Overwrite (v1)** | OpenCode's merge involves array concat + remote config — too complex for v1. Deferred |
| Secrets detection | Allowlist, denylist, regex patterns | **Regex patterns** (API keys, tokens, passwords) | Simple, extensible. Generate `.env.example` with placeholders |
| Cloud sync v1 | GitHub repo, Gist, S3, rclone | **GitHub Gist** (private) | Lowest complexity. Token auth, REST API. v2: repo, Codeberg, rclone |
| Git library | shelling out to git, go-git | **go-git** | No external git dependency. Cross-platform. Embedded repo management |
| Path storage | Absolute, relative to home, canonical `~/` | **Canonical `~/`-prefixed** | OS-agnostic. Restore resolves `~` to current home. Handles Windows drive letters |

## Data Flow

### Backup

```
User runs `bak backup [--preset quick|full|skills]`
         │
         ▼
   ┌─────────────┐     ┌──────────────┐
   │  Preset      │────▶│  Resolve     │
   │  Resolver    │     │  Categories  │
   └─────────────┘     └──────┬───────┘
                              │
         ┌────────────────────┼────────────────────┐
         ▼                    ▼                     ▼
  ┌─────────────┐    ┌──────────────┐     ┌──────────────┐
  │ OpenCode    │    │ Claude Code  │     │  ...other    │
  │ Adapter     │    │ Adapter      │     │  adapters    │
  │ .Detect()   │    │ .Detect()    │     │  .Detect()   │
  │ .ListItems()│    │ .ListItems() │     │  .ListItems()│
  │ .Backup()   │    │ .Backup()    │     │  .Backup()   │
  └──────┬──────┘    └──────┬───────┘     └──────┬───────┘
         │                   │                    │
         └───────────────────┼────────────────────┘
                             ▼
                    ┌─────────────────┐
                    │  Backup Engine   │
                    │  1. Copy files   │
                    │  2. Scan secrets │
                    │  3. Build manifest│
                    │  4. Git commit   │
                    └────────┬────────┘
                             ▼
                    ~/.bak/backups/<id>/
                    ├── manifest.json
                    ├── opencode/
                    │   ├── opencode.json
                    │   ├── skills/...
                    │   └── commands/...
                    └── .env.example
```

### Restore

```
User runs `bak restore <id>`
         │
         ▼
   ┌──────────────┐     ┌────────────────┐
   │ Read manifest │────▶│ Normalize paths │
   │ from backup   │     │ for target OS   │
   └──────────────┘     └───────┬────────┘
                                ▼
                       ┌────────────────┐
                       │  Dry-run diff   │ ◀── MANDATORY gate
                       │  (show changes) │
                       └───────┬────────┘
                               │ user confirms
                               ▼
                      ┌─────────────────┐
                      │ Git: commit      │
                      │ current state    │
                      │ (pre-restore)    │
                      └───────┬─────────┘
                              ▼
                      ┌─────────────────┐
                      │ Apply files      │
                      │ (overwrite)      │
                      └───────┬─────────┘
                              ▼
                      ┌─────────────────┐
                      │ Git: commit      │
                      │ new state        │
                      │ (post-restore)   │
                      └─────────────────┘
```

## Key Interfaces

### Adapter Interface

```go
// internal/adapters/adapter.go
type Adapter interface {
    // Identity
    Name() string                    // "opencode", "claude-code"
    
    // Detection
    Detect(homeDir string) (installed bool, configDir string, err error)
    
    // Inventory — returns categorized file/directory paths
    ListItems(homeDir string, categories []string) ([]Item, error)
    
    // Backup — copy files from source to backup dir
    Backup(homeDir string, backupDir string, items []Item) error
    
    // Restore — copy files from backup dir to target, adapting paths
    Restore(backupDir string, homeDir string, items []Item) error
}

type Item struct {
    Category    string // "skills", "commands", "config", "mcp", "plugins", "agents"
    SourcePath  string // absolute path on disk (canonical form)
    RelPath     string // relative to adapter's config dir
    IsDir       bool
    Hash        string // SHA-256 of content (for diff)
}
```

### Registry

```go
// internal/adapters/registry.go
type Registry struct {
    adapters map[string]Adapter
}

func NewRegistry() *Registry                    // empty
func (r *Registry) Register(a Adapter) error    // add adapter
func (r *Registry) DetectAll(homeDir string) []DetectedAdapter
func (r *Registry) Get(name string) (Adapter, bool)
func (r *Registry) All() []Adapter
```

### Manifest Schema

```json
{
  "version": "1.0.0",
  "id": "20260604-232200",
  "created_at": "2026-06-04T23:22:00Z",
  "os_source": "windows",
  "hostname": "dev-machine",
  "bak_version": "0.1.0",
  "preset": "full",
  "categories": ["skills", "commands", "config", "mcp", "plugins"],
  "adapters": {
    "opencode": {
      "version_detected": "1.2.3",
      "config_dir": "~/.config/opencode",
      "items": [
        {
          "category": "skills",
          "source_path": "~/.config/opencode/skills/my-skill/SKILL.md",
          "backup_path": "opencode/skills/my-skill/SKILL.md",
          "hash": "sha256:abc123...",
          "size": 2048
        }
      ]
    }
  },
  "secrets_excluded": true,
  "file_count": 47,
  "total_size": 102400
}
```

## Config Discovery

### OpenCode (first-class adapter)

Based on source analysis of `opencode-fork/packages/opencode/src/config/`:

| What | Path | Notes |
|------|------|-------|
| Global config dir | `~/.config/opencode/` | XDG on Linux, `%APPDATA%` on Windows adapted via Go's `os.UserConfigDir()` |
| Main config | `opencode.jsonc` → `opencode.json` → `config.json` | Merge order: later overrides. Deep merge with array concat for `instructions` |
| Skills | `~/.config/opencode/skills/` | Each skill is a directory with `SKILL.md` |
| Commands | `~/.config/opencode/commands/` | Markdown files |
| Agents | `~/.config/opencode/agent/` | Markdown files defining sub-agents |
| Plugins | `~/.config/opencode/plugins/` | TypeScript/JSON plugin files |
| System prompt | `~/.config/opencode/AGENTS.md` | Single file, full replace strategy |
| MCP config | `~/.config/opencode/opencode.json` (embedded) or `mcp.json` | Merged into settings or separate file |
| TUI config | `~/.config/opencode/tui.json` | Theme, keybinds |

### Community Adapters (v2 stubs)

Each adapter implements `Detect()` by checking its known config directory:
- Claude Code: `~/.claude/`
- Cursor: `~/.cursor/`
- Codex: `~/.codex/`
- Kiro: `~/.kiro/`
- pi.dev: `~/.pi/`
- KiloCode: `~/.kilocode/`
- CommandCode: TBD

## Error Handling Strategy

| Scenario | Behavior |
|----------|----------|
| Adapter detect fails | Log warning, skip adapter, continue backup |
| File unreadable | Skip file, record in manifest as `error: "permission_denied"` |
| Secret detected | Exclude file content, add path to `.env.example`, warn user |
| Git init fails | Fallback to plain directory backup, warn user |
| Restore path outside home | **Refuse** — security check (borrowed from Gentle AI's `isPathUnderHome`) |
| Manifest tampered | Validate hash per-file, refuse mismatched entries |
| Network failure (push/pull) | Retry 3x with backoff, clear error message |
| Disk full | Check before backup, abort with clear message |

All errors use `fmt.Errorf("context: %w", err)` wrapping. CLI displays user-friendly messages; `--verbose` shows full stack.

## Testing Strategy

| Layer | What | Approach |
|-------|------|----------|
| Unit | Adapter detect/list, path normalization, secret detection, manifest serialization | Table-driven tests with temp dirs. Mock FS for path tests |
| Integration | Backup→restore round-trip, git operations, preset resolution | Real FS in `t.TempDir()`. Verify file content equality |
| E2E | Full CLI commands (`bak backup`, `bak restore`, `bak push/pull`) | `exec.Command` against built binary. Assert exit codes + stdout |
| Cross-platform | Path normalization Windows↔Linux | CI matrix (Windows, macOS, Linux). Canonical path tests |

## Release Strategy

- **goreleaser**: Cross-compile for Windows (exe), macOS (arm64+amd64), Linux (arm64+amd64)
- **Distribution**: GitHub Releases, Homebrew tap, Scoop manifest (Windows), apt repo (Linux)
- **Versioning**: SemVer. `bak version` prints build info from goreleaser ldflags
- **Linting**: golangci-lint with `govet`, `errcheck`, `staticcheck`, `gofmt`

## Open Questions

- [ ] Should `bak restore` support selective category restore (e.g., only skills)?
- [ ] How to handle OpenCode project-level `.opencode/` configs — include in backup or global only?
- [ ] Gist naming convention: one gist per backup, or one gist updated per push?
- [ ] Should `bak undo` be a thin wrapper over `git revert HEAD` or have its own logic?
