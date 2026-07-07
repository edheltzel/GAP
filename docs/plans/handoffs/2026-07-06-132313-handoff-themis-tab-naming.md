# Handoff — Themis GH#<issue>-<role> tab naming (merged)

**Date:** 2026-07-06
**Status:** Done — merged to master

## What happened

Ed asked for a herdr tab-naming convention on Themis: any tab hosting a dispatched agent (worker/reviewer/explorer) tied to a numbered issue gets labeled `GH#<issue>-<role>` (e.g. `GH#21-worker`). No issue number → short descriptive label.

Encoded identically across all three harnesses:

- `claude/.claude/commands/Themis.md` — Dispatch Surface section, new "Tab naming" paragraph
- `packages/pi-themis/extensions/themis.ts` + `skills/themis-pm/SKILL.md` + `AGENTS.md`
- `packages/omp-themis/src/main.ts` + `skills/themis-pm/SKILL.md` + `AGENTS.md`

Convention: label at creation (`herdr tab create --label`) or via `herdr tab rename` immediately after a split lands an agent in an existing tab.

Also archived the legacy Claude Themis subagent (`claude/.claude/agents/Themis.md` → `claude/.claude/_archived/Themis.md`, 100% rename) — superseded by the `/Themis` slash command which runs in the main session.

Shipped as **PR #35** (`edheltzel/atlas-config`), merged as commit `667553f`. Five commits total; the last three were unrelated housekeeping swept in at Ed's direction (untrack `claude/.claude/settings.json` + statusline test, pi voice-adapter pointer update to `atlasEcho`, skills-store sync for `impeccable`/`tdd`/issue-tracker refs).

## Repo-memory note

`atlas-config/AGENTS.md` was updated (by Ed or a linter, not this session) to move project-local Atlas artifacts from `.agents/atlas/` to `docs/` (`plans/`, `handoffs/`, `artifacts/`, `worktrees/`), gitignored as of 2026-06-30. This handoff file itself lands under the new `docs/plans/handoffs/` location per that shift — the older global rule pointing at `.agents/atlas/handoffs/` is stale and should be updated next time that file is touched.

## Open thread

Ed asked to create a new issue: **install PAI/LifeOS v6** (tracked separately — see issue filed same session).

## Nothing blocked

No open questions, no in-flight agents, working tree was clean before this handoff was written.
