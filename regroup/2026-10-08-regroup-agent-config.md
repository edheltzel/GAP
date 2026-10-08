# Regroup: agent-config (GAP harness publish)

Date: 2026-10-08

Source: `origin/master` history (#57-#62), the open PR list on GitHub (none open), `but-axi status`, the live harness files under `~/` (read only), and the red team findings posted on PR #58. The previous file, `regroup/2026-10-04-regroup-status-check.md`, is an all-projects check-in, not a one-project file.

## Last completed

- 2026-10-08: #62 merged. Legacy Codex cleanup refuses a symlinked `hooks.json`.
- 2026-10-08: #61 merged. Sync refuses when `global/` differs from `origin/HEAD`, and refuses cleanly when `origin/HEAD` is missing.
- 2026-10-08: #60 merged. Backups are copied before the write, and a symlinked `hooks.json` is refused.
- 2026-10-08: #59 merged. Legacy cleanup requires that this run published the Codex policy file.
- 2026-10-08: #58 merged. Codex gets the policy through a SessionStart hook; legacy cleanup sits behind an explicit flag.
- 2026-10-08: #57 merged. Global policy requires `master` as the default branch for new repos. Published to all five harnesses.

## Up next

- Run `npm run sync` from `master` to publish #58-#62. Blocked: waits on you to check out `master` in this folder. As of this check, `~/.codex/gap/global-agent-policy.md` does not exist, the GAP hook is not in `hooks.json`, and the ledger was last written 2026-10-07 20:48.
- Trust the GAP hook in Codex with `/hooks`. Blocked: waits on the sync.

## Remaining

- `--remove-codex-legacy`: backlogged. Held until Codex subagents get GAP; SessionStart does not reach them.
- Red team finding 6 (a second cleanup run fails with a misleading error): backlogged. Matters only once the flag is used.
- Red team finding 7 (`--check` does not run the main sync loop): not started.
- Red team finding 8 (rolling back after cleanup brings back the replace prompt): backlogged. Matters only once the flag is used.
- `chore/project-records` (decision log, 2026-10-04 regroup file, `.empryo` ignore): pushed, no PR opened.
- `backup/gap-sync-owned-block-d9026722` local branch: old version of the Codex work, kept for safety. Delete when satisfied.
- Cursor load of `global-agent-policy.mdc`: not confirmed by a live Cursor session (plan, Known limits).
- D6, a backup keeps bytes, not behavior: backlogged (plan).
