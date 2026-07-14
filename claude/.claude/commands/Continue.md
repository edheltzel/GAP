---
name: atlas:Continue
description: Pick the next task — board state plus local plans/handoffs, reconciled against GitHub. Wraps the pm-tools skill's Continue recipe and adds a plan/handoff review pass.
argument-hint: [optional — focus area, or "suggest"]
---

Surface what's next by combining three sources: the GitHub Projects **board**, local **plans and handoffs**, and a **reconciliation** of the two.

$ARGUMENTS

## 1. Board state

Use the **pm-tools** skill and follow its `references/Continue.md` recipe to gather and report board state. Report open items grouped **In Progress → Ready → Backlog**, ordered by priority (P0 → P1 → P2 → P3), lower issue number first, skipping issues whose native `blockedBy` dependencies are still open. Keep the `gh issue list` / `gh project item-list` results — the reconciliation pass reuses them.

## 2. Review local plans & handoffs

Read plan and handoff docs from both documented locations (the two drift; check both):

- `.agents/atlas/plans/` and `.agents/atlas/handoffs/` (per AGENTS.md — gitignored, local-only)
- `docs/plans/` and `docs/plans/handoffs/` (per the `handoff.md` rule — where tracked handoffs currently land)

For each, note its **Status** and any **Open thread** / blocked section. A handoff marked `Done — merged #PR` is closed; one with an unfinished open thread is live work.

## 3. Reconcile against GitHub

Cross-reference each live plan/handoff thread against the board data from step 1 (add `gh pr list --repo OWNER/REPO --state all` for handoffs that cite a PR). Flag mismatches:

- **Untracked work** — an open thread in a plan/handoff with no matching open issue.
- **Stale local** — a plan/handoff still marked live whose issue/PR is already closed or merged.
- **Board-only** — an open issue with no plan/handoff context.

## 4. Recommend next task

Draw the recommendation from **both** the board and unfinished handoff threads, using the board's work-order priority. The board wins for *status* when board and a local doc disagree.

## Boundary

The pm-tools recipe stays board-only by design — this command composes on top of it; do not push the plan/handoff logic upstream. Session resumption and conversational context stay owned by Recall + the SessionStart hook.
