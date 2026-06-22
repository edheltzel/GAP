---
name: atlas:Status
description: Board progress report from GitHub Projects — phase progress, status counts, recently-closed, and top up-next items. Wraps the github-projects skill's Status recipe.
argument-hint: [optional — phase or filter]
---

Use the **github-projects** skill and follow its `references/Status.md` recipe to produce a board progress report.

$ARGUMENTS

Report phase progress (closed vs open per phase), board status counts (Backlog / Ready / In Progress / Done), issues closed in the last week, and the top 3 up-next items by priority. Keep it scannable — bullets and a small table, no prose. This is board reporting, distinct from `Recall:stats` (the memory DB).
