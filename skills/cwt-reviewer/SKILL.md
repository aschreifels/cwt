# PR Review — Router

Orchestrate a structured pull request review. Detect languages and domains in the PR, dispatch to the appropriate specialist review skills, and compile a unified report.

## When to Use

- User asks to review a PR (by number, URL, or branch name)
- User asks for feedback on code changes, architecture, or quality
- `cwt review` injects this skill automatically

## Workflow

Execute every step in order. Do not skip steps. Do not ask the user for information you can find yourself.

**CRITICAL: Step tracking is mandatory.** Before starting the review, create a todo list with one item per step. Mark each step in_progress before starting it and completed after finishing it.

### Step 1: Fetch PR Context

The injected prompt includes PR metadata and the list of changed files, but **not** the diff. You must fetch the diff and any other details yourself.

Using the `gh` CLI:

```bash
gh pr view <number> --repo <owner/repo> --json title,body,headRefName,baseRefName,state,author,files,additions,deletions,changedFiles
gh pr diff <number> --repo <owner/repo>
gh api repos/<owner>/<repo>/pulls/<number>/comments
```

If no `--repo` is provided, infer from the current git remote. Note the PR size — large PRs (>500 lines, >20 files) deserve extra scrutiny and a note about reviewability.

Check out the branch locally:
```bash
git fetch origin <headRefName> && git checkout <headRefName>
```

### Step 2: Categorize Changed Files

Sort every changed file into review domains by extension and path:

| Domain | File Patterns | Review Skill |
|--------|---------------|--------------|
| **Go** | `*.go`, `go.mod`, `go.sum` | `cwt-reviewer-go` |
| **TypeScript** | `*.ts`, `*.tsx`, `*.js`, `*.jsx`, `package.json`, `tsconfig.json` | `cwt-reviewer-typescript` |
| **Database** | `*.sql`, `*.prisma`, `migrations/`, `schema/` | `cwt-reviewer-database` |
| **Infrastructure** | `*.tf`, `*.yml`/`*.yaml` in `.github/`, `Dockerfile*`, `docker-compose*`, `k8s/`, `infra/`, `deploy/` | `cwt-reviewer-infra` |
| **Other** | Anything not matched above | Review with general software engineering judgment |

A PR can span multiple domains — load every relevant skill. Files like `go.mod` or `package.json` are reviewed by their language skill, not infra.

### Step 3: Check Existing Reviews

Before starting your analysis:

```bash
gh api repos/<owner>/<repo>/pulls/<number>/reviews
gh api repos/<owner>/<repo>/pulls/<number>/comments
```

- Do not duplicate issues already raised by other reviewers or CI bots.
- If you agree with existing feedback, reference it instead of restating.
- If you disagree, explain why.

### Step 4: Run Domain-Specific Reviews

For each detected domain, follow that skill's full checklist:

- **Go** → follow `cwt-reviewer-go` steps (build, test, lint, idiomatic review, security, performance, patterns)
- **TypeScript** → follow `cwt-reviewer-typescript` steps (typecheck, lint, test, patterns, security, performance)
- **Database** → follow `cwt-reviewer-database` steps (migration safety, index coverage, query performance, backward compatibility)
- **Infrastructure** → follow `cwt-reviewer-infra` steps (resource review, security contexts, supply chain, secrets)

For files that don't match a specific domain, apply general review principles: correctness, error handling, readability, test coverage.

### Step 5: Verify All Factual Claims

**Mandatory. Cannot be skipped.**

Before compiling the final review, scan every finding for factual claims about:

- Software version numbers or release status
- Module/package paths and availability
- Install commands and CLI usage
- API behavior, deprecations, or breaking changes
- Library features and when they were introduced
- Configuration identifiers (model IDs, service names, etc.)

For EACH such claim:

1. Use `agentic_fetch` or `fetch` to check the official source (GitHub releases, official docs, pkg.go.dev, npm registry, cloud provider docs)
2. If confirmed correct, keep it
3. If wrong or unverifiable, **silently remove it** — do not post incorrect information

Never tell a PR author their dependency version is wrong, their module path is invalid, or their config is incorrect based solely on memory. Training data is stale. Always verify live.

If removing findings changes the numbering, renumber the remaining items.

### Step 6: Compile Review

Organize all findings into this format:

```markdown
# PR #<number> Review: <title>

## Summary
[1-2 sentences: what this PR does and your overall impression]

## What's Good
- [Genuine positive observations — approach, structure, thoroughness, test quality]
- [At least one item. The author put in effort — acknowledge it.]

## Must Fix (Blocking)
Items that could cause downtime, data loss, security issues, broken builds, or incorrect behavior.

### 1. [Issue Title]
**File:** `path/to/file:line`
**Risk:** [What could go wrong in production]
**Suggestion:** [Concrete fix with code example]

## Should Fix (Non-Blocking)
Items that improve reliability, observability, maintainability, or correctness but won't cause immediate harm.

### 1. [Issue Title]
**File:** `path/to/file:line`
**Why:** [Explanation of the concern]
**Suggestion:** [How to fix]

## Consider (Nice to Have)
Style improvements, documentation, consistency, developer experience.

### 1. [Issue Title]
[Brief explanation and suggestion]

## Questions
- [Clarifying questions for the author, if any]

## Verification
- [Which checks ran: build, tests, linters, etc.]
- [Note any checks that were skipped and why]
```

If a severity category has no findings, include it with "No issues found" to confirm it wasn't skipped.

### Step 7: Post or Present

- If the user asks to "post", "submit", or "comment" — hand off to the `cwt-reviewer-comments` skill
- Otherwise, present the review in the conversation for the user to read first
- Always ask before posting — never auto-submit a review to GitHub

## Tone Guidelines (All Domains)

- **Positivity first.** Always find something genuinely good to call out.
- **Frame as suggestions.** "Have you considered..." or "one thing that might help..." — not "you need to..." or "this is wrong."
- **Explain the why.** Don't just say "add an index" — explain what breaks without it and under what conditions.
- **Provide code examples.** When suggesting a change, include a concrete snippet they can use.
- **Don't pile on.** Group small nits. Don't leave 15 separate comments for formatting.
- **Pick your battles.** Focus on correctness, safety, performance, data integrity. Not everything needs to be perfect.
- **Respect existing patterns.** Even if suboptimal, consistency matters. Note tech debt as "consider for a future PR."
- **Never block on style.** If it works, matches existing code, and passes formatters/linters, it's fine.
- **Be explicit about severity.** Clearly label blocking vs. suggestion. Don't leave the author guessing.
