---
name: review-handoff
description: Resume work from a handoff doc in .atlas-plans/handoffs/ (defaults to newest), then archive it. Use as the first thing in a fresh session after /clear. Invoke with /skill:review-handoff [optional-filename].
allowed-tools: Read, Bash(fd:*), Bash(ls:*), Bash(mv:*), Bash(mkdir:*), Bash(git:*)
---

# Review Handoff

Read a handoff document created by `/skill:create-handoff` and continue the work it describes. After loading, automatically move the handoff file to `.atlas-plans/handoffs/archive/` so only active handoffs remain in the main directory.

Arguments are appended as user input after this skill loads. If the user supplied text after `/skill:review-handoff`, treat it as an optional filename or full path to a specific handoff. Otherwise default to the newest file.

## Behavior

1. **Resolve the target file:**
   - If the user supplied a bare filename (e.g., `2026-04-18-handoff-foo.md`), target `.atlas-plans/handoffs/<filename>`.
   - If the user supplied a full path, use it as-is.
   - If no argument was supplied, pick the newest file in `.atlas-plans/handoffs/` by mtime:
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
   Do this immediately after reading so subsequent `/skill:create-handoff` calls don't collide with stale handoffs.

5. **Continue the work.** Using "Next" as the action list and "Open Questions" as the first things to clarify with Ed, pick up where the prior session left off. If blockers exist, ask Ed about them before doing any implementation work.

## Notes

- Archiving on read (not on completion) is intentional: if the session ends without finishing, a new `/skill:create-handoff` writes a fresh file rather than merging into the old one.
- Archived handoffs remain accessible at `.atlas-plans/handoffs/archive/`.
- This skill is meant to be the **first** thing invoked in a fresh session after `/clear`. Don't take other actions before loading the handoff context.
