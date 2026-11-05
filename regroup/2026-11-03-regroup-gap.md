# Regroup: GAP

Date: 2026-11-03
Source: `todo.txt` via `TODO_FILE=<repo>/todo.txt DONE_FILE=<repo>/done.txt tuxedo lsa --json` (no `done.txt` exists; 47 completed lines live in `todo.txt`), `git log origin/master` since 2026-10-29, and `docs/plans/2026-10-20-gap-env-provisioner-plan.md` section 9. Resume point: no active handoff (`docs/plans/handoffs/` holds only `archive/`). Previous file: `regroup/2026-10-29-regroup-gap.md`.

## Last completed

- 2026-10-29 Plannotator owns its extra skills (`9c595dfc`). Git only; no todo.txt task.
- 2026-10-29 Vendored skills removed (`0b66377f`, PR #103): `skills/`, `docs/global`, `docs/project`, and `scripts/update-readme.sh` are gone. Plan section 6 step 4. Skills install from the pinned Do-Skills store. Git only; no todo.txt task.
- 2026-10-24 Store status labels fixed, and installer build output is ignored without hiding tracked edits. `report:cmd/gap/store_status_test.go`.

## Up next

Up next not specified. The one open task has no priority or due date.

## Remaining

- `2026-10-27 Decide jcode support: gap sync could publish to ~/.jcode/prompt-overlay.md (jcode reads ~/AGENTS.md, which GAP keeps as a pointer) +gap`: not started. Unchanged since the 2026-10-29 file.

Not tracked as tasks, still open in the plan:

- Plan section 9 step 4: the Do-Skills README still points at `npx skills add edheltzel/Do-Skills`. That README lives in the Do-Skills repo, not here.
- Plan section 9 step 5: GAP README "Manage skills" still documents the `npx skills add` path and the Node requirement (`README.md` lines 74 and 86).
