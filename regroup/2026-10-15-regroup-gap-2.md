# Regroup: GAP

Date: 2026-10-15

Sources: `todo.txt` and `done.txt` via `tuxedo lsa --json`, the `+gap` tasks in `~/todo.txt`, the newest handoff `docs/plans/handoffs/2026-10-15-gap-skills.md`, and the previous file `regroup/2026-10-15-regroup-gap.md`.

Resume point: `docs/plans/handoffs/2026-10-15-gap-skills.md`.

## Last completed

- `x 2026-10-15 2026-10-15 Fix untracked global/skills.json merge-gate bypass + regression test, amend into suw +gap note:docs/plans/handoffs/2026-10-15-gap-skills.md`: amended into `suw` (`feat/gap-skills`). It is not pushed. A follow-up line in `cmd/gap/main_test.go`, `commitMerged("restore manifest")`, is still uncommitted.
- `x 2026-10-14 2026-10-14 Merge README tutorial PR +gap pr:https://github.com/edheltzel/agent-config/pull/72`

## Up next

- `(A) 2026-09-30 ATL-623 GAP rename: Global Agent Policies in README +gap @intern @review gate:pr id:ATL-623 pr:https://github.com/edheltzel/agent-config/pull/75`: stalled. The previous regroup says #75 merged, but the task is still open.
- `(C) 2026-09-28 ATL-613 gap: Codex-in-Herdr review then Red-Team +gap @intern @todo id:ATL-613 pr:https://github.com/edheltzel/agent-config/pull/76`: stalled. The Codex-in-Herdr review has not run.

## Remaining

- `2026-10-15 Decide GitButler vs No Mistakes publish path, add phase-agent exception, publish docs/no-mistakes-autonomous +gap note:docs/plans/handoffs/2026-10-15-no-mistakes-autonomous-policy.md`: not started; it needs the user's choice.
- `2026-10-15 Decide U8 public history (fresh repo / filter-repo / full) and give the go +gap note:docs/plans/handoffs/2026-10-15-gap-skills.md`: not started; it needs the user's choice.
- `2026-10-15 Decide whether to abort stale no-mistakes run 01M4CAEW62RRW7GN17D3RZ8F8T +gap note:docs/plans/handoffs/2026-10-15-no-mistakes-autonomous-policy.md`: not started.
- `2026-10-15 Plan gap skills (A+B) with do-plan +gap note:/Users/ed/Developer/Atlas/GAP/docs/plans/2026-10-15-gap-skills-requirements.md`: not started per todo.txt. The handoff says `docs/plans/2026-10-15-gap-skills-plan.md` exists.
- `2026-10-15 Restore the lost DoSkills --yes fix (fix/no-mistakes-yes discarded 2026-10-15 04:31) or re-apply after the merge +gap note:/Users/ed/Developer/Atlas/GAP/docs/plans/2026-10-15-gap-skills-requirements.md`: not started per todo.txt. The handoff says DoSkills `fix/no-mistakes-yes` (`mop`) holds U1.
- `2026-10-15 U2 gitleaks scan of agent-config and DoSkills history +gap note:/Users/ed/Developer/Atlas/GAP/docs/plans/2026-10-15-gap-skills-plan.md`: not started per todo.txt. The handoff says the findings are in `docs/plans/2026-10-15-secrets-scan.md`.
- `2026-10-15 U3-U7 build gap skills (manifest, install, drift, docs) +gap note:/Users/ed/Developer/Atlas/GAP/docs/plans/2026-10-15-gap-skills-plan.md`: not started per todo.txt. The handoff says it is built in `suw`, which is unpushed.
- `2026-10-15 U8 publish: flip agent-config public, rename to GAP, archive DoSkills (needs explicit go) +gap note:/Users/ed/Developer/Atlas/GAP/docs/plans/2026-10-15-gap-skills-plan.md`: blocked on the user's go and the U8 history decision.
- `2026-10-15 Write requirements for (C) gap installs harnesses +gap note:/Users/ed/Developer/Atlas/GAP/docs/plans/2026-10-15-gap-skills-requirements.md`: not started.
- `2026-10-14 After #74 merged: go install ./cmd/gap, then gap sync from a terminal +gap`: stalled.
- `2026-10-14 Rename edheltzel/agent-config to GAP; archive/rename old edheltzel/GAP Go scaffold repo so go install path resolves +gap`: stalled.
- `2026-10-14 Red-team 8: rollback after --remove-codex-legacy brings back the replace prompt +gap`: stalled.
- `2026-10-14 Confirm in a live Cursor session that ~/.cursor/rules/global-agent-policy.mdc loads +gap`: stalled.
- `2026-10-14 D6: a foreign-file backup keeps bytes, not the harness behavior it gave +gap`: stalled.
