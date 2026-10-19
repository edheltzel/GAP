# Handoff: gap manages skills (A+B built, unpublished)

Date: 2026-10-15

## State

- GAP local branches. Nothing is pushed.
  - `feat/gap-skills` (`suw`) is stacked on `feat/gap-help` (`oqv`). It holds plan units U3-U7: `skills/`, `docs/global|project`, `global/skills.json` (73 skills), `cmd/gap/skills.go` (install via skills.sh, drift check), the `skills/` merge gate, and docs.
  - `docs/no-mistakes-autonomous` (`lsu` policy line, `nyt` handoff, regroup and todos).
- DoSkills local branch `fix/no-mistakes-yes` (`mop`) holds U1, the `--yes` restore.
- U2 scan is in `docs/plans/2026-10-15-secrets-scan.md`. U8 (go public, rename, archive DoSkills) has not started.
- **Known bug, must fix before publishing `suw`.** `readCheckout` (`cmd/gap/main.go:208-210`) checks untracked files only under `skills/`. `readManifest` reads `global/skills.json` from the working tree, so an untracked manifest on a checkout whose origin lacks one can install an unmerged external skill. Fix: include `global/` in the `ls-files --others` check, and add an e2e case with an untracked manifest. The e2e fixture now commits its manifest (`commitMerged`), so the current tests cannot catch this. (Reviewer advisory; I agree.)
- `gap sync` in this checkout refuses until `skills/` is merged. That is expected.
- Read-only probe of the real home (machine-local): all 73 managed skills count as installed, with 0 drift.

## Decisions

- User decisions, from the interview and the build:
  - DoSkills merges into GAP, copied without history.
  - `gap` runs skills.sh.
  - Managed skills are mirrored into Claude.
  - On drift, show the diff and choose per skill.
  - Ignore unmanaged skills.
  - Ship A+B first.
  - Gate `skills/` like `global/`.
  - Keep DoSkills' `REPO_URL` until U8.
  - Reopen the public-history question.
- Mine:
  - Mirror Claude through skills.sh `-a claude-code`. The other agents read `~/.agents/skills` directly.
  - Manifest source `"."` means the checkout `gap` was built from.
  - Drop the stale lock entries `do-red-team` and `typesafe-ai`.
  - Record "keep" in the ledger as `skill:<name>`.

## Open questions

- Public history for U8: a fresh public repo, filter-repo, or full history? The scan found pi transcripts and two real-looking values.
- GitButler vs No Mistakes publish path: allow `git push no-mistakes <branch>`, skip the gate, or hold?
- Abort the stale no-mistakes run `01M4CAEW62RRW7GN17D3RZ8F8T`?
- ATL-613's Codex-in-Herdr review still has not run.

## Next step

Fix the untracked-manifest gate bypass, with a failing test first, and amend the fix into `suw`.

## Links

- `docs/plans/2026-10-15-gap-skills-requirements.md`: the interview outcome.
- `docs/plans/2026-10-15-gap-skills-plan.md`: units U1-U8 and blockers. Its U3 and U8 notes record the build-time changes.
- `docs/plans/2026-10-15-secrets-scan.md`: U2 findings.
- `docs/plans/handoffs/archive/2026-10-15-no-mistakes-autonomous-policy.md`: the GitButler/No Mistakes blocker.
- `cmd/gap/skills.go`: install, drift, `treeHash`.
- `cmd/gap/main.go:202-232`: the merge gate.

## todo.txt

- Fix the untracked global/skills.json merge-gate bypass, add a regression test, amend into suw
- Decide U8 public history (fresh repo / filter-repo / full) and give the go
