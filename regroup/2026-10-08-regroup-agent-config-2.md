# Regroup: agent-config (GAP harness publish)

Date: 2026-10-08 (second check-in today)

Source: `origin/master` history (#63-#65), `but-axi status`, the live ledger and Codex files under `~/` (read only), and the earlier file `regroup/2026-10-08-regroup-agent-config.md`.

## Last completed

- 2026-10-08: #65 merged. Legacy cleanup records a `cleaned` marker in the ledger, and refuses when the ledger has no entry instead of reporting success.
- 2026-10-08: #64 merged and synced. Codex subagents get GAP through a SubagentStart hook, and the hook is installed in `~/.codex/hooks.json`.
- 2026-10-08: #63 merged. `chore/project-records` landed.
- 2026-10-08: Sync published #58-#64 to all five harnesses from `master`.

## Up next

- Trust both GAP hooks (SessionStart and SubagentStart) in Codex with `/hooks`. Blocked: waits on you in a live Codex session.
- Confirm that a Codex session and a subagent both get the policy. Blocked: waits on the hook trust.
- Run `npm run sync -- --remove-codex-legacy` from `master`. Blocked: waits on that confirmation. The ledger entry (`d0a29072`) still matches the GAP prefix of `~/.codex/AGENTS.md`.

## Remaining

- `chore/regroup-2026-10-08`: committed, not pushed.
- `backup/gap-sync-owned-block-d9026722`: local branch, waiting on your OK to delete.
- Red team finding 7 (`--check` does not run the main sync loop): not started.
- Red team finding 8 (rollback after cleanup brings back the replace prompt): backlogged.
- Cursor loading `global-agent-policy.mdc`: not confirmed by a live Cursor session.
- D6 (a backup keeps the file's bytes but not the harness behavior): backlogged.
