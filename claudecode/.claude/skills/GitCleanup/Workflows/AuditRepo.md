# AuditRepo Workflow

Full sweep report of a repository's cleanup state — branches, worktrees, PRs, and issues.

## Voice Notification

```bash
curl -s -X POST http://localhost:8888/notify \
  -H "Content-Type: application/json" \
  -d '{"message": "Running the AuditRepo workflow in the GitCleanup skill to audit repository state"}' \
  > /dev/null 2>&1 &
```

Running the **AuditRepo** workflow in the **GitCleanup** skill to audit repository state...

---

## Step 1: Gather Local State

Run all local git commands in parallel mentally, then compile:

### Worktrees
```bash
git worktree list
git worktree prune --dry-run
```

### Local branches
```bash
# All local branches with tracking info
git branch -vv

# Merged into main/master
git branch --merged main 2>/dev/null || git branch --merged master

# Branches with gone upstream
git branch -vv | grep '\[.*: gone\]'
```

### Stash
```bash
git stash list
```

### Current state
```bash
git status --short
git log --oneline -5
```

## Step 2: Fetch and Check Remote State

```bash
git fetch --prune --dry-run 2>&1 | head -20
```

### Remote branches (after fetch)
```bash
git branch -r | grep -v HEAD

# Remote branches merged into default
git branch -r --merged origin/main 2>/dev/null \
  | grep -v 'HEAD\|main\|master\|develop\|release\|hotfix'
```

## Step 3: Query GitHub (if available)

```bash
# Check gh auth
gh auth status 2>/dev/null && GH_AVAILABLE=true || GH_AVAILABLE=false
```

If `gh` is available:

### Open PRs
```bash
gh pr list --state open --json number,title,author,updatedAt,isDraft,reviewDecision \
  --template '{{range .}}#{{.number}} {{.title}} [{{.reviewDecision}}] @{{.author.login}} — {{.updatedAt}}{{"\n"}}{{end}}'
```

### Open Issues (optional)
```bash
gh issue list --state open --json number,title,assignees,updatedAt,labels \
  | head -20
```

### Stale PRs (no activity in 30+ days)
```bash
gh pr list --state open --json number,title,updatedAt \
  --jq '.[] | select(.updatedAt < (now - 2592000 | todate)) | "#\(.number) \(.title)"'
```

## Step 4: Compile Audit Report

Output a structured, scannable report:

```
╔══════════════════════════════════════════════════════╗
║           GitCleanup — Repository Audit              ║
║           {repo} · {date}                            ║
╚══════════════════════════════════════════════════════╝

WORKTREES  ─────────────────────────────────────────────
  Active:    3 worktrees
  Prunable:  1 (path no longer exists)
  ⚠️  .claude/worktrees/old-feature  [branch: merged into main]

LOCAL BRANCHES  ────────────────────────────────────────
  Total:     8 branches
  Merged:    3 (safe to delete)
    - feature/add-auth     [merged 2 days ago]
    - fix/login-bug        [merged 1 week ago]
    - chore/deps           [merged 3 weeks ago]
  Gone upstream: 1
    - worktree-old-thing   [upstream deleted]
  Active:    4 (keep)

REMOTE BRANCHES (origin)  ──────────────────────────────
  Total:     12 branches
  Merged:    5 (safe to delete)
    - feature/add-auth, fix/login-bug, ...
  Protected: main, develop (skip)
  Active:    7 (keep)

PULL REQUESTS  ─────────────────────────────────────────
  Open:      6
  Approved, ready to merge:  1  (#42 feat: dark mode)
  Stale (>30 days):          2  (#38, #31)
  Draft:                     1  (#55 wip: dashboard)
  In review:                 2

STASH  ─────────────────────────────────────────────────
  Stashed:   2 entries (review manually)

CLEANUP PRIORITY  ──────────────────────────────────────
  1. Prune worktree:      git worktree prune
  2. Merge ready PR:      gh pr merge 42 --squash
  3. Delete local:        git branch -d feature/add-auth fix/login-bug chore/deps
  4. Delete remote:       git push origin --delete feature/add-auth fix/login-bug
  5. Close stale PRs:     #38, #31 (review first)
  6. Review stash:        git stash list
```

## Step 5: Offer to Proceed

After showing the audit, offer to run specific workflows:

```
What would you like to do?

  [1] SweepLocal  — Remove worktrees + delete merged local branches
  [2] SweepRemote — Delete remote merged branches on GitHub
  [3] ClosePR     — Close/merge the identified PRs
  [all] Run all three in sequence
  [n]  Nothing — just the audit
```

Route to the appropriate workflow based on response.

---

## Output Format

- Always use the structured report format above
- Use visual separators for scannability
- Highlight the cleanup priority list — this is the most actionable section
- If no issues found in a category, show: `✓ Clean`
- If `gh` is not available, note it and skip the PRs/issues sections
