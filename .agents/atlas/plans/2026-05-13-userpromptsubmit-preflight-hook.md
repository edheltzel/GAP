# UserPromptSubmit Preflight Hook

**Status:** draft
**Author:** Atlas (with Ed)
**Date:** 2026-05-13

## Goal

Force a 4-question preflight check to fire on every user prompt I see, so the Karpathy guideline of "Think before coding → ask if unclear" becomes structurally unavoidable rather than aspirational.

This is the enforcement leg of the rule-compliance discussion on 2026-05-13. The companion memory entry is `~/.claude/projects/-Users-ed-Developer-atlas-recall/memory/feedback_rule_compliance.md`.

## Success criteria

- Every prompt I receive is prefixed (server-side, before I see it) with a 4-line preflight checklist.
- The checklist is short enough that it does not dominate the prompt context.
- The hook fires on every prompt — no skip conditions, no opt-out — because skip logic is exactly where discipline erodes.
- Verification: submit a trivial prompt ("hello") and confirm the preflight appears in the next session's transcript / debug output.

## Mechanism

Claude Code supports `UserPromptSubmit` hooks. They:

- Receive the prompt as JSON on stdin.
- Output stdout, which is prepended to the prompt as additional context Claude sees.
- Are registered in `~/.claude/settings.json` under `hooks.UserPromptSubmit`.

The hook here is intentionally **dumb**: a single bash script that echoes a fixed preamble. No logic, no conditions, no I/O beyond stdout. The whole point is that it cannot be circumvented by clever skip rules.

## Hook script

**Location (source of truth):** `~/Developer/atlas-config/claudecode/.claude/hooks/atlas-preflight.sh`
**Deployed location:** `~/.claude/hooks/atlas-preflight.sh` (copy of source, independent inode)

**Why copy, not symlink.** Recall is a distributable tool for any human × any coding agent. Future hook features Ed wants to add will follow Recall's distribution model — users won't have a stable `atlas-config` repo to symlink against. Copying matches that pattern, keeps the deployed hook resilient to source-tree moves, and is consistent with how the existing Recall hooks (`SessionExtract.ts`, `SessionRecall.ts`, etc.) are already deployed to `~/.claude/hooks/`.

```bash
#!/usr/bin/env bash
# atlas-preflight.sh — UserPromptSubmit hook
#
# Injects a 4-question preflight checklist into every prompt Atlas sees,
# so Karpathy Rule 1 (Think before coding → ask if unclear) is enforced
# structurally instead of aspirationally.
#
# See: ~/.claude/Rules/coding-guidelines.md
# Memory: feedback_rule_compliance.md

# Discard stdin (the prompt JSON); we don't need it.
cat >/dev/null

cat <<'EOF'
[atlas-preflight] Before responding, internally answer:
  1. Which reading of this request am I picking? Are there others equally defensible?
  2. Do I have enough to answer, or do I need to ask one clarifying question first?
  3. What is my one-line verdict? (That sentence opens the response.)
  4. Am I about to produce more than was asked (lists, tables, roadmaps, sections)? If yes, cut.

If #2 is yes, ask. Do not produce artifacts.
EOF
```

## settings.json registration

Append to the existing `hooks` block in `~/.claude/settings.json`:

```json
"UserPromptSubmit": [
  {
    "matcher": "",
    "hooks": [
      {
        "type": "command",
        "command": "/Users/ed/.claude/hooks/atlas-preflight.sh"
      }
    ]
  }
]
```

No `timeout` needed — the hook does nothing but echo a static string.

## Implementation steps

1. Write `~/Developer/atlas-config/claudecode/.claude/hooks/atlas-preflight.sh` with content above. `chmod +x`.
2. Copy to `~/.claude/hooks/atlas-preflight.sh` (`cp` — independent file, no symlink). `chmod +x` on the deployed copy too.
3. Add the `UserPromptSubmit` block to `~/.claude/settings.json`.
4. Verify with a fresh session: send "hello", check that the preflight appears in the prompt context Atlas receives.
5. Commit the hook + settings change to atlas-config.

## Rollback

If the hook becomes noise rather than signal:

- Remove the `UserPromptSubmit` block from `~/.claude/settings.json`.
- The script can stay on disk; without registration it does nothing.

## Resolved

- **Deploy mechanism:** copy, not symlink. Reason: Recall is a distributable tool; future features must work on user machines without an atlas-config checkout. (Ed, 2026-05-13.)

## Still open

- **Preflight visibility.** Does the prepended text appear in the visible transcript, or is it system-only context Atlas sees? Affects whether Ed reads the preflight every turn alongside Atlas. Test on first run; tune if noisy.
- **Cross-harness parity.** Whether an equivalent preflight should exist for OpenCode and Pi. Discipline failures span harnesses; the hook does not. Defer until the Claude Code version has proved its worth.
- **First-review checkpoint.** Recommended 2026-06-13 — one month after install — to assess whether the preflight is doing real work or has become wallpaper.
