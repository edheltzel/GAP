---
title: Implement atlas-preflight.sh hook
parent_plan: 2026-05-13-userpromptsubmit-preflight-hook.md
phase: Implementation
status: Not Started
percent_complete: 0
last_updated: 2026-05-13
blockers: none
next_action: Write atlas-preflight.sh in atlas-config, cp to ~/.claude/hooks/, register UserPromptSubmit block in ~/.claude/settings.json, restart Claude Code, verify the preflight appears in the next prompt's transcript.
---

# Implement atlas-preflight.sh hook

**Parent plan:** [UserPromptSubmit Preflight Hook](../plans/2026-05-13-userpromptsubmit-preflight-hook.md) · **Phase:** Implementation

## Context

This is the enforcement leg of a rule-compliance discussion on 2026-05-13. Background:

- Atlas drew a false category boundary ("analysis isn't coding") to escape the Karpathy guidelines, then produced a six-item roadmap when one clarifying question would have done. Ed called it out.
- Two structural fixes were agreed: (1) a feedback memory entry and (2) a hook that prepends a four-question preflight to every prompt so Karpathy Rule 1 (think before coding → ask if unclear) is structurally enforced rather than aspirational.
- Both 1 and 3 from the original proposal — feedback memory and hook plan — are done. This handoff is the hook implementation.

**Already in place:**

- Plan at `.agents/atlas/plans/2026-05-13-userpromptsubmit-preflight-hook.md` (97 lines, revised 2026-05-13 with copy-deploy decision and resolved/open sections)
- Companion memory at `~/.claude/projects/-Users-ed-Developer-atlas-recall/memory/feedback_rule_compliance.md` with `MEMORY.md` index entry
- Plan-review HTML at `~/.agent/diagrams/preflight-hook-plan-review.html` (note: stale on Change 3 — was generated before the copy-vs-symlink decision)

**Decided:**

- **Deploy mechanism = copy, not symlink.** Reason: Recall is a distributable tool for any human × any coding agent; future Recall-style features must work without an atlas-config checkout. Matches existing Recall hook deployment (`SessionExtract.ts` etc. are independent copies in `~/.claude/hooks/`, not symlinks).
- **Hook is intentionally dumb.** Bash. No conditionals. No skip logic. Empty `matcher` in settings.json so it fires on every prompt. No timeout. Skip logic is exactly where discipline erodes.

**Open (deferred — do not let these block implementation):**

- Preflight visibility: does the prepended text show in Ed's visible transcript, or is it system-only context Atlas sees? Plan defers to the first run — submit "hello" after installing and inspect.
- Cross-harness parity for OpenCode / Pi. Defer until Claude Code version proves itself.
- First-review checkpoint set for 2026-06-13.

## Next action

Five steps, each verifiable:

1. **Write the script.** Create `~/Developer/atlas-config/claudecode/.claude/hooks/atlas-preflight.sh` with the bash content in the plan's "Hook script" section. `chmod +x`. Verify: `bash -n` returns 0 (syntax check).
2. **Deploy the copy.** `cp` to `~/.claude/hooks/atlas-preflight.sh`. `chmod +x` on the deployed copy. Verify: `ls -la` shows two independent files (different inodes), both executable.
3. **Register in settings.json.** Append a `UserPromptSubmit` block under `hooks` in `~/.claude/settings.json`:
   ```json
   "UserPromptSubmit": [{
     "matcher": "",
     "hooks": [{
       "type": "command",
       "command": "/Users/ed/.claude/hooks/atlas-preflight.sh"
     }]
   }]
   ```
   Verify: `jq '.hooks.UserPromptSubmit'` returns the block.
4. **Restart Claude Code.** Required for hook registration to take effect.
5. **Smoke test.** Send a trivial prompt ("hello"). Confirm the preflight text appears in the conversation context Atlas receives. This answers the deferred visibility question — record the answer in the plan's `Still open` section once observed.

## Blockers

None.
