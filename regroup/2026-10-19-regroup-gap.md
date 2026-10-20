# Regroup: GAP

Date: 2026-10-19
Source: `todo.txt` via `tuxedo ls --json` (no `done.txt` exists), the `+gap` lines in `~/todo.txt`, `gh pr list -R edheltzel/GAP`, a live `gap sync`, and the previous check-in `regroup/2026-10-18-regroup-gap-2.md`. Resume point: no active handoff (`docs/plans/handoffs/` holds only `archive/`); the plan in flight is `docs/plans/2026-10-19-gap-empryo-harness-plan.md`.

## Last completed

- 2026-10-19 Empryo first-class harness (U1-U6) merged as PR #91 (`4a955640`). `gap sync` publishes the policy to `~/.empryo/EMPRYO.md` beside the SoulStack block.
- 2026-10-19 `gap sync` run clean from a terminal: answered yes for `~/.empryo/EMPRYO.md` and `~/.omp/agent/commands/state-docs.md` (backups under `~/.gap/backups/`); the stale `~/.claude/skills/typesafe-ai` was trashed and reinstalled as a link. A second sync changed nothing.
- 2026-10-19 Empryo `skills.autoLoad` cut from 35 skills to `do-todo`, `do-commit`, `but-axi` (global, via the settings tool).
- 2026-10-19 `do-git-worktree` rewritten for GitButler lanes.
- 2026-10-19 Closed as done or replaced: `global/commands` merge-gate block (already in `ignoredPathBlocks`, tested in `cmd/gap/commands_test.go`), the `git push no-mistakes` end-to-end test (replaced by `axi run --branch`), `/state-docs` publish, and the checkpoint reinstall.

## Up next

- U7: in a new Empryo tab, confirm the GAP policy loads once and only the three pins are active, then run `/soulstack update` and recheck that `~/.empryo/EMPRYO.md` still has one gap block.

## Remaining

- U9: offer the No Mistakes changes upstream: needs your explicit go.
- Revoke the Eight Sleep `APP_CLIENT_SECRET` from commit 98ee113c (rotate, no history rewrite): not started.
- Confirm `~/.cursor/rules/global-agent-policy.mdc` loads in a live Cursor session: not started.
- (C) ATL-613 Codex-in-Herdr review then red-team (global list): its PR #76 merged 2026-10-15; still open, likely only needs closing.
- Follow-up, not tracked yet: the No Mistakes review fix `94f2e20` from the aborted run never reached master (it exists only in the gate repo).

## Raw lines

```
2026-10-14 Confirm in a live Cursor session that ~/.cursor/rules/global-agent-policy.mdc loads +gap
2026-10-15 U2: revoke the Eight Sleep APP_CLIENT_SECRET from commit 98ee113c (rotate, no history rewrite) +gap note:docs/plans/2026-10-15-secrets-scan.md
2026-10-17 U9: offer the NM changes upstream (needs explicit go) +gap note:/Users/ed/Developer/Atlas/GAP/docs/plans/2026-10-17-nm-gitbutler-safe-plan.md
2026-10-19 U7: in a new Empryo tab confirm the GAP policy loads once and only do-todo/do-commit/but-axi are pinned, then /soulstack update and recheck rg -c gap:begin ~/.empryo/EMPRYO.md = 1 (PR #91 merged, gap sync clean 2026-10-19) +gap note:/Users/ed/Developer/Atlas/GAP/docs/plans/2026-10-19-gap-empryo-harness-plan.md
(C) 2026-09-28 ATL-613 gap: Codex-in-Herdr review then Red-Team +gap @intern @todo id:ATL-613 pr:https://github.com/edheltzel/agent-config/pull/76
```
