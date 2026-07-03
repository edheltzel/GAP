---
name: atlas:Continue
description: Pick the next task from the GitHub Projects board — grouped by status, ordered by priority and dependencies. Wraps the pm-tools skill's Continue recipe.
argument-hint: [optional — focus area, or "suggest"]
---

Use the **pm-tools** skill and follow its `references/Continue.md` recipe to surface board state and recommend the next task.

$ARGUMENTS

Report open items grouped **In Progress → Ready → Backlog**, then recommend the next task by priority (P0 → P1 → P2 → P3), lower issue number first, skipping issues whose native `blockedBy` dependencies are still open.

Scope: this reads the **board** only. Session resumption and conversational context are owned by Recall + the SessionStart hook — don't duplicate them here.
