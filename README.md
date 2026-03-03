# cwt — Crush Worktree Tool

`cwt` creates [git worktrees](https://git-scm.com/docs/git-worktree) with full [cmux](https://github.com/charmbracelet/cmux) dev environments — each with its own [Crush](https://github.com/charmbracelet/crush) AI assistant, git TUI, and editor, all in one workspace.

```
┌──────────────┬───────────┐
│              │  lazygit  │
│    crush     ├───────────┤
│              │  helix .  │
└──────────────┴───────────┘
```

## Features

- **One command** to create a worktree + cmux workspace with configurable panes
- **Ticket integration** — fetch Linear/GitHub/Jira tickets and inject context into Crush automatically
- **Draft mode** — create new tickets on the fly and track work incrementally
- **Guided setup** — `cwt init` walks you through configuration via a TUI wizard
- **Bundled Crush skills** — install skills that teach Crush to use cmux notifications and orchestrate parallel worktrees
- **Beautiful TUI** — animated boot sequence with per-pane status spinners

## Prerequisites

- [cmux](https://github.com/charmbracelet/cmux) installed and running
- [Crush](https://github.com/charmbracelet/crush) installed
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
# First-time setup (or run cwt spawn — it auto-triggers init)
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
| `cwt skills list` | List bundled Crush skills |
| `cwt skills install` | Install Crush skills to `~/.config/crush/skills/` |
| `cwt version` | Print version info |

### `cwt spawn` flags

| Flag | Description |
|---|---|
| `-b, --base <branch>` | Base branch (auto-detects `main`/`master`) |
| `--branch <branch>` | Checkout an existing branch |
| `--existing` | Use existing branch matching `<name>` |
| `-t, --ticket <ID>` | Fetch a ticket and seed Crush with its context |
| `-d, --draft` | Create a draft ticket for this worktree |
| `--no-editor` | Skip launching programs |

## Configuration

Config lives at `~/.config/cwt/config.toml`. Run `cwt init` for guided setup, or edit directly.

### Example config

```toml
[defaults]
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
position = "main"

[[layout.panes]]
name = "lazygit"
command = "lazygit"
position = "right"

[[layout.panes]]
name = "editor"
command = "hx ."
position = "bottom-right"
```

### Key settings

| Setting | Description |
|---|---|
| `defaults.branch_prefix` | Prepended to branch names (e.g. `jd` → `jd/PROJ-123_feature`) |
| `defaults.base_branch` | Override auto-detected base branch |
| `defaults.worktree_dir` | Override default worktree location |
| `project_management.provider` | `linear`, `github`, `jira`, or `none` |
| `project_management.default_project` | Default project key for draft tickets |
| `layout.panes` | Array of pane configs with `name`, `command`, `position`, `disabled` |

### Template variables

Prompt templates and pane commands support these variables:

| Variable | Expands to |
|---|---|
| `{{provider}}` | Project management provider name |
| `{{ticket}}` | Ticket ID (e.g. `PROJ-123`) |
| `{{project}}` | Default project key |
| `{{name}}` | Worktree name |
| `{{worktree_dir}}` | Absolute path to the worktree |

## Crush Skills

`cwt` bundles two [Crush skills](https://charm.sh/crush/) that enhance the AI assistant experience:

- **cmux-notifications** — Teaches Crush to use cmux sidebar APIs (status bars, progress, log, toast notifications) to keep you informed
- **cwt-orchestrator** — Teaches Crush to analyze project tickets, build execution plans with dependency graphs, and spawn parallel worktrees

### Install skills

```bash
# Install all bundled skills
cwt skills install

# Install a specific skill
cwt skills install cmux-notifications

# Overwrite existing skills
cwt skills install --force
```

Skills are installed to `~/.config/crush/skills/`. Ensure your `~/.config/crush/crush.json` includes:

```json
{
  "options": {
    "skills_paths": ["~/.config/crush/skills"]
  }
}
```

## How It Works

1. `cwt spawn` creates a git worktree branched from your base branch
2. A new cmux workspace is created with your configured panes (Crush, git tool, editor)
3. Each pane is launched with `cd` into the worktree directory
4. If `--ticket` is provided, Crush receives the ticket context as its first prompt
5. If `--draft` is provided, Crush is instructed to create and maintain a ticket as it works
6. cmux sidebar status is set with branch, base, and ticket info

## License

[MIT](LICENSE)
