# Regroup status check

Date: 2026-10-04

Source: `docs/plans/2026-09-27-0932-feat-harness-publish-plan.md` as committed in `91bae00a` on branch `fix/gap-sync-owned-block`, `git status` of this workspace, the transcript of the grok-4.7 omp tab (Tern MISC tab 3), and `~/.codex/AGENTS.md` (read only). GitHub issues were not read for this check-in.

## Counts

- Active projects: 1
- Blocked tasks: 2
- Stalled tasks: none can be named. There is no earlier regroup file to compare against.

## Changed since the previous regroup

There is no earlier regroup file. This is the first check-in.

## GAP harness publish

### Phase 1: Original publish (U1 to U4)

- U1. Foreign-file backup and quiet overwrite. Done. Percent not stated. The TTY yes path that moves a file to `~/.gap/backups/` is still not exercised.
- U2. Codex destination. Done. Percent not stated. Replaced by U5 for how Codex is written.
- U3. Cursor destination. Done. Percent not stated. A live Cursor session has still not confirmed it loads the `.mdc` file.
- U4. Load notes. Done. Percent not stated. Updated again in `91bae00a`.

### Phase 2: Status review and red team fixes (U5 to U9)

- Q1 and Q2 plan amendments. Done. Percent not stated. R3 now accepts the last-publish ledger, and R4 refuses only the foreign file.
- U5. Codex owned block (D3). Blocked. Percent not stated. Waiting on handling of the existing unmarked GAP text in `~/.codex/AGENTS.md`. The live file's first 46 lines are an old GAP publish that does not match the current payload (digest `3d3c5e11` vs `e7052e7d`). `spliceCodex()` then falls through to `block + existing`, so the first real sync would load the policy twice in Codex.
- U6. Absolute Codex home (D1). Active. Percent not stated. Implemented in `91bae00a`, not pushed.
- U7. Check covers a partial refuse (D2). Active. Percent not stated. Implemented. `--check` from `91bae00a` prints `ok`.
- U8. Primary checkout only (D4). Active. Percent not stated. Implemented: publish follows `origin/HEAD`. Not pushed.
- U9. Atomic ledger write (D5). Active. Percent not stated. Implemented: the ledger is written to a temp file, then renamed.
- D6. A backup keeps bytes, not behavior. Backlogged. Percent not stated. Not in the grok-4.7 tab's task list. Recorded as a known limitation.

### Phase 3: Ship

- Validate and push `fix/gap-sync-owned-block`. Blocked. Percent not stated. The tab reports that No Mistakes sees `gitbutler/workspace`, not the feature branch, so the gate cannot run. Nothing was pushed.
- Run `npm run sync` against the real home. Not started. Waits on your approval, per the tab's instructions.

### Deferred in the plan

- Grok destination if Claude compatibility is turned off. Backlogged.
- Folding reviewed backup text into the policy. Backlogged. It stays a manual edit.

### Outside this commit

- `.gitignore` has one uncommitted line. The tab deliberately left it out of the commit.

## Open decisions

There is no earlier regroup file, so no open decision is on record from a previous check-in.
