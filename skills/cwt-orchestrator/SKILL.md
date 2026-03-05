# cmux Worktree Orchestrator

You have access to `cwt` (cmux Worktree Tool) via bash, a Go CLI that creates git worktrees with full cmux dev environments (AI agent + lazygit + editor). Use this skill to analyze Linear projects or epics and spawn parallelized worktrees — each with its own AI agent instance that auto-receives the ticket context.

## Prerequisites

- `cwt` binary installed (typically at `/opt/homebrew/bin/cwt`)
- `cmux` installed and running
- Linear MCP tools available
- Inside a git repository
- Configuration set up via `cwt init` (auto-triggers on first spawn if missing)

## Workflow: Analyze a Linear Project and Spawn Worktrees

When the user asks you to work on a Linear project, initiative, or set of tickets:

### 1. Fetch and Analyze

- Use Linear MCP tools to fetch the project, initiative, or individual issues
- Read all ticket titles, descriptions, comments, acceptance criteria, and linked documents
- Identify dependencies between tickets (which must be done first, which can be parallelized)
- Map tickets to areas of the codebase by searching for relevant files

### 2. Build an Execution Plan

Present a plan as a TODO list with:
- **Ticket grouping**: Which tickets can be worked on in parallel vs sequentially
- **Dependency graph**: Which tickets block others
- **Worktree mapping**: One worktree per independent ticket or tightly-coupled ticket group
- **Risk assessment**: Merge conflicts likely between parallel branches, shared files, etc.

### 3. Wait for User Confirmation

**Always wait for the user to review and approve the plan before spawning worktrees.**

### 4. Spawn Parallel Worktrees

After confirmation, use `cwt spawn` to create worktrees. Each worktree gets its own AI agent session with the Linear ticket context automatically injected.

```bash
# Spawn a worktree for an existing ticket
cwt spawn <feature-name> -t <TICKET-ID>

# Spawn a worktree and create a draft ticket
cwt spawn <feature-name> --draft

# With explicit base branch
cwt spawn <feature-name> -t <TICKET-ID> -b <base-branch>

# Use an existing branch instead of creating a new one
cwt spawn <feature-name> --existing
cwt spawn <feature-name> --branch <full-branch-name>
```

### 5. Monitor Progress

Use cmux to check on worktree status:
```bash
# List all workspaces to see active worktrees
cmux list-workspaces

# Read a specific workspace's screen to check agent progress
cmux read-screen --workspace <id>

# Send a message to a specific agent instance
cmux send --workspace <id> "status update please"
```

## cwt Command Reference

```
cwt spawn <name> [flags]

Flags:
  -b, --base <branch>       Base branch (auto-detects main/master)
  --branch <branch>         Checkout existing branch
  --existing                Use existing branch matching <name>
  --no-editor               Skip launching editor/tools
  -t, --ticket <TICKET>     Ticket ID to fetch (e.g. PROJ-123)
  -d, --draft               Create a draft ticket for this worktree

cwt rm <name> [flags]
  -D                        Also delete the branch

cwt list                    List active worktrees (alias: ls)
cwt init                    Guided config wizard (branch prefix, provider, layout, etc.)
```

## Configuration

cwt stores config at `~/.config/cwt/config.toml`. Run `cwt init` for guided setup, or edit directly.

Key settings:
- **branch_prefix**: Prepended to branch names (e.g. "jd" → `jd/PROJ-123_feature`)
- **provider**: Project management integration (`linear`, `github`, `jira`, `none`)
- **default_project**: Default project key for ticket lookups
- **prompt templates**: Customizable templates for agent context injection with `{{provider}}`, `{{ticket}}`, `{{project}}`, `{{name}}` variables
- **layout panes**: Configurable pane layout (agent, lazygit, editor positions)

## Rules

- **Never spawn worktrees without user confirmation** of the execution plan
- Keep worktree names short and descriptive (kebab-case)
- Warn about potential merge conflicts between parallel worktrees touching the same files
- Suggest sequential ordering when tickets have hard dependencies
- Use cmux notifications to alert the user when all worktrees are spawned
- When monitoring, summarize status across all worktrees concisely

## Example Interaction

**User**: "Work on the PROJ-100 initiative from Linear"

**You**:
1. Fetch PROJ-100 and all child issues via Linear MCP
2. Analyze the codebase to map each ticket to affected files
3. Present a plan:
   - Group A (parallel): PROJ-101 (api changes), PROJ-102 (ui changes) — no shared files
   - Group B (sequential, after A): PROJ-103 (integration tests) — depends on both
4. Wait for confirmation
5. Spawn: `cwt spawn api-changes -t PROJ-101` and `cwt spawn ui-changes -t PROJ-102`
6. After both complete, spawn: `cwt spawn integration-tests -t PROJ-103`
