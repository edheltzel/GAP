# Regroup: GAP

Date: 2026-10-29
Source: `todo.txt` via `tuxedo lsa --json` (no `done.txt` lines), `git log origin/master` since 2026-10-22, and `docs/plans/2026-10-20-gap-env-provisioner-plan.md`. Resume point: no active handoff (`docs/plans/handoffs/` holds only `archive/`). Previous file: `regroup/2026-10-22-regroup-gap.md`.

## Last completed

- 2026-10-29 Vendored skills removed (`0b66377`, PR #103): `skills/`, `docs/global`, `docs/project`, and `scripts/update-readme.sh` are gone. This is plan section 6 step 4. Skills install from the pinned Do-Skills store. Git only; no todo.txt task.
- 2026-10-27 README images redrawn with Blip (`d92ee4b`, PR #102), and the installer sets `origin/HEAD` after its shallow clone (`62e24e5`). Git only; no todo.txt task.
- 2026-10-24 Store status labels fixed, and installer build output is ignored without hiding tracked edits. `report:cmd/gap/store_status_test.go`.
- 2026-10-23 Provisioner and breadcrumb ledger published through No Mistakes (PRs #99, #100). These close the two gaps the previous file named: the Recall setup breadcrumb and Echo adapters in `state.json`.

## Up next

Up next not specified. The one open task has no priority or due date.

## Remaining

- `2026-10-27 Decide jcode support: gap sync could publish to ~/.jcode/prompt-overlay.md (jcode reads ~/AGENTS.md, which GAP keeps as a pointer) +gap`: not started.

Seen this session but not tracked as tasks:

- `plannotator-compound`, `plannotator-setup-goal`, and `plannotator-visual-explainer` keep drifting. Something strips `disable-model-invocation: true` from the installed `SKILL.md` (last on 2026-10-28 10:32), so `gap sync` exits 1.
- Plan section 9 steps 4 and 5 are still open while `global/skills.json` externals install through `npx skills add`: the Do-Skills README install text, and dropping the README's Node note.
