# Status — board progress report

Produce a progress report across the board. Adapted from `status.md`. This is **board reporting** — distinct from `Recall:stats` (which reports the memory database, not project status).

## 1. Gather

```bash
# open issues with labels
gh issue list --repo OWNER/REPO --state open --json number,title,labels,state --limit 50

# board items with custom fields
gh project item-list PROJECT_NUM --owner OWNER --format json

# recently closed
gh issue list --repo OWNER/REPO --state closed --json number,title,closedAt --limit 10

# recent activity
git log --oneline -15
```

## 2. Compute

- Issues per Phase: closed vs open.
- Count per board Status column (Backlog / Ready / In Progress / Done).
- High-priority items not yet started.

## 3. Report (scannable, no prose)

**Phase progress**
- Phase N: X/Y complete (list phases with open issues)

**Board status**

| Status | Count | Issues |
|--------|-------|--------|
| In Progress | N | #X, #Y |
| Ready | N | #X, #Y |
| Backlog | N | … |

**Recently completed** — issues closed in the last week.

**Up next (suggested)** — top 3 from Ready/Backlog by priority (use `Continue.md`'s ordering).

**Blockers / decisions needed** — anything flagged blocked or awaiting input.
