---
description: Daily standup — review active plans, surface live handoffs, pick what to work on
argument-hint: [migrate]
---

# /Standup

You are running a daily standup for the user. The user is the agency AND the client (solo operator on an e-commerce site). This is NOT a client-facing report — it is a working punch list to refresh memory and immediately start the next piece of work.

## Subcommand routing

- If `$ARGUMENTS` contains `migrate` → execute the **MIGRATION FLOW** below.
- Otherwise → execute the **STANDUP FLOW**.

---

## Namespace & schema (shared with /CreateHandoff)

All planning artifacts live under `.agents/atlas/` in the project root:

```
.agents/atlas/
├── plans/
│   ├── *.md              # active plans
│   └── archive/*.md      # completed plans (status=Complete)
├── handoffs/
│   ├── *.md              # active handoffs (one-time briefings, not yet picked up)
│   └── archive/*.md      # consumed handoffs (picked up via /Standup)
└── …                     # everything else from the legacy .atlas-plans tree
```

### Plan frontmatter (required)

```yaml
---
title: Human-readable plan name
status: Not Started | In Progress | Blocked | In Review | Approved | Complete
percent_complete: 0-100
last_updated: YYYY-MM-DD
phase: Phase 2 — Schema Augmentation # OR `epic:` — pick whichever fits
blockers: short text, or "none"
next_action: one-line description
---
```

### Handoff frontmatter (required)

```yaml
---
title: Human-readable handoff name
parent_plan: filename-of-the-plan.md # references .agents/atlas/plans/<file>
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

1. Check that `.agents/atlas/` exists in the working directory. If only `.atlas-plans/` or `.atlas/plans/` exists, tell the user: _"Legacy `.atlas-plans/` `.atlas/plans/` directory detected. Run `/Standup migrate` first."_ and stop.
2. Ensure `.agents/atlas/plans/`, `.agents/atlas/plans/archive/`, `.agents/atlas/handoffs/`, `.agents/atlas/handoffs/archive/` all exist (create silently if missing).

### Step 2 — Inventory

1. Read frontmatter from every `.agents/atlas/plans/*.md` (top-level only, not `archive/`).
2. Read frontmatter from every `.agents/atlas/handoffs/*.md` (top-level only).
3. If any plan or handoff is missing required frontmatter, list them at the top of your output as **"Schema gaps"** with the field(s) missing. Do NOT auto-fix — the user will decide.

### Step 3 — Render the standup board

Output a single compact table grouped by status, in this order: `In Progress`, `Blocked`, `In Review`, `Approved`, `Not Started`. Skip `Complete` (those should already be archived; if any slipped through, list under a "⚠ should be archived" section and offer to move them).

For each plan show: title · status · % · last_updated · next_action (truncate next_action to ~70 chars).

### Step 4 — Drill into active work

For every plan with status `In Progress` or `Blocked`:

1. Write a 2-3 sentence **plan refresher** (read the plan body, not just frontmatter — pull the goal/scope).
2. List its **outstanding tasks** (unchecked checkboxes, open sections, or whatever the plan uses).
3. Find handoffs where `parent_plan` matches this plan's filename. For each active handoff: 1-line summary + next_action + blockers.
4. If a plan has no matching handoff but has open tasks, note: _"No active handoff — consider `/CreateHandoff`."_

### Step 5 — Ask what to work on

Present a numbered list of actionable items (active handoffs first, then plans without handoffs). Ask the user which to pick up.

### Step 6 — On selection: archive the handoff and begin

When the user picks a handoff:

1. **Move** the handoff file from `.agents/atlas/handoffs/<name>.md` → `.agents/atlas/handoffs/archive/<name>.md` using `mv` (do not copy + delete; preserve the file).
2. Update the parent plan's `last_updated` to today's date.
3. Run `chown pwsysadmin:pwsysadmin` and `chmod 644` on any file you touched (per project CLAUDE.md).
4. Briefly restate the next_action and start the work.

If the user picks a plan without a handoff, skip the archive step and just begin the work.

### Step 7 — On plan completion

If the user signals (or you determine) that a plan has reached `Complete` during the session:

1. Update its frontmatter `status: Complete`, `percent_complete: 100`, `last_updated: <today>`.
2. **Move** the plan from `.agents/atlas/plans/<name>.md` → `.agents/atlas/plans/archive/<name>.md`.
3. Fix ownership/perms.
4. Confirm the archive in your end-of-turn summary.

---

## MIGRATION FLOW (`/Standup migrate`)

This is a one-time operation. Be careful — it moves files and rewrites path references.

### Step 1 — Pre-flight

1. Confirm `.atlas-plans/` exists. If not, tell the user nothing to migrate and stop.
2. Confirm `.agents/atlas/` does NOT already exist. If it does, ask the user whether to merge into it or abort.
3. Show the user the proposed move plan BEFORE executing:
   - `.atlas-plans/*HANDOFF*.md` → `.agents/atlas/handoffs/`
   - `.atlas-plans/*.md` (remaining) → `.agents/atlas/plans/`
   - `.atlas-plans/<everything else>` (subdirs, scripts, etc.) → `.agents/atlas/`
4. Wait for explicit user confirmation before any `mv`.

### Step 2 — Execute moves

Use `mkdir -p` to create the new structure, then `mv` everything according to the plan. Preserve the inner structure of subdirectories.

### Step 3 — Backfill frontmatter

For each migrated plan and handoff that lacks the required frontmatter:

1. Read the file.
2. Infer best-guess values: `title` from H1 or filename, `status: In Progress` (safe default), `percent_complete: 50`, `last_updated: <today>`, `next_action: "review and update"`, `blockers: "none"`.
3. For handoffs, prompt the user to identify the `parent_plan` (offer the list of plans).
4. Insert the frontmatter block at the top of the file.

### Step 4 — Rewrite path references

Search the entire project for occurrences of `.atlas-plans` (use Grep across all text files). For each hit:

1. Show the user the file + line.
2. Replace `.atlas-plans/` → `.agents/atlas/` (and update sub-paths if a file moved into `plans/` or `handoffs/`).
3. **Critical files to check explicitly**: `CLAUDE.md`, any `*.sh` scripts (especially `deploy-post.sh`), and the user's memory files at `~/.claude/projects/-home-pwsysadmin-webapps-peptideware/memory/*.md`.

### Step 5 — Permissions

Run `chown pwsysadmin:pwsysadmin` and `chmod 644` recursively on `.agents/atlas/` (use `find` with `-exec` or the project's `fixOwner.sh` helper).

### Step 6 — Verify and report

1. Confirm `.atlas-plans/` is empty, then ask the user before removing it.
2. Print a summary: N plans migrated, N handoffs migrated, N path references rewritten, N files needing manual frontmatter review.

---

## Output style

- Terse. Tables and bullets. No preamble.
- If something is ambiguous, ask one question and wait — don't assume.
- Always finish your standup output with the numbered "what to work on" list so the user can pick immediately.
