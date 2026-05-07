---
description: Create a new handoff doc linked to an active plan and phase
argument-hint: [optional title]
---

# /CreateHandoff

Create a one-time briefing handoff document under `.atlas/handoffs/`. Handoffs are consumed by `/Standup` — the moment one is picked up, it is archived. So write them to be self-contained: enough context for the next session to start cold.

## Pre-flight
1. Confirm `.atlas/` exists. If only `.atlas-plans/` exists, tell the user to run `/Standup migrate` first and stop.
2. Ensure `.atlas/handoffs/` exists (create if missing).

## Interview the user

Ask these in a single message (not one-at-a-time), then wait for answers:

1. **Parent plan** — present the list of `.atlas/plans/*.md` files (title + filename, exclude `archive/`). Ask which one this handoff belongs to.
2. **Phase / epic** — read the parent plan's frontmatter `phase` field as the default. Ask the user to confirm or override.
3. **Title** — if `$ARGUMENTS` is set, use it as the default; otherwise ask.
4. **Next action** — one line, the very first concrete step.
5. **Blockers** — anything blocking, or "none".
6. **Context body** — ask what should go in the body: scope, files touched, decisions made, links, anything the future-self needs to start cold. Long-form is fine.

## Generate the file

Slugify the title to `kebab-case.md` and write to `.atlas/handoffs/<slug>.md`:

```yaml
---
title: <title>
parent_plan: <plan-filename.md>
phase: <phase or epic>
status: Not Started
percent_complete: 0
last_updated: <today YYYY-MM-DD>
blockers: <text or "none">
next_action: <one line>
---

# <title>

**Parent plan:** [<plan title>](../plans/<plan-filename.md>) · **Phase:** <phase>

## Context
<body content from the interview>

## Next action
<next_action expanded with any specifics>

## Blockers
<blockers expanded, or "None.">
```

## Post-create
1. `chown pwsysadmin:pwsysadmin` and `chmod 644` the new file.
2. Update the parent plan's `last_updated` to today.
3. Tell the user: *"Handoff created at `.atlas/handoffs/<slug>.md`. Will surface in next `/Standup`."*

## Output style
- Single interview message, then single confirmation message after creation.
- No preamble. No trailing summary beyond the create confirmation.
