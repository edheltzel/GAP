# Regroup: GAP

Date: 2026-10-21
Source: `todo.txt` via `tuxedo lsa --json` (no `done.txt` exists), `docs/plans/2026-10-20-gap-env-provisioner-plan.md`, and the tree on `master` at `ef5ab1fd`. Resume point: no active handoff (`docs/plans/handoffs/` holds only `archive/`); the plan in flight is `docs/plans/2026-10-20-gap-env-provisioner-plan.md`.

## Last completed

- 2026-10-20 M1 proof: throwaway HOME sync then restore byte-identical for 67 skills, second sync a no-op; after merge `gap sync` + `gap status` zero drift on BigMac. Merged as PR #96 (`4697a0af`).
- 2026-10-20 M1 pin: `gap.toml` (67 names), `gap pin do-skills@master`, the `"."` entry dropped from `global/skills.json`. Lock commit `ae1571346429`, tree `f3ef5fe1`.
- 2026-10-20 ATL-GAPDEUCE part 1: all 67 skills match Do-Skills at `ae157134`.
- 2026-10-21 The plan itself and the M1 proof task landed: PR #97 (`a79b201b`) and PR #98 (`ef5ab1fd`), both squash-merged to `master`.

## Up next

- M2 Gap's own skill installer, load levels, and project skills. Not started. The plan's next milestone; `dep:gapenv-m1-proof` is now done, so nothing blocks it.

## Remaining

- M3 Recall component + `gap doctor` (ATL-645). Not started. Blocked only by the milestone order; its `dep:gapenv-m1-proof` is done.
- M4 Echo component, after Echo's first tagged release. Not started. Also waits on an Echo tag that does not exist yet (plan decision 9: never pin `master`).
- M5 Forge component + `gap init` / `gap init --adopt`. Not started. Also waits on Forge going public, after a secrets scan and the captain's go (plan decision 3).
- M6 override layer (`~/.config/gap`, `<repo>/.gap`). Not started.
- M7 crew/cloud profile + new-machine bootstrap. Not started.

## Plan vs repo (2026-10-21)

Update, 2026-10-21: M2 through M7 have since landed in the working tree. The milestone table below describes the repo as it was when this file was written.

| Milestone | Plan says done when | Repo today |
|---|---|---|
| M1 Components + lock, Do-Skills only | temp HOME sync then restore byte-identical; second sync changes nothing; BigMac `gap status` zero drift | Done. `gap.toml` schema 2 with one component, `gap.lock` pins `ae1571346429`, `~/.gap/store` in use, verbs `sync`/`status`/`restore`/`pin` exist. Proof recorded on the M1 proof task, merged #96. |
| M2 Own installer, `load` levels, project skills | no `npx`; `do-astro` only in Astro repos | Not started. `cmd/gap/components.go` still shells out to `npx` for skill restore. No `load` or `when` keys in `gap.toml`. |
| M3 Recall + `gap doctor` | `gap doctor` runs `recall doctor`; slug; Recall breadcrumb | Not started. No `doctor` verb, no `recall` component. |
| M4 Echo | adapters in `state.json`; `echo doctor` green | Not started. No `echo` component. Waits on a tagged Echo release. |
| M5 Forge + `gap init` | forge-interview everywhere; adopt dry-run makes no unmarked writes | Not started. No `init` verb. Waits on Forge going public. |
| M6 Overrides | an override survives `gap update` | Not started. No `profiles/`, no `~/.config/gap` handling, no `<repo>/.gap`. |
| M7 Crew/cloud + bootstrap | fresh repo reads the pointer; fresh-HOME doctor green | Not started. |

Also still ahead of the plan, all M2 or later:

- Verbs named in section 5 that do not exist: `gap check`, `gap update`, `gap diff`, `gap doctor`, `gap init`, `gap add`.
- Section 6 step 4, removing `skills/` and the copied `docs/global` and `docs/project`: not done. `skills/` still holds 826 files, `docs/global` 30, `docs/project` 19.
- `global/skills.json` still lists the external skills (plannotator, gitbutler, codegraph, herdr, and others). That matches decision 8, which keeps them in the distro; moving them into `gap.toml` is M2 work.
- Decision 11 (replace the Eight Sleep `APP_CLIENT_SECRET`) is already closed: the task was dropped by the user on 2026-10-19, before this plan.

## Open decisions

None. The plan records all eleven captain decisions from 2026-10-20, and the previous regroup file (`regroup/2026-10-19-regroup-gap.md`) left none open.
