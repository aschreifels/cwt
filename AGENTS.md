# AGENTS.md — cwt (cmux Worktree Tool)

## What This Project Is

`cwt` is a Go CLI that creates git worktrees with full [cmux](https://github.com/charmbracelet/cmux) dev environments. Each worktree gets its own AI coding agent (Crush or Claude Code), git TUI, and editor in a multi-pane workspace. It supports ticket integration (Linear, GitHub Issues, Jira), PR review workspaces, and bundles installable agent skills.

## Commands

```bash
# Build
make build              # → ./cwt binary (uses ldflags for version/commit/date)
make install            # → copies to $GOPATH/bin or /usr/local/bin

# Test
make test               # → go test ./... -v
go test ./...           # without verbose
go test ./internal/config/... -run TestSaveAndLoad  # single test

# Lint
make lint               # → go vet ./...

# Clean
make clean              # removes binary, runs go clean
```

## Project Structure

```
main.go                          # Entrypoint — calls cmd.Execute()
cmd/
  root.go                        # Cobra root command setup
  spawn.go                       # cwt spawn — main feature, creates worktree + cmux workspace
  init.go                        # cwt init — TUI config wizard (charmbracelet/huh)
  list.go                        # cwt list/ls — lists active worktrees
  review.go                      # cwt review — opens a PR review workspace (no worktree required)
  rm.go                          # cwt rm — removes worktrees, optional branch delete
  skills.go                      # cwt skills list/install — manages agent skills (Crush + Claude Code)
  version.go                     # cwt version — prints version info (injected via ldflags)
internal/
  config/
    config.go                    # Config types, Load/Save (TOML), template rendering
    config_test.go               # Thorough unit tests for config
  workspace/
    spawn.go                     # Core spawn logic: git worktree + cmux workspace creation
    spawn_test.go                # Unit tests for branch naming, dir resolution, prompts
    review.go                    # Review workspace logic: PR fetch, diff, prompt building
    review_test.go               # Unit tests for review prompt, workspace naming
  git/
    git.go                       # Git operations via exec.Command (worktree add/remove/list, branch ops)
    gh.go                        # GitHub CLI (gh) helpers: PR view, diff, repo detection, URL parsing
    gh_test.go                   # Unit tests for URL parsing
  cmux/
    client.go                    # cmux CLI wrapper (workspaces, splits, send, status, progress, notify)
  tui/
    spawn.go                     # Bubble Tea TUI for spawn progress (spinners, per-pane status)
skills/
  skills.go                      # Skill registry using //go:embed
  skills_test.go                 # Verifies all embedded skills exist and have content
  cmux-notifications/SKILL.md    # Skill: teaches agent to use cmux sidebar APIs
  cwt-orchestrator/SKILL.md      # Skill: teaches agent to orchestrate parallel worktrees
  cwt-reviewer/SKILL.md          # Skill: PR review router — detects languages, dispatches to domain skills
  cwt-reviewer-comments/SKILL.md # Skill: posts review findings as inline GitHub PR comments
  cwt-reviewer-go/SKILL.md       # Skill: Go-specific PR review checklist
  cwt-reviewer-typescript/SKILL.md # Skill: TypeScript-specific PR review checklist
  cwt-reviewer-database/SKILL.md # Skill: database/migration PR review checklist
  cwt-reviewer-infra/SKILL.md    # Skill: infrastructure/CI-CD PR review checklist
```

## Architecture and Key Patterns

### Cobra CLI Structure
- Each command is defined in `cmd/` as a package-level `*cobra.Command` var
- Commands register themselves via `func init()` calling `rootCmd.AddCommand()`
- Flags are defined as package-level vars and bound in `init()`
- `cmd.Execute()` is the sole public API from the `cmd` package

### Config System (`internal/config`)
- TOML config at `~/.config/cwt/config.toml` (respects `XDG_CONFIG_HOME`)
- `Config` struct with `Defaults`, `Layout`, `ProjectManagement`, and `Review` sections
- `Defaults.Agent` field: `"crush"` (default) or `"claude"` — determines pane commands, prompt injection, and skill installation
- `DefaultConfig()` provides sensible defaults for Crush; `DefaultConfigForAgent(agent)` for agent-specific defaults
- `Load()` merges file config with defaults — empty prompts, empty pane lists, and empty agent are backfilled
- Template rendering via `strings.NewReplacer` with `{{provider}}`, `{{ticket}}`, `{{project}}`, `{{name}}`, `{{worktree_dir}}`
- Helper methods on `Config`: `EnabledPanes()`, `MainPane()`, `SidePanes()`, `HasProjectManagement()`, `IsClaude()`, `RenderPrompt()`

### External Tool Integration
- **git**: All git operations in `internal/git/git.go` via `exec.Command` — no git library
- **gh**: GitHub CLI operations in `internal/git/gh.go` via `exec.Command("gh", ...)` — PR view, diff, repo detection
- **cmux**: All cmux operations in `internal/cmux/client.go` via `exec.Command("cmux", ...)` — wraps the CLI
- All three packages parse CLI output (porcelain format for git, JSON for gh, `OK <id>` for cmux)
- No mocking of these — tests that need git/cmux/gh are either pure-logic unit tests or require the real tools

### Spawn Flow (`workspace.Spawn`)
1. Build branch name (prefix + ticket + name)
2. Resolve worktree directory (config override or `../worktrees/<name>`)
3. Resolve prompt (agent-specific skill loading prompt + ticket context)
4. Build main command (for Claude: appends prompt as CLI arg; for Crush: plain command)
5. Create git worktree (or reuse existing)
6. Create cmux workspace with main pane
7. Create right/bottom splits for side panes
8. Inject prompt: Crush via `cmux send-text`, Claude via CLI argument to `claude "prompt"`
9. Set cmux sidebar status (branch, base, ticket, provider)
- Progress is reported via `chan StepUpdate` → Bubble Tea model

### Review Flow (`workspace.Review`)
1. Fetch PR metadata via `gh pr view --json` (title, body, files, author, branch)
2. Fetch PR diff via `gh pr diff`
3. Optionally check out PR branch in a lightweight worktree (`../worktrees/review-<pr>`)
4. Build review prompt (PR context + diff + review config template)
5. Create cmux workspace (reuses spawn's TUI and pane layout)
6. Inject prompt to agent (same mechanism as spawn)
7. Set cmux sidebar status (PR number, branch, author, review mode)
- The `--no-checkout` flag skips the worktree and reviews in the current directory
- The `--url` flag allows reviewing PRs from other repos

### TUI (`internal/tui`)
- Bubble Tea (charmbracelet/bubbletea) for the spawn progress view
- `Model` receives `StepUpdate` messages from a channel
- `stepState` tracks per-pane status (name, status string, done bool, error)
- Uses lipgloss for styled output with consistent color palette:
  - Purple `#a78bfa` — titles/headings
  - Green `#22c55e` — success
  - Red `#ef4444` — errors
  - Gray `#6b7280` — dim/secondary text
  - Amber `#f59e0b` — warnings/spinners

### Skills System
- Skills are embedded at compile time via `//go:embed` directives in `skills/skills.go`
- Each skill is a `SKILL.md` file in a subdirectory under `skills/`
- `skills.All()` returns the full registry (8 skills total)
- `skills.ReviewSkills()` returns only the 6 review-related skills
- Review skills use a router pattern: `cwt-reviewer` detects languages/domains and dispatches to specific skills (`cwt-reviewer-go`, `cwt-reviewer-typescript`, `cwt-reviewer-database`, `cwt-reviewer-infra`), with `cwt-reviewer-comments` handling GitHub PR comment posting
- Installation is agent-aware:
  - **Crush**: copies to `~/.config/crush/skills/<dir>/SKILL.md`
  - **Claude Code**: appends to `~/.claude/CLAUDE.md` with `<!-- cwt-skill:name -->` markers for idempotent updates

## Code Conventions

### Go Style
- Standard Go formatting (tabs, gofmt)
- Error wrapping with `fmt.Errorf("context: %w", err)` — consistent `context: %w` pattern
- Package-level vars for cobra flags and lipgloss styles
- Exported functions use clear verb-noun names: `BuildBranchName`, `ResolveWorktreeDir`, `ResolveBaseBranch`
- Unexported helpers: `expandCommand`, `shellQuote`, `resolvePrompt`, `buildMainCommand`, `skillLoadingPrompt`, `parseSurface`, `extractRepoFromURL`
- No constructor functions for simple structs — use struct literals directly

### Error Handling
- Commands use `RunE` (returns error) rather than `Run`
- Errors are wrapped with context at each layer: `"loading config: %w"`, `"creating worktree: %w"`
- `main.go` calls `os.Exit(1)` on any error from `cmd.Execute()`

### Testing
- Standard library `testing` only — no testify or other frameworks
- Table-driven tests with `[]struct` and `t.Run()` for named subtests
- `t.TempDir()` and `t.Setenv()` for isolated filesystem/env tests
- Config tests override `XDG_CONFIG_HOME` to test Save/Load in isolation
- Tests focus on pure logic (branch naming, config behavior, template rendering) — no integration tests for git/cmux

## Adding a New Command

1. Create `cmd/<name>.go` with a `*cobra.Command` var
2. Register via `func init() { rootCmd.AddCommand(<cmd>) }`
3. Define flags as package-level vars, bind in `init()`
4. Use `RunE` for error-returning commands
5. Load config with `config.Load()` if needed

## Adding a New Skill

1. Create `skills/<skill-name>/SKILL.md`
2. Add `//go:embed <skill-name>/SKILL.md` var in `skills/skills.go`
3. Add entry to `All()` slice with Name, Description, Dir, Content
4. If it's a review skill, add it to `ReviewSkills()` filter
5. Update `skills_test.go` — increment expected count and add name to `expected` map

## Gotchas

- **Version injection**: `cmd/version.go` vars (`version`, `commit`, `date`) are set via `-ldflags` at build time. Running `go run .` shows `dev`/`none`/`unknown`. Use `make build` for real version info.
- **cmux dependency**: Most of the spawn flow requires a running cmux instance. Without it, `cwt spawn` will fail at workspace creation. Tests don't exercise this path.
- **git worktree location**: Default worktree directory is `../worktrees/` relative to repo root (via `git.DefaultWorktreeBase()`). This can be overridden in config with `defaults.worktree_dir`.
- **Auto-init**: `cwt spawn` and `cwt review` automatically trigger `cwt init` if no config file exists.
- **Template variables**: Pane commands and prompt templates use `{{worktree_dir}}` and other mustache-style variables — these are simple string replacements, not a template engine.
- **Shell quoting**: `shellQuote()` in `workspace/spawn.go` uses single-quote wrapping with escaped internal quotes — be aware when constructing commands with special characters.
- **Sleep timers**: `workspace.Spawn` uses `time.Sleep(300ms)` delays between cmux operations to allow the terminal to settle. These are not configurable.
- **No interfaces/mocking**: git and cmux packages use direct `exec.Command` calls with no interface abstraction. To unit test code that calls these, isolate pure logic into separate functions (as done with `BuildBranchName`, `ResolveBaseBranch`, etc.).

## Dependencies

| Dependency | Purpose |
|---|---|
| `spf13/cobra` | CLI framework |
| `charmbracelet/bubbletea` | TUI framework (spawn progress) |
| `charmbracelet/bubbles` | TUI components (spinner) |
| `charmbracelet/huh` | Form/wizard framework (init command) |
| `charmbracelet/lipgloss` | Terminal styling |
| `pelletier/go-toml/v2` | TOML config parsing |
