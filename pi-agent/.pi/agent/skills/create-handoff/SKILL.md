---
name: create-handoff
description: Create a one-time briefing handoff in .atlas/handoffs/ linked to an active plan and phase, and dump the session to Recall (if installed). Handoffs surface in /skill:standup and are archived on pickup. Invoke with /skill:create-handoff [optional title].
allowed-tools: Read, Write, Edit, Bash(mkdir:*), Bash(mem:*), Bash(command:*), Bash(date:*), Bash(git:*)
---

# Create Handoff

Create a one-time briefing handoff document under `.atlas/handoffs/`. Handoffs are consumed by `/skill:standup` — the moment one is picked up, it is archived. So write them self-contained: enough context for the next session to start cold.

Arguments are appended as user input after this skill loads. If the user supplied text after `/skill:create-handoff`, treat it as a title hint. Otherwise ask during the interview.

## Pre-flight
1. Confirm `.atlas/` exists. If only `.atlas-plans/` exists, tell the user to run `/skill:standup migrate` first and stop.
2. Ensure `.atlas/handoffs/` exists (create if missing).

## Interview the user

Ask these in a single message (not one-at-a-time), then wait:

1. **Parent plan** — list `.atlas/plans/*.md` files (title + filename, exclude `archive/`). Ask which one this handoff belongs to.
2. **Phase / epic** — read the parent plan's frontmatter `phase` field as default. Confirm or override.
3. **Title** — use args if provided; otherwise ask.
4. **Next action** — one line, the very first concrete step.
5. **Blockers** — anything blocking, or "none".
6. **Context body** — scope, files touched, decisions made, links, anything future-self needs to start cold. Long-form OK.

## Generate the file

Slugify the title to `kebab-case.md` and write to `.atlas/handoffs/YYYY-MM-DD-<slug>.md`:

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
<next_action expanded with specifics>

## Blockers
<blockers expanded, or "None.">

## Resume
After `/clear`, run `/skill:standup` — this handoff will surface in the next standup board and be archived on pickup.
```

## Dump session to Recall (conditional)

Check whether the Recall CLI is installed before attempting the dump:

```bash
if command -v mem >/dev/null 2>&1; then
  mem dump "Handoff: <title>" --skip-fabric
else
  echo "Recall CLI (mem) not installed — skipping session dump. Handoff file still created."
fi
```

`--skip-fabric` is used for speed. For exceptionally valuable sessions, omit the flag to capture Fabric wisdom.

Handoff file = source of truth for next session. Recall dump = backstop for ad-hoc details, when available.

## Post-create
1. Apply project CLAUDE.md ownership/permissions convention to the new file if one exists.
2. Update the parent plan's `last_updated` to today.
3. Warn if the git tree is dirty — recommend committing or stashing before `/clear`.
4. Tell the user: *"Handoff created at `.atlas/handoffs/<slug>.md`. Will surface in next `/skill:standup`."* Confirm whether the Recall dump ran or was skipped.

## Output style
- Single interview message, single confirmation message after creation.
- No preamble. No trailing summary beyond the create confirmation.
