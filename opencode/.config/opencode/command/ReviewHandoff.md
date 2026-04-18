---
description: "Resume work from a handoff doc in .atlas-plans/handoffs/ (defaults to newest), then archive it"
---

# Review Handoff

Read a handoff document created by `/CreateHandoff` and continue the work it describes. After loading, automatically move the handoff file to `.atlas-plans/handoffs/archive/` so only active handoffs remain in the main directory.

**Optional filename:** $ARGUMENTS

## Behavior

1. **Resolve the target file:**
   - If `$ARGUMENTS` is a bare filename (e.g., `2026-04-18-handoff-foo.md`), target `.atlas-plans/handoffs/$ARGUMENTS`.
   - If `$ARGUMENTS` is a full path, use it as-is.
   - If `$ARGUMENTS` is empty, pick the newest file in `.atlas-plans/handoffs/` by mtime:
     ```bash
     fd -e md . .atlas-plans/handoffs/ --max-depth 1 -x stat -f '%m %N' | sort -rn | head -1 | cut -d' ' -f2-
     ```
     (Fallback if `fd` is unavailable: `ls -t .atlas-plans/handoffs/*.md 2>/dev/null | head -1`.)
   - If no handoff exists, stop and report: "No handoff found in `.atlas-plans/handoffs/`."

2. **Read the handoff** and present a concise summary (5–10 bullets) covering: Task, Done, Next, Open Questions, Git State.

3. **Confirm git state matches expectations.** Compare the handoff's "Git State" section to current `git status --short` and branch. If they differ, flag it before continuing — the user may be on a different branch or have additional uncommitted changes.

4. **Archive the handoff file** to keep the active directory clean:
   ```bash
   mkdir -p .atlas-plans/handoffs/archive
   mv <handoff-path> .atlas-plans/handoffs/archive/
   ```
   Do this immediately after reading so subsequent `/CreateHandoff` calls don't collide with stale handoffs.

5. **Continue the work.** Using the "Next" section as the action list and "Open Questions" as the first things to clarify with Ed, pick up where the prior session left off. If blockers exist, ask Ed about them before doing any implementation work.

## Notes

- Archiving on read (not on completion) is intentional: if the session ends without finishing, a new `/CreateHandoff` creates a fresh file rather than merging into the old one.
- If Ed wants to revisit an archived handoff, it's still at `.atlas-plans/handoffs/archive/`.
- This command is meant to be the **first** thing run in a fresh session after `/clear`. Don't take other actions before loading the handoff context.
