# ClosePR Workflow

Close or merge stale pull requests and merge requests on GitHub (primary) or GitLab.

## Voice Notification

```bash
curl -s -X POST http://localhost:8888/notify \
  -H "Content-Type: application/json" \
  -d '{"message": "Running the ClosePR workflow in the GitCleanup skill to manage stale pull requests"}' \
  > /dev/null 2>&1 &
```

Running the **ClosePR** workflow in the **GitCleanup** skill to manage stale pull requests...

---

## Step 1: List Open PRs/MRs

### GitHub
```bash
# List all open PRs with metadata
gh pr list --state open --json number,title,author,createdAt,updatedAt,isDraft,labels \
  --template '{{range .}}#{{.number}} {{.title}} | {{.author.login}} | updated: {{.updatedAt}} | draft: {{.isDraft}}{{"\n"}}{{end}}'
```

### GitLab
```bash
glab mr list --state opened
```

## Step 2: Identify Stale PRs

A PR/MR is stale if ANY of:
- Open for > 30 days with no activity (no comments, commits, or review updates)
- Open for > 90 days regardless of activity
- Marked as draft with no commits in > 14 days
- Its branch has been deleted (orphaned PR)
- The author is no longer on the team (check with user)

```bash
# GitHub: find PRs with no activity in 30+ days
gh pr list --state open --json number,title,updatedAt \
  --jq '.[] | select(.updatedAt < (now - 2592000 | todate)) | "#\(.number) \(.title) — last updated: \(.updatedAt)"'
```

## Step 3: Check PR Mergeability

Before closing, check if any stale PRs can be auto-merged:

```bash
gh pr view <number> --json mergeable,mergeStateStatus,reviewDecision
```

**Merge if:** `mergeable == "MERGEABLE"` and `reviewDecision == "APPROVED"`
**Close if:** stale, abandoned, superseded, or no longer relevant

## Step 4: Present Options to User

For each stale PR, show:

```
Stale PRs found:

  #42  feat: add dark mode         | @alice  | 45 days old | APPROVED ✓
       → Can merge (clean, approved)

  #38  fix: broken login on safari | @bob    | 62 days old | CHANGES_REQUESTED
       → Stale draft — close?

  #31  chore: update dependencies  | @alice  | 91 days old | REVIEW_REQUIRED
       → Very old, no reviews — close?

Actions: [m]erge, [c]lose, [s]kip for each
```

## Step 5: Execute

### Merge a PR (GitHub)
```bash
gh pr merge <number> --squash --delete-branch
# or
gh pr merge <number> --merge --delete-branch
# or
gh pr merge <number> --rebase --delete-branch
```

**Merge strategy intent mapping:**

| User says | Flag | When to use |
|-----------|------|-------------|
| "squash" or default | `--squash` | Clean linear history |
| "merge commit" | `--merge` | Preserve all commits |
| "rebase" | `--rebase` | Linear without merge commit |

### Close a PR (GitHub)
```bash
gh pr close <number> --comment "Closing as stale — no activity for X days. Reopen if still relevant."
```

### Delete branch after closing (optional)
```bash
gh pr close <number> --delete-branch
```

### GitLab
```bash
# Merge
glab mr merge <number>

# Close
glab mr close <number>
```

## Step 6: Report

Summarize what was merged and closed:

```
GitCleanup — PR sweep complete:
  Merged:  #42 feat: add dark mode
  Closed:  #38 fix: broken login on safari (stale)
  Closed:  #31 chore: update dependencies (stale)
  Skipped: #55 wip: new dashboard (in progress)
```

---

## Safety Rules

- **Never merge without confirmation** — always show the PR and ask
- **Never force-merge** an unapproved PR
- **Always leave a close comment** explaining why the PR was closed
- **Preserve the branch** by default unless user says to delete it
- Draft PRs: close only if no commits in > 14 days AND user confirms
