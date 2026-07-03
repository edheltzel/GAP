# Continue — pick the next task from board state

Decide what to work on next, sourced from the board. Adapted from `continue.md` — **board-prioritization half only.** Session resumption and conversational context are owned by Recall + the SessionStart hook, not this recipe; don't duplicate them here.

## 1. Gather board state

```bash
# open issues with labels + native blocked-by dependencies (gh >= 2.94)
gh issue list --repo OWNER/REPO --state open \
  --json number,title,labels,state,blockedBy --limit 50

# board items with their field values (Phase, Priority, Status)
gh project item-list PROJECT_NUM --owner OWNER --format json
```

## 2. Report, grouped by status

Present a table grouped **In Progress → Ready → Backlog** (the default `solo-software` vocabulary — use the board archetype's own status names if it differs):

| # | Issue | Phase | Priority | Status | Labels |
|---|-------|-------|----------|--------|--------|

Then summarize: what's In Progress, what's Ready to pick up, any blockers/decisions needed.

## 3. Work-order priority

When choosing the next task (or suggesting one):

1. Anything already **In Progress** on the board comes first.
2. Then **Priority** field: P0 → P1 → P2 → P3 (the canonical scheme; map accordingly if the board uses another).
3. Within the same priority, **lower issue number first** (earlier work before later).
4. Skip issues only while any native `blockedBy` dependency is still open (a node with `state == "OPEN"` — gh keeps closed blockers in the list, so non-empty alone doesn't mean blocked).
5. **The board is the source of truth** — if board and memory disagree, the board wins for *status*.

## Boundary

This recipe answers "what's next on the board." It does **not** read git history, scan the repo, or rebuild conversational context — for that use `Recall:scout` (repo orientation) or `Recall` recall/search (prior sessions). Keep the concerns separate.
