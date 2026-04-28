---
description: Daily standup — review active plans, surface live handoffs, pick what to work on
argument-hint: [list-plans]
---

# /Standup

You are running a daily standup for the user. This is NOT a client-facing report — it is a working punch list to refresh memory and immediately start the next piece of work.

## Subcommand routing

- `$ARGUMENTS` contains `list-plans` → **LIST-PLANS FLOW**
- Otherwise → **STANDUP FLOW**

---

## Namespace & schema (shared with /CreateHandoff)

All planning artifacts live under `.atlas/` in the project root:

```
.atlas/
├── plans/
│   ├── *.md              # active plans
│   └── archive/*.md      # completed plans (status=Complete)
├── handoffs/
│   ├── *.md              # active handoffs (one-time briefings, not yet picked up)
│   └── archive/*.md      # consumed handoffs (picked up via /Standup)
└── …                     # everything else (audits, content, scripts, etc.)
```

### Plan frontmatter (required)

```yaml
---
title: Human-readable plan name
status: Not Started | In Progress | Blocked | In Review | Approved | Complete
percent_complete: 0-100
last_updated: YYYY-MM-DD
phase: Phase 2 — Schema Augmentation     # OR `epic:` — pick whichever fits
blockers: short text, or "none"
next_action: one-line description
---
```

### Handoff frontmatter (required)

```yaml
---
title: Human-readable handoff name
parent_plan: filename-of-the-plan.md     # references .atlas/plans/<file>
phase: matches the plan's phase or epic
status: Not Started | In Progress | Blocked | In Review | Approved | Complete
percent_complete: 0-100
last_updated: YYYY-MM-DD
blockers: short text, or "none"
next_action: one-line description
---
```

### Status taxonomy (PM/marketing flow, terminal state = "Complete")

`Not Started → In Progress → Blocked → In Review → Approved → Complete`

---

## STANDUP FLOW

### Step 1 — Verify namespace
1. Ensure `.atlas/plans/`, `.atlas/plans/archive/`, `.atlas/handoffs/`, `.atlas/handoffs/archive/` all exist (create silently if missing).

### Step 2 — Inventory
1. Read frontmatter from every `.atlas/plans/*.md` (top-level only, not `archive/`).
2. Read frontmatter from every `.atlas/handoffs/*.md` (top-level only).
3. If any plan or handoff is missing required frontmatter, list them at the top of your output as **"Schema gaps"** with the field(s) missing. Do NOT auto-fix — the user will decide.

### Status color convention

Render each status as a color-coded emoji pill for at-a-glance scannability. Use this canonical mapping in every table, heading, and listing that shows a status:

| Status | Pill | Meaning |
|---|---|---|
| In Progress | 🟢 `In Progress` | Actively being worked |
| Blocked | 🔴 `Blocked` | Stuck on a dependency |
| In Review | 🟡 `In Review` | Awaiting review/approval |
| Approved | 🔵 `Approved` | Approved, not yet started next phase |
| Not Started | ⚪ `Not Started` | Planned but untouched |
| Complete | ✅ `Complete` | Done (should be archived) |

> **Bash output blocks only:** ANSI escape codes can produce true background colors, e.g. `printf '\033[42;30m In Progress \033[0m'` for green bg + black fg. Use this only inside fenced `bash` blocks; the emoji pills are the canonical rendering everywhere else.

### Step 3 — Render the standup board
Output a single compact table grouped by status, in this order: 🟢 `In Progress`, 🔴 `Blocked`, 🟡 `In Review`, 🔵 `Approved`, ⚪ `Not Started`. Skip `Complete` (those should already be archived; if any slipped through, list under a "⚠ should be archived" section with the ✅ `Complete` pill and offer to move them).

For each plan show: title · status pill · % · last_updated · next_action (truncate next_action to ~70 chars).

### Step 4 — Drill into active work
For every plan with status 🟢 `In Progress` or 🔴 `Blocked`:
1. Write a 2-3 sentence **plan refresher** (read the plan body, not just frontmatter — pull the goal/scope).
2. List its **outstanding tasks** (unchecked checkboxes, open sections, or whatever the plan uses).
3. Find handoffs where `parent_plan` matches this plan's filename. For each active handoff: 1-line summary + next_action + blockers.
4. If a plan has no matching handoff but has open tasks, note: *"No active handoff — consider `/CreateHandoff`."*

### Step 5 — Ask what to work on
Present a numbered list of actionable items (active handoffs first, then plans without handoffs). Ask the user which to pick up.

### Step 6 — On selection: archive the handoff and begin
When the user picks a handoff:
1. **Move** the handoff file from `.atlas/handoffs/<name>.md` → `.atlas/handoffs/archive/<name>.md` using `mv` (do not copy + delete; preserve the file).
2. Update the parent plan's `last_updated` to today's date.
3. If the project has a CLAUDE.md ownership/permissions convention (check it), apply it to any file you touched. Otherwise skip.
4. Briefly restate the next_action and start the work.

If the user picks a plan without a handoff, skip the archive step and just begin the work.

### Step 7 — On plan completion
If the user signals (or you determine) that a plan has reached `Complete` during the session:
1. Update its frontmatter `status: Complete`, `percent_complete: 100`, `last_updated: <today>`.
2. **Move** the plan from `.atlas/plans/<name>.md` → `.atlas/plans/archive/<name>.md`.
3. Apply project ownership/permissions convention if one exists.
4. Confirm the archive in your end-of-turn summary.

---

## LIST-PLANS FLOW (`/Standup list-plans`)

A quick read-only listing. No interview, no drill-down.

### Step 1 — Active plans
```bash
eza -1 .atlas/plans/*.md 2>/dev/null || echo "No active plans."
```

### Step 2 — Archive count
```bash
eza -1 .atlas/plans/archive/*.md 2>/dev/null | wc -l | xargs echo "Archived plans:"
```

### Step 3 — Format
```
## Active Plans
- <filename> — <status pill> <title from frontmatter if available>

## Handoffs (active)
- <filename>

## Archive
<N> archived plans · <M> archived handoffs
```

Stop after printing — do not enter the standup drill-down flow unless the user asks.

---

## Output style
- Terse. Tables and bullets. No preamble.
- If something is ambiguous, ask one question and wait — don't assume.
- Always finish your standup output with the numbered "what to work on" list so the user can pick immediately.
