# cwt — cmux Worktree Tool

`cwt` creates [git worktrees](https://git-scm.com/docs/git-worktree) with full [cmux](https://github.com/charmbracelet/cmux) dev environments — each with its own AI coding agent, git TUI, and editor, all in one workspace. Supports both [Crush](https://github.com/charmbracelet/crush) and [Claude Code](https://docs.anthropic.com/en/docs/claude-code).

```
┌──────────────┬───────────┐
│              │  lazygit  │
│ crush/claude ├───────────┤
│              │  helix .  │
└──────────────┴───────────┘
```

## Features

- **Agent-agnostic** — choose between Crush or Claude Code as your AI assistant
- **One command** to create a worktree + cmux workspace with configurable panes
- **Ticket integration** — fetch Linear/GitHub/Jira tickets and inject context into your agent automatically
- **Draft mode** — create new tickets on the fly and track work incrementally
- **Guided setup** — `cwt init` walks you through configuration via a TUI wizard
- **Bundled skills** — install skills that teach your agent to use cmux notifications and orchestrate parallel worktrees
- **Beautiful TUI** — animated boot sequence with per-pane status spinners

## Prerequisites

- [cmux](https://github.com/charmbracelet/cmux) installed and running
- [Crush](https://github.com/charmbracelet/crush) and/or [Claude Code](https://docs.anthropic.com/en/docs/claude-code) installed
- [Git](https://git-scm.com/) 2.15+ (worktree support)
- [Go](https://go.dev/) 1.24+ (to build from source)

## Install

### From source

```bash
go install github.com/aschreifels/cwt@latest
```

### Manual build

```bash
git clone https://github.com/aschreifels/cwt.git
cd cwt
make build
make install
```

## Quick Start

```bash
# First-time setup — picks your AI agent, git tool, editor, etc.
cwt init

# Create a worktree with a full dev environment
cwt spawn my-feature

# Create a worktree linked to a ticket
cwt spawn my-feature -t PROJ-123

# Create a worktree with a new draft ticket
cwt spawn my-feature --draft

# List active worktrees
cwt list

# Remove a worktree
cwt rm my-feature

# Remove worktree and delete the branch
cwt rm my-feature -D
```

## Commands

| Command | Description |
|---|---|
| `cwt spawn <name>` | Create a worktree with a full cmux workspace |
| `cwt rm <name>` | Remove a worktree (optionally delete branch with `-D`) |
| `cwt list` | List active worktrees (alias: `ls`) |
| `cwt init` | Guided configuration wizard |
| `cwt skills list` | List bundled agent skills |
| `cwt skills install` | Install agent skills |
| `cwt version` | Print version info |

### `cwt spawn` flags

| Flag | Description |
|---|---|
| `-b, --base <branch>` | Base branch (auto-detects `main`/`master`) |
| `--branch <branch>` | Checkout an existing branch |
| `--existing` | Use existing branch matching `<name>` |
| `-t, --ticket <ID>` | Fetch a ticket and seed your agent with its context |
| `-d, --draft` | Create a draft ticket for this worktree |
| `--no-editor` | Skip launching programs |

## Configuration

Config lives at `~/.config/cwt/config.toml`. Run `cwt init` for guided setup, or edit directly.

### Example config

```toml
[defaults]
agent = "crush"          # "crush" or "claude"
branch_prefix = "jd"
base_branch = ""
worktree_dir = ""

[project_management]
provider = "linear"
default_project = "PROJ"

[project_management.prompts]
fetch = "Fetch the {{provider}} issue {{ticket}}..."
create = "Create a draft {{provider}} issue in project {{project}}..."

[[layout.panes]]
name = "crush"
command = "crush -c {{worktree_dir}}"
split = "main"

[[layout.panes]]
name = "lazygit"
command = "lazygit"
split = "right"

[[layout.panes]]
name = "editor"
command = "hx ."
split = "down"
```

For Claude Code, set `agent = "claude"` and the main pane defaults to:

```toml
[[layout.panes]]
name = "claude"
command = "claude"
split = "main"
```

### Key settings

| Setting | Description |
|---|---|
| `defaults.agent` | AI agent: `crush` (default) or `claude` |
| `defaults.branch_prefix` | Prepended to branch names (e.g. `jd` → `jd/PROJ-123_feature`) |
| `defaults.base_branch` | Override auto-detected base branch |
| `defaults.worktree_dir` | Override default worktree location |
| `project_management.provider` | `linear`, `github`, `jira`, or `none` |
| `project_management.default_project` | Default project key for draft tickets |
| `layout.panes` | Array of pane configs with `name`, `command`, `split`, `disabled` |

### Template variables

Prompt templates and pane commands support these variables:

| Variable | Expands to |
|---|---|
| `{{provider}}` | Project management provider name |
| `{{ticket}}` | Ticket ID (e.g. `PROJ-123`) |
| `{{project}}` | Default project key |
| `{{name}}` | Worktree name |
| `{{worktree_dir}}` | Absolute path to the worktree |

## Agent Skills

`cwt` bundles skills that enhance the AI assistant experience. Skills are agent-aware — they install differently depending on your configured agent.

- **cmux-notifications** — Teaches the agent to use cmux sidebar APIs (status bars, progress, log, toast notifications)
- **cwt-orchestrator** — Teaches the agent to analyze project tickets, build execution plans with dependency graphs, and spawn parallel worktrees

### Install skills

```bash
# Install all bundled skills
cwt skills install

# Install a specific skill
cwt skills install cmux-notifications

# Overwrite existing skills
cwt skills install --force
```

### Where skills go

| Agent | Install location | Format |
|---|---|---|
| Crush | `~/.config/crush/skills/<name>/SKILL.md` | Individual skill files |
| Claude Code | `~/.claude/CLAUDE.md` | Appended with `<!-- cwt-skill:name -->` markers |

For Crush, also add them to your global context in `~/.config/crush/crush.json`:

```jsonc
{
  "options": {
    "skills_paths": [
      "~/.config/crush/skills"
    ],
    "context_paths": [
      "~/.config/crush/skills/cmux-notifications/SKILL.md",
      "~/.config/crush/skills/cwt-orchestrator/SKILL.md"
    ]
  }
}
```

## How It Works

1. `cwt spawn` creates a git worktree branched from your base branch
2. A new cmux workspace is created with your configured panes
3. Each pane is launched with `cd` into the worktree directory
4. **Crush**: ticket context is injected via `cmux send-text` after the agent is ready
5. **Claude Code**: ticket context is passed as a CLI argument (`claude "prompt"`)
6. cmux sidebar status is set with branch, base, and ticket info

## License

[MIT](LICENSE)
