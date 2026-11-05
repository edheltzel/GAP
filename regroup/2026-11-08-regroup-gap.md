# Regroup: GAP

Date: 2026-11-08. Read-only task ATL-GAP-REGROUP-01.

Sources read on registered desktop `6d56310b-4f2e-499c-8954-c63fbc3e13af`, labelled `ip-163-182-62-183.dynamic.fuse.net`; Darwin, home `/Users/ed`. The label alone does not establish that this is BigMac. MacDaddy was disconnected; its files were not read. No remote GitHub files substituted.

- `/Users/ed/Developer/Atlas/GAP/todo.txt` (contains completed and open tasks).
- `/Users/ed/Developer/Atlas/GAP/TODO.md` (points to todo.txt).
- `/Users/ed/Developer/Atlas/GAP/done.txt`: missing.
- `/Users/ed/todo.txt`: GAP route only, not a task; `/Users/ed/done.txt`: no GAP matches.
- `/Users/ed/Developer/Atlas/GAP/docs/plans/2026-10-20-gap-env-provisioner-plan.md`.
- `/Users/ed/Developer/Atlas/GAP/docs/plans/2026-10-22-gap-breadcrumb-ledger-plan.md` (status and outstanding Echo contract).
- Resume point: `/Users/ed/Developer/Atlas/GAP/docs/plans/handoffs/2026-11-05-gap-plans-sync.md`, newest active dated handoff.
- Previous snapshot: `/Users/ed/Developer/Atlas/GAP/regroup/2026-11-05-regroup-gap.md`.
- `/Users/ed/Documents/Atlas/Gap/ATL-GAPENV.md`: external original research, relevant architecture/override sections read.
- `/Users/ed/Developer/Atlas/Echo/todo.txt` and `/Users/ed/Developer/Atlas/DoSkills/todo.txt`: matching open GAP/Plannotator dependencies read.
- `/Users/ed/Developer/Atlas/GAP/regroup/AGENTS.md` and relevant project instructions.

Changed since previous snapshot: preview/reinstall/repair task is now marked completed November 5; site migration and old-site archive marked completed November 6. The newer handoff now names an explicit next dependency. Two documentation/adapter tasks are now tracked rather than only carried in plan prose. jcode remains unresolved.

## Last completed

- 2026-11-06: GAP site migrated into `site/` on Nimbus, Pages cutover verified; old gap-site README updated and repository archived, per local task records. Not independently reverified live in this regroup.

Raw completion records:

```text
x 2026-11-06 2026-11-06 Migrate gap-site into GAP site/ on Nimbus: PR, approval, Pages cutover, verify gap.edheltzel.codes +gap id:site-migration pr:https://github.com/edheltzel/GAP/pull/107 cutover 2026-11-06 19:05Z: gap.edheltzel.codes on Pages gap-website at a28f0973
```

```text
x 2026-11-06 2026-11-06 After cutover verified: update gap-site README, close its issues/PRs, archive edheltzel/gap-site +gap id:site-archive dep:site-migration gap-site README 0fb9ded3, 0 open issues/PRs, archived 2026-11-06
```

## Up next

- Recorded next, handoff: add Echo's `scripts/install.sh --list-installed`, then tag a release. Not started in the task record; no priority or due date.
- Recorded dependent next: pin Echo at that release and verify `gap doctor` records adapters. Blocked on `id:echo-list-installed`.

This order is explicit in the handoff, not inferred from task display order. No third task is explicitly prioritized.

## Remaining

- jcode support decision: not started; unchanged from previous snapshot. Source provides no decision.
- Do-Skills README migration (provisioner section 9 step 4): not started; tracked in DoSkills, not GAP's own task file.
- GAP README cleanup (section 9 step 5): blocked until `global/skills.json` no longer requires npx; do not remove accurate Node/skills.sh prerequisites prematurely.
- Related personal Plannotator task in DoSkills: not started; move Tern launcher out of `/tmp` into dotfiles and export `PLANNOTATOR_BROWSER`, with `open` fallback outside Tern. This is launcher setup, not full GAP plugin support.

Plan reconciliation (recommendation, not recorded next or implementation authorization): the original provisioner plan already describes component self-descriptions, optional extras, merged component settings, machine/project-local overrides, and skill text patches/appends. Therefore the broader extensibility goal is already recorded. However, its November 5 status marks M1–M7 landed and remaining tasks name only documentation and Echo ledger completion. No separate open local task records completion of a general personal arbitrary-tool installation/settings path or Plannotator Plan Review/Code Review/Inbox integration. Compare the proposed contracts with implementation and explicitly track gaps before calling the LazyVim vision complete. The current read did not audit parser/implementation; a proposal is not evidence a feature works.

Raw open GAP task lines:

```text
2026-10-27 Decide jcode support: gap sync could publish to ~/.jcode/prompt-overlay.md (jcode reads ~/AGENTS.md, which GAP keeps as a pointer) +gap
2026-11-05 Pin Echo at the first tag with install.sh --list-installed, then confirm gap doctor records adapters +gap dep:echo-list-installed note:/Users/ed/Developer/Atlas/GAP/docs/plans/2026-10-22-gap-breadcrumb-ledger-plan.md
2026-11-05 Provisioner section 9 step 5: drop the skills.sh and Node path from README Manage skills once global/skills.json no longer needs npx +gap note:/Users/ed/Developer/Atlas/GAP/docs/plans/2026-10-20-gap-env-provisioner-plan.md
```

Raw related task lines:

```text
2026-11-05 Add scripts/install.sh --list-installed (print each installed adapter, exit 0 before warn_if_linked_worktree) and tag a release, so GAP can pin it id:echo-list-installed note:/Users/ed/Developer/Atlas/GAP/docs/plans/2026-10-22-gap-breadcrumb-ledger-plan.md +echo
2026-10-14 Move the Tern Plannotator launcher from /tmp into dotfiles and export PLANNOTATOR_BROWSER, falling back to open outside Tern +doskills dep:absorb-review
2026-11-05 Change the README install section from npx skills add to Gap, keeping a short skills.sh note for outside users (GAP provisioner section 9 step 4) +doskills note:/Users/ed/Developer/Atlas/GAP/docs/plans/2026-10-20-gap-env-provisioner-plan.md
```

Only this new snapshot was written. No task, code, configuration, installation, or version-control changes were made. Transcript lookup from the room was unavailable; current room context and the local sources above were used instead.
