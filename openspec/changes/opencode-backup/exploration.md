## Exploration: OpenCode Backup CLI Tool

### 1. Existing Solutions (Competitive Landscape)

| Tool / Project | URL / Source | Stars | What it backs up | Tech Stack | Status | Learnings for us |
|---|---|---|---|---|---|---|
| **opencode-synced** (plugin) | OpenCode ecosystem plugin | Unknown | Global `~/.config/opencode` dir, sessions, secrets, prompts | TS (OpenCode plugin) | Active | Syncs via GitHub repo; proves demand for OpenCode config sync. Plugin model limits portability. |
| **opencode-migration** (dlukt/) | GitHub community script | Low | Prompts, agent config, session data | Python | Maintained | Auto-backup before migration; `unapply.py` for rollback. Shows need for safety nets. |
| **chezmoi** | github.com/twpayne/chezmoi | ~20k+ | Any dotfiles, cross-platform | Go | Very active | Gold standard: Go templates, secrets mgmt (age/gpg/1Password), `diff`/dry-run, one-command bootstrap. **Best-in-class reference.** |
| **GNU Stow** | Classic Unix tool | N/A | Symlink farms for dotfiles | Perl | Stable | Too low-level for our scope; no merge/strategy logic. |
| **Cursor IDE** | cursor.com | N/A | Manual: `%APPDATA%\Cursor`, `~/.cursor` | N/A (no tool) | N/A | No native export; users copy folders manually. Pain point we can solve. |
| **Windsurf/Codeium** | windsurf.com | N/A | Manual: `~/.codeium/` (settings, memories, database) | N/A (no tool) | N/A | Same manual pattern as Cursor. No CLI tooling exists. |
| **Claude Code** | claude.com | N/A | Manual: `~/.claude/` (settings.json, CLAUDE.md, commands/, agents/, hooks/) | N/A (no tool) | N/A | Users version-control `~/.claude/` with Git. Proves "AI dotfiles" is a real category. |
| **dfm (dotfiles-manager)** | GitHub community | Low | Dotfiles + AI config mirroring to GitHub | Go | New | AI-assisted config improvement proposals. Interesting but too early. |

**Key insight**: No single tool backs up AI coding assistant configs holistically. Existing solutions are either (a) manual file copying, (b) generic dotfiles managers that don't understand AI-specific structures, or (c) tool-specific plugins. **There's a clear gap for a purpose-built CLI that understands OpenCode + Gentle AI config structures.**

---

### 2. OpenCode Config Structure Analysis

**Source**: `C:\Users\sumad\Desktop\opencode-fork` (TypeScript monorepo, Effect-TS based)

#### Config Paths (cross-platform)
| Platform | Global Config Dir | Settings File |
|---|---|---|
| Windows | `%USERPROFILE%\.config\opencode\` | `opencode.json` or `opencode.jsonc` |
| macOS | `~/.config/opencode/` | `opencode.json` or `opencode.jsonc` |
| Linux | `~/.config/opencode/` | `opencode.json` or `opencode.jsonc` |

Additional paths:
- Auth: `~/.local/share/opencode/auth.json`
- Models cache: `~/.cache/opencode/models.json`
- Project config: `.opencode/` directory (walks up from cwd)

#### What lives in `~/.config/opencode/` (from actual user directory)
```
~/.config/opencode/
├── AGENTS.md              # System prompt (global instructions)
├── opencode.json          # Main config: agents, mcp, permissions, plugins, providers
├── tui.json               # TUI-specific settings
├── skills/                # 39 skill directories (SDD, testing, frontend, etc.)
│   ├── _shared/
│   ├── sdd-apply/
│   ├── sdd-explore/
│   └── ... (37 more)
├── plugins/               # Local plugin files
│   ├── background-agents.ts
│   ├── engram.ts
│   └── model-variants.ts
├── commands/              # Slash commands (10 .md files)
├── prompts/               # Prompt templates
│   └── sdd/              # SDD phase prompts
├── profiles/              # SDD profile configs (empty currently)
├── secrets/               # API keys, tokens (SENSITIVE)
├── mcp.json               # Legacy MCP config
├── node_modules/          # Plugin dependencies
├── package.json           # Plugin dependency manifest
└── .gitignore
```

#### opencode.json Structure (the main config file)
```json
{
  "$schema": "https://opencode.ai/config.json",
  "agent": { ... },         // Custom agent definitions (model, prompt, tools, permissions)
  "mcp": { ... },           // MCP server configs (local + remote, with commands/env/headers)
  "permission": { ... },    // Bash/read permission rules
  "plugin": [ ... ],        // Plugin specifiers (npm packages or local paths)
  "provider": { ... },      // Custom model providers (models, costs, limits, variants)
  "share": "disabled"       // Session sharing setting
}
```

**Complexity note**: The user's actual `opencode.json` is 1292 lines with 30+ agent definitions, 12 MCP servers, 3 custom providers with 20+ models each. This is NOT a simple config file — it's a rich, structured document.

#### Config Loading Order (from config.ts)
1. `~/.config/opencode/config.json` (legacy compat)
2. `~/.config/opencode/opencode.json`
3. `~/.config/opencode/opencode.jsonc`
4. Project-level `.opencode/` directories (walk up)
5. Remote configs (URL-based, with variable substitution)

Arrays are concatenated (not replaced) during merge. Instructions are deduplicated.

---

### 3. Gentle AI Config Structure Analysis

**Source**: `C:\Users\sumad\Desktop\gentle-ai-fix` (Go, bubbletea TUI)

#### What Gentle AI Installs/Manages
Gentle AI is a Go CLI + TUI that configures AI coding agents. For OpenCode specifically:

| Component | Path | Strategy |
|---|---|---|
| System prompt | `~/.config/opencode/AGENTS.md` | FileReplace (full overwrite) |
| Skills | `~/.config/opencode/skills/` | Directory of SKILL.md files |
| MCP servers | Merged into `opencode.json` | StrategyMergeIntoSettings |
| Slash commands | `~/.config/opencode/commands/` | Markdown files |
| Plugins | `~/.config/opencode/plugins/` | TypeScript files |
| SDD prompts | `~/.config/opencode/prompts/sdd/` | Markdown files per phase |
| SDD model profiles | Agent entries in `opencode.json` | Generated-multi (suffixed agents) |

#### SDD Profiles (balanced|economy|performance)
From `model/types.go` and actual config:
- **Default profile**: Base `sdd-*` agents (e.g., `sdd-apply`, `sdd-explore`)
- **Named profiles**: Generate suffixed copies (e.g., `sdd-apply-balanceado`, `sdd-explore-balanceado`)
- Each profile assigns different models per phase via `ModelAssignment{ProviderID, ModelID, Effort}`
- Profile strategy: `SDDProfileStrategyGeneratedMulti` — named profiles coexist as suffixed agents in opencode.json

#### Gentle AI's Own Config
- State: `~/.gentle-ai/` (internal state, selections, cache)
- Backups: `~/.gentle-ai/backups/` (already has a backup system!)
- Cache: `~/.gentle-ai/cache/model-variants.json`

#### Gentle AI's Existing Backup System
Gentle AI ALREADY has a backup module (`internal/backup/`) with:
- `manifest.go` — JSON manifest with ID, timestamps, entries, source, checksums
- `snapshot.go` — Creates tar.gz archives of config files
- `restore.go` — Restores from backup with safety checks
- `retention.go` — Prunes old backups (pinned backups excluded)
- `compression.go` — tar.gz archive creation/extraction

**This is critical**: Gentle AI's backup system is designed for pre-operation safety nets (backup before install/sync/upgrade/uninstall), NOT for cross-machine migration. Our tool needs a different backup format that's portable.

#### Adapter Interface (what we need to understand per agent)
```go
type Adapter interface {
    Agent() model.AgentID
    GlobalConfigDir(homeDir string) string
    SystemPromptDir(homeDir string) string
    SystemPromptFile(homeDir string) string
    SkillsDir(homeDir string) string
    SettingsPath(homeDir string) string
    CommandsDir(homeDir string) string
    // ... etc
}
```

Gentle AI supports 14 agents: claude-code, opencode, kilocode, gemini-cli, cursor, vscode-copilot, codex, antigravity, windsurf, kimi, qwen-code, kiro-ide, openclaw, pi, trae-ide.

---

### 4. Stack Recommendation

#### Evaluation Matrix

| Technology | Verdict | Reasoning |
|---|---|---|
| **Go** | ✅ **RECOMMENDED** | Gentle AI is Go + bubbletea/lipgloss. Same ecosystem = potential code reuse (backup module, adapter interface, path resolution). Single binary distribution. Excellent cross-platform. `charmbracelet` TUI ecosystem is mature. |
| **Rust** | ⚠️ Maybe | Safe, fast, single binary. But: no code reuse with Gentle AI. Steeper learning curve. Ecosystem mismatch. Only if performance is critical (it isn't for a backup tool). |
| **Node.js/Bun** | ⚠️ Maybe | OpenCode is TypeScript. Could share config parsing logic. BUT: distribution requires runtime. `bun build --compile` is new. No TUI ecosystem as rich as charmbracelet. |
| **Hono** | ❌ Not suitable | Web framework. We're building a CLI, not a server. |
| **React/Vue/Next.js** | ❌ Not suitable | Frontend frameworks. Irrelevant for CLI tool. |
| **Tailwind** | ❌ Not suitable | CSS framework. Irrelevant for CLI. |
| **Drizzle ORM** | ❌ Not suitable | ORM. We don't need a database for backup/restore. |
| **Vite** | ❌ Not suitable | Build tool for web apps. Irrelevant. |
| **Oxlint** | ❌ Not suitable | Linter. Irrelevant. |
| **Playwright** | ⚠️ Maybe | E2E testing. Could use for integration tests but overkill. Go's `testing` + `testify` is sufficient. |

#### Recommended Stack

```
Language:     Go 1.24+
TUI:          bubbletea + lipgloss (same as Gentle AI)
CLI:          cobra or flag (standard library)
Archive:      archive/tar + compress/gzip (stdlib)
Config:       encoding/json (stdlib) + jsonc parser
Paths:        os.UserHomeDir + filepath (stdlib, cross-platform)
Testing:      testing + testify
Distribution: goreleaser (same as Gentle AI)
```

#### Why Go wins decisively:
1. **Code reuse**: Gentle AI's `internal/backup/`, `internal/agents/` adapter interface, path resolution, platform detection — all directly reusable or at least reference-able
2. **Single binary**: `go build` → one executable, no runtime dependencies
3. **Cross-platform**: Go's `filepath`, `os.UserHomeDir` handle Windows/macOS/Linux paths natively
4. **Distribution**: goreleaser (already configured in Gentle AI) produces platform-specific binaries
5. **Ecosystem alignment**: Same team, same patterns, same TUI library

---

### 5. Open Design Questions

#### Must decide before proposal:

1. **Backup format**: Single `.tar.gz` archive with manifest.json? Or a directory structure? 
   - Recommendation: tar.gz with manifest (matches Gentle AI's existing pattern, portable, compact)

2. **Scope**: OpenCode only? Or multi-agent (Claude Code, Cursor, Windsurf)?
   - Recommendation: Start OpenCode-only + Gentle AI state. Design adapter interface for future multi-agent.

3. **Selective backup/restore**: All-or-nothing vs. pick-and-choose?
   - Recommendation: Selective. Categories: skills, agents, mcp, plugins, commands, prompts, providers, permissions.

4. **Secrets handling**: Include `secrets/` dir? Or exclude by default with opt-in?
   - Recommendation: EXCLUDE by default. Secrets are machine-specific and sensitive. Warn loudly if user opts in.

5. **Dry-run / diff**: Show what would change before restoring?
   - Recommendation: Yes, mandatory. `restore --dry-run` shows file-by-file diff. Critical for safety.

6. **Cloud sync**: GitHub Gist? S3? Git repo?
   - Recommendation: v1 = local file only. v2 = GitHub private repo (like opencode-synced does).

7. **Cross-platform paths**: MCP configs contain absolute paths (e.g., `C:\Users\sumad\...`). How to handle?
   - Recommendation: Path normalization layer. Detect absolute paths in backup, mark as machine-specific, warn on restore to different OS.

8. **Encryption**: Should backups be encrypted?
   - Recommendation: v1 = no encryption (local files). v2 = optional age encryption for cloud sync.

9. **Relationship to Gentle AI's existing backup**: Extend? Replace? Independent?
   - Recommendation: Independent tool. Gentle AI's backup is for pre-operation safety nets. This tool is for cross-machine migration. Different use cases.

10. **Name**: `opencode-backup`? `ocbk`? `pack-opencode`?
    - Recommendation: Short binary name (`ocbk` or `opack`), descriptive repo name (`opencode-backup`).

---

### 6. Affected Areas (what the tool must understand)

- `~/.config/opencode/opencode.json` — Main config (agents, mcp, providers, permissions, plugins)
- `~/.config/opencode/AGENTS.md` — Global system prompt
- `~/.config/opencode/skills/` — 39+ skill directories
- `~/.config/opencode/plugins/` — Local TypeScript plugin files
- `~/.config/opencode/commands/` — Slash command markdown files
- `~/.config/opencode/prompts/` — Prompt templates (SDD phases)
- `~/.config/opencode/secrets/` — API keys (EXCLUDE by default)
- `~/.config/opencode/tui.json` — TUI settings
- `~/.config/opencode/profiles/` — SDD profile configs
- `~/.gentle-ai/` — Gentle AI state, cache, model assignments
- `~/.local/share/opencode/auth.json` — OAuth credentials (EXCLUDE)
- `~/.cache/opencode/models.json` — Models cache (EXCLUDE, regenerated)

---

### 7. Recommendation

**Build a Go CLI tool** that:
1. Backs up OpenCode + Gentle AI config to a portable `.tar.gz` with manifest
2. Supports selective backup by category (skills, agents, mcp, etc.)
3. Restores with dry-run preview and file-by-file diff
4. Handles cross-platform path normalization for MCP configs
5. Excludes secrets and auth by default with clear warnings
6. Uses the adapter pattern from Gentle AI for future multi-agent support

**Start with OpenCode-only scope**, design for extensibility.

---

### Ready for Proposal
**Yes.** The exploration has identified:
- Clear market gap (no holistic AI config backup tool exists)
- Well-understood config structures (from source code analysis)
- Recommended tech stack (Go, matching Gentle AI ecosystem)
- 10 open design questions that need user input before proposal
- Existing Gentle AI backup code that can inform the design

The orchestrator should present the design questions to the user and collect preferences before launching `sdd-propose`.
