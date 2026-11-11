# Handoff: GAP plans synced and archived

Date: 2026-11-05

## State

- Active plans in `docs/plans/`:
  - `2026-10-20-gap-env-provisioner-plan.md`: M1-M7 landed (#96, #99). Status line under the header. Open: section 9 steps 4 and 5, listed in [its Todo section](../2026-10-20-gap-env-provisioner-plan.md#todo).
  - `2026-10-22-gap-breadcrumb-ledger-plan.md` and its requirements: U1-U3 landed in #100. Open: the [Echo release and pin](../2026-10-22-gap-breadcrumb-ledger-plan.md#todo).
- Archived to `docs/plans/archive/` because every task is done or dropped: the 2026-10-14 product-name (#75) and merge-gate (#76) plans, gap-skills plan and requirements, secrets scan, and the nm-gitbutler-safe plan and requirements. Repo paths inside them now point at `archive/`.
- `todo.txt`: the `note:` paths of 11 completed lines now point at `archive/`. Two new `+gap` tasks are open (see below).
- Machine-local, outside this repo: an Echo task (`id:echo-list-installed`) in `/Users/ed/Developer/Atlas/Echo/todo.txt`, a Do-Skills README task in `/Users/ed/Developer/Atlas/DoSkills/todo.txt`, and an Echo `@route` in `/Users/ed/todo.txt` (it had none). These are uncommitted in their repos.

## Decisions

- Mine: a plan is archived only when every task in it is `x` or recorded as dropped. The provisioner plan stays active for section 9 steps 4-5.
- Mine: the Echo flag and the Do-Skills README change go in those repos' own lists, per the routing rule. The GAP pin task depends on `id:echo-list-installed`.
- Dated regroup snapshots keep their old plan links. `regroup/AGENTS.md` says not to rewrite them.

## Open questions

- jcode support (todo.txt, 2026-10-27): publish to `~/.jcode/prompt-overlay.md` or not. Your call.

## Next step

Add `--list-installed` to Echo's `scripts/install.sh` and tag a release. Then run `gap pin echo@<tag>` and confirm `gap doctor` writes adapters to `~/.gap/state.json`. Contract: [breadcrumb ledger plan U2](../2026-10-22-gap-breadcrumb-ledger-plan.md#u2-echo-adapter-ledger).

## Links

- `docs/plans/2026-10-20-gap-env-provisioner-plan.md`: status line, section 9, and its `## Todo`
- `docs/plans/2026-10-22-gap-breadcrumb-ledger-plan.md`: status line, U2, and its `## Todo`
- `regroup/2026-11-05-regroup-gap.md`: the snapshot this sync closes out
- `todo.txt`: the project list

## Todo

Open tasks in `todo.txt`, copied verbatim:

```text
2026-10-27 Decide jcode support: gap sync could publish to ~/.jcode/prompt-overlay.md (jcode reads ~/AGENTS.md, which GAP keeps as a pointer) +gap
2026-11-05 Pin Echo at the first tag with install.sh --list-installed, then confirm gap doctor records adapters +gap dep:echo-list-installed note:/Users/ed/Developer/Atlas/GAP/docs/plans/2026-10-22-gap-breadcrumb-ledger-plan.md
2026-11-05 Provisioner section 9 step 5: drop the skills.sh and Node path from README Manage skills once global/skills.json no longer needs npx +gap note:/Users/ed/Developer/Atlas/GAP/docs/plans/2026-10-20-gap-env-provisioner-plan.md
```
