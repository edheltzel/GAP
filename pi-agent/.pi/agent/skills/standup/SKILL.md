---
name: standup
description: Daily standup — review active plans in .atlas/, surface live handoffs, pick what to work on. Subcommands `migrate` (move legacy .atlas-plans/ → .atlas/) and `list-plans` (quick read-only listing). Invoke with /skill:standup [migrate|list-plans].
allowed-tools: Read, Write, Edit, Grep, Bash(mkdir:*), Bash(mv:*), Bash(eza:*), Bash(wc:*), Bash(date:*), Bash(git:*)
---

# Standup

You are running a daily standup for the user. This is NOT a client-facing report — it is a working punch list to refresh memory and immediately start the next piece of work.

Arguments are appended as user input after this skill loads. If the user supplied `migrate` or `list-plans`, route to the matching flow. Otherwise run the default STANDUP FLOW.

## Subcommand routing

- Args contain `migrate` → **MIGRATION FLOW**
- Args contain `list-plans` → **LIST-PLANS FLOW**
- Otherwise → **STANDUP FLOW**

---

## Namespace & schema (shared with /skill:create-handoff)

All planning artifacts live under `.atlas/` in the project root:

```
.atlas/
├── plans/
│   ├── *.md              # active plans
│   └── archive/*.md      # completed plans (status=Complete)
├── handoffs/
│   ├── *.md              # active handoffs (one-time briefings, not yet picked up)
│   └── archive/*.md      # consumed handoffs (picked up via /skill:standup)
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

### Status taxonomy (terminal state = "Complete")

`Not Started → In Progress → Blocked → In Review → Approved → Complete`

---

## STANDUP FLOW

### Step 1 — Verify namespace
1. Check that `.atlas/` exists. If only `.atlas-plans/` exists, tell the user: *"Legacy `.atlas-plans/` directory detected. Run `/skill:standup migrate` first."* and stop.
2. Ensure `.atlas/plans/`, `.atlas/plans/archive/`, `.atlas/handoffs/`, `.atlas/handoffs/archive/` all exist (create silently if missing).

### Step 2 — Inventory
1. Read frontmatter from every `.atlas/plans/*.md` (top-level only, not `archive/`).
2. Read frontmatter from every `.atlas/handoffs/*.md` (top-level only).
3. If any plan or handoff is missing required frontmatter, list them at the top as **"Schema gaps"** with the missing field(s). Do NOT auto-fix.

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
Single compact table grouped by status in this order: 🟢 `In Progress`, 🔴 `Blocked`, 🟡 `In Review`, 🔵 `Approved`, ⚪ `Not Started`. Skip `Complete`; if any slipped through, list under "⚠ should be archived" with the ✅ `Complete` pill and offer to move them.

For each plan: title · status pill · % · last_updated · next_action (truncate to ~70 chars).

### Step 4 — Drill into active work
For every plan with status 🟢 `In Progress` or 🔴 `Blocked`:
1. 2-3 sentence **plan refresher** (read the body, pull the goal/scope).
2. List **outstanding tasks** (unchecked checkboxes, open sections).
3. Match handoffs where `parent_plan` == this plan's filename. 1-line summary + next_action + blockers each.
4. If a plan has open tasks but no handoff, note: *"No active handoff — consider /skill:create-handoff."*

### Step 5 — Ask what to work on
Numbered list — active handoffs first, then plans without handoffs. Ask the user to pick.

### Step 6 — On selection: archive the handoff and begin
When the user picks a handoff:
1. **Move** `.atlas/handoffs/<name>.md` → `.atlas/handoffs/archive/<name>.md` using `mv` (preserve the file, don't copy+delete).
2. Update the parent plan's `last_updated` to today.
3. Apply project ownership/permissions convention from CLAUDE.md if present.
4. Briefly restate the next_action and start the work.

If the user picks a plan without a handoff, skip the archive step.

### Step 7 — On plan completion
If a plan reaches `Complete` during the session:
1. Update frontmatter: `status: Complete`, `percent_complete: 100`, `last_updated: <today>`.
2. **Move** `.atlas/plans/<name>.md` → `.atlas/plans/archive/<name>.md`.
3. Apply project ownership/permissions convention if present.
4. Confirm the archive in your summary.

---

## LIST-PLANS FLOW (`/skill:standup list-plans`)

Read-only listing. No interview, no drill-down.

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

## MIGRATION FLOW (`/skill:standup migrate`)

One-time operation. Be careful — it moves files and rewrites path references.

### Step 1 — Pre-flight
1. Confirm `.atlas-plans/` exists. If not, say nothing to migrate and stop.
2. Confirm `.atlas/` does NOT already exist. If it does, ask whether to merge or abort.
3. Show the proposed move plan BEFORE executing:
   - `.atlas-plans/*HANDOFF*.md` → `.atlas/handoffs/`
   - `.atlas-plans/*.md` (remaining) → `.atlas/plans/`
   - `.atlas-plans/<everything else>` → `.atlas/`
4. Wait for explicit user confirmation before any `mv`.

### Step 2 — Execute moves
`mkdir -p` to create the new structure, then `mv` everything per the plan. Preserve subdirectory structure.

### Step 3 — Backfill frontmatter
For each migrated plan/handoff lacking required frontmatter:
1. Read the file.
2. Infer: `title` from H1 or filename, `status: In Progress` (safe default), `percent_complete: 50`, `last_updated: <today>`, `next_action: "review and update"`, `blockers: "none"`.
3. For handoffs, prompt the user for `parent_plan` (offer the plan list).
4. Insert the frontmatter block at the top.

### Step 4 — Rewrite path references
Grep the project for `.atlas-plans` in all text files. For each hit:
1. Show file + line.
2. Replace `.atlas-plans/` → `.atlas/` (adjust sub-paths if a file moved into `plans/` or `handoffs/`).
3. **Critical files to check explicitly**: `CLAUDE.md`, `AGENTS.md`, any `*.sh` scripts, and project-scoped memory files if any.

### Step 5 — Permissions
Apply CLAUDE.md ownership/permissions convention recursively to `.atlas/` if one exists. Otherwise skip.

### Step 6 — Verify and report
1. Confirm `.atlas-plans/` is empty, then ask before removing.
2. Summary: N plans migrated, N handoffs migrated, N path references rewritten, N files needing manual review.

---

## Output style
- Terse. Tables and bullets. No preamble.
- If ambiguous, ask one question and wait — don't assume.
- Always finish the default standup with a numbered "what to work on" list so the user can pick immediately.
