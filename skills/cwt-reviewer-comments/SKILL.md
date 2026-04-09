# PR Review — Post Comments

Submit review findings as inline GitHub comments on specific diff lines using the `gh` CLI and GitHub Pull Request Reviews API.

## When to Use

- After `cwt-reviewer` (or a domain-specific review skill) produces findings and the user wants them posted to the PR
- User asks to "add comments", "submit the review", "comment on the lines", or "post the review"
- Never auto-submit — only when the user explicitly requests it

## Workflow

### Step 1: Check for Existing Comments

Before doing anything else, check what's already been posted:

```bash
gh api repos/<owner>/<repo>/pulls/<number>/comments --jq '.[].body'
gh api repos/<owner>/<repo>/pulls/<number>/reviews --jq '.[].body'
```

Do not duplicate issues already commented on. If a finding overlaps with an existing comment, skip it.

### Step 2: Gather Data

1. Get owner, repo, PR number from context or the current git remote
2. Get head commit SHA:
   ```bash
   gh api repos/<owner>/<repo>/pulls/<number> --jq '.head.sha'
   ```
3. Get the diff:
   ```bash
   gh pr diff <number> --repo <owner>/<repo>
   ```

### Step 3: Map Findings to Diff Lines

The GitHub API only accepts comments on lines that appear in the diff. For each finding:

1. Locate the target line in a `+` (added) or context (unchanged) line within the diff
2. If the exact line isn't in the diff, use the nearest line in the same hunk
3. If no suitable line exists in the diff for a finding, include it in the top-level review body instead

Use `side: "RIGHT"` for all inline comments (this targets the new version of the file).

### Step 4: Build Payload with Python

Always use a Python script to build the JSON payload. This avoids shell escaping issues with Markdown, code fences, backticks, and quotes in comment bodies.

```python
import json

comments = [
    {
        "path": "pkg/handler.go",
        "line": 42,
        "side": "RIGHT",
        "body": "`userID` comes straight from the query string and hits the database without validation. Worth adding a bounds check or UUID parse before the query.\n\n```suggestion\nuserID, err := uuid.Parse(r.URL.Query().Get(\"id\"))\nif err != nil {\n    http.Error(w, \"invalid id\", http.StatusBadRequest)\n    return\n}\n```"
    },
]

payload = {
    "commit_id": "<sha>",
    "event": "<EVENT>",
    "body": "<review summary>",
    "comments": comments,
}

with open("/tmp/review_payload.json", "w") as f:
    json.dump(payload, f)
```

Then submit:
```bash
gh api repos/<owner>/<repo>/pulls/<number>/reviews \
  --method POST \
  --input /tmp/review_payload.json \
  --jq '.html_url'
```

### Step 5: Handle Errors

- **422 "line not in diff"**: Re-map to the nearest diff line in the same hunk. If no hunk covers the finding, move it to the review body.
- **422 "validation failed"**: Re-fetch head SHA and retry — the branch may have been updated since the review started.
- **403 "not accessible"**: Fall back to a single review comment:
  ```bash
  gh pr comment <number> --repo <owner>/<repo> --body "<formatted findings>"
  ```
- **Rate limiting**: If you hit rate limits, batch remaining comments into the review body.

### Step 6: Confirm Submission

After posting, output the review URL so the user can verify:
```bash
echo "Review posted: <html_url>"
```

## Event Type Selection

Choose the event based on the review findings:

| Condition | Event | Rationale |
|-----------|-------|-----------|
| No "Must Fix (Blocking)" items | `"APPROVE"` | Non-blocking suggestions and nits are fine alongside an approval |
| Has blocking items | `"REQUEST_CHANGES"` | Blocking means blocking — the PR shouldn't merge as-is |
| User explicitly says "just comment, don't block" | `"COMMENT"` | Respect the user's intent even if there are blocking findings |
| User explicitly says "approve it" | `"APPROVE"` | User override takes precedence |

Default to `"REQUEST_CHANGES"` when blocking items exist. This is what "blocking" means. The user can override to `"COMMENT"` if they want to flag issues without hard-gating.

## Code Suggestions

When suggesting a single-line or small multi-line fix, use GitHub's suggestion syntax inside the comment body. This creates a one-click "Apply suggestion" button for the author:

````
```suggestion
newCode := doTheThing(ctx, validated)
```
````

Use suggestions for:
- Simple renames, type changes, or argument additions
- Adding a nil check or error check
- Swapping a function call

Do NOT use suggestions for:
- Large refactors spanning multiple functions
- Changes that require new imports (the suggestion can't add imports)
- Architectural recommendations

For larger changes, use a regular fenced code block showing the recommended approach.

## Comment Voice and Style

Write comments like a peer reviewer who has read the code carefully and is being helpful — not like an automated tool generating a report. Match the user's communication style from conversation history.

**Rules:**

- **No severity titles or labels.** Don't start comments with "Must Fix —", "Should Fix —", "Nit —", or any bolded header. Just talk about the issue directly.
- **Conversational, not formulaic.** Write like you'd talk in a PR review — varied sentence structure, no repeated patterns. Each comment should feel like its own thought, not a template.
- **Lead with the problem, not a category.** Instead of "**Security — SQL injection**" just say what's wrong: "`userID` goes straight into the query here without validation."
- **Keep it tight.** 2-4 sentences explaining the issue is usually enough. Don't over-explain things the author likely understands.
- **Code suggestions are inline.** When suggesting a fix, drop the code block naturally after explaining the issue. No "**Suggestion:**" header.
- **Skip the preamble.** Don't start every comment with "Great work but..." or "Nice job here, however...". Just get to the point.
- **Use "we" and "this" not "you should".** "We should validate this" or "this needs a bounds check" reads better than "you need to add validation here."
- **One idea per comment.** Don't combine unrelated issues. Each comment lives on the line it's about.

**Good example:**
```
`format` is user-provided and goes straight into the SQL string here — something like `text); DROP TABLE users; --` would work. Worth validating against an allow-list.

` ``suggestion
var validFormats = map[string]bool{"text": true, "json": true, "xml": true}
` ``
```

**Bad example:**
```
**Must Fix — SQL injection via unsanitized `format` parameter**

The `format` argument is user-provided and interpolated directly into SQL via `fmt.Sprintf`. A malicious value like `text) DROP TABLE users; --` would produce valid destructive SQL.

**Suggestion:** Validate `format` against an allow-list:
```

The bad example reads like a security scanner. The good example reads like a human who noticed something.

**Overall review body:** Keep the `body` field (the top-level review summary) to 1-2 natural sentences. Mention what you reviewed (e.g. "Reviewed the Go changes and the migration — looks solid, few things inline."). Don't list categories or use bullet points in the summary.

## Multi-Domain Reviews

When a review spans multiple languages or domains (e.g. Go + SQL + Terraform), the overall review body should briefly note which areas were covered:

> "Looked at the API handler changes and the schema migration. Handler looks good, couple thoughts on the migration inline."

Individual comments should stand alone — don't reference "as mentioned in the Go section" since GitHub displays comments per-file, not per-domain.
