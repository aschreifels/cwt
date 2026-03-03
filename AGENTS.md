# AGENTS.md — cwt (Crush Worktree Tool)

## What This Project Is

`cwt` is a Go CLI that creates git worktrees with full [cmux](https://github.com/charmbracelet/cmux) dev environments. Each worktree gets its own Crush AI assistant, git TUI, and editor in a multi-pane workspace. It supports ticket integration (Linear, GitHub Issues, Jira) and bundles installable Crush skills.

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
  rm.go                          # cwt rm — removes worktrees, optional branch delete
  skills.go                      # cwt skills list/install — manages Crush skills
  version.go                     # cwt version — prints version info (injected via ldflags)
internal/
  config/
    config.go                    # Config types, Load/Save (TOML), template rendering
    config_test.go               # Thorough unit tests for config
  workspace/
    spawn.go                     # Core spawn logic: git worktree + cmux workspace creation
    spawn_test.go                # Unit tests for branch naming, dir resolution, prompts
  git/
    git.go                       # Git operations via exec.Command (worktree add/remove/list, branch ops)
  cmux/
    client.go                    # cmux CLI wrapper (workspaces, splits, send, status, progress, notify)
  tui/
    spawn.go                     # Bubble Tea TUI for spawn progress (spinners, per-pane status)
skills/
  skills.go                      # Skill registry using //go:embed
  skills_test.go                 # Verifies all embedded skills exist and have content
  cmux-notifications/SKILL.md    # Skill: teaches Crush to use cmux sidebar APIs
  cwt-orchestrator/SKILL.md      # Skill: teaches Crush to orchestrate parallel worktrees
```

## Architecture and Key Patterns

### Cobra CLI Structure
- Each command is defined in `cmd/` as a package-level `*cobra.Command` var
- Commands register themselves via `func init()` calling `rootCmd.AddCommand()`
- Flags are defined as package-level vars and bound in `init()`
- `cmd.Execute()` is the sole public API from the `cmd` package

### Config System (`internal/config`)
- TOML config at `~/.config/cwt/config.toml` (respects `XDG_CONFIG_HOME`)
- `Config` struct with `Defaults`, `Layout`, and `ProjectManagement` sections
- `DefaultConfig()` provides sensible defaults (3 panes: crush, lazygit, helix)
- `Load()` merges file config with defaults — empty prompts and empty pane lists are backfilled
- Template rendering via `strings.NewReplacer` with `{{provider}}`, `{{ticket}}`, `{{project}}`, `{{name}}`, `{{worktree_dir}}`
- Helper methods on `Config`: `EnabledPanes()`, `MainPane()`, `SidePanes()`, `HasProjectManagement()`, `RenderPrompt()`

### External Tool Integration
- **git**: All git operations in `internal/git/git.go` via `exec.Command` — no git library
- **cmux**: All cmux operations in `internal/cmux/client.go` via `exec.Command("cmux", ...)` — wraps the CLI
- Both packages parse CLI output (porcelain format for git, `OK <id>` for cmux)
- No mocking of these — tests that need git/cmux are either pure-logic unit tests or require the real tools

### Spawn Flow (`workspace.Spawn`)
1. Build branch name (prefix + ticket + name)
2. Resolve worktree directory (config override or `../worktrees/<name>`)
3. Create git worktree (or reuse existing)
4. Create cmux workspace with main pane
5. Create right/bottom splits for side panes
6. Inject ticket prompt into Crush via `cmux send-text`
7. Set cmux sidebar status (branch, base, ticket, provider)
- Progress is reported via `chan StepUpdate` → Bubble Tea model

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
- `skills.All()` returns the full registry
- Installation copies embedded content to `~/.config/crush/skills/<dir>/SKILL.md`

## Code Conventions

### Go Style
- Standard Go formatting (tabs, gofmt)
- Error wrapping with `fmt.Errorf("context: %w", err)` — consistent `context: %w` pattern
- Package-level vars for cobra flags and lipgloss styles
- Exported functions use clear verb-noun names: `BuildBranchName`, `ResolveWorktreeDir`, `ResolveBaseBranch`
- Unexported helpers: `expandCommand`, `shellQuote`, `resolvePrompt`, `parseSurface`
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
4. Update `skills_test.go` — increment expected count and add name to `expected` map

## Gotchas

- **Version injection**: `cmd/version.go` vars (`version`, `commit`, `date`) are set via `-ldflags` at build time. Running `go run .` shows `dev`/`none`/`unknown`. Use `make build` for real version info.
- **cmux dependency**: Most of the spawn flow requires a running cmux instance. Without it, `cwt spawn` will fail at workspace creation. Tests don't exercise this path.
- **git worktree location**: Default worktree directory is `../worktrees/` relative to repo root (via `git.DefaultWorktreeBase()`). This can be overridden in config with `defaults.worktree_dir`.
- **Auto-init**: `cwt spawn` automatically triggers `cwt init` if no config file exists (`cmd/spawn.go:46`).
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
