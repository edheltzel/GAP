# Regroup: GAP

Date: 2026-11-08
Source: `/Users/ed/Developer/Atlas/GAP/todo.txt` (workspace copy; `done.txt` absent), `+gap` lines in `~/todo.txt` (route only), handoff `docs/plans/handoffs/2026-11-05-gap-plans-sync.md` (`## Next step`; the resume point), compared with the earlier `regroup/2026-11-08-regroup-gap.md`.

## Last completed

- 2026-11-06: Migrate gap-site into GAP `site/` on Nimbus; gap.edheltzel.codes cut over to Pages `gap-website` (PR #107).
  `x 2026-11-06 2026-11-06 Migrate gap-site into GAP site/ on Nimbus: PR, approval, Pages cutover, verify gap.edheltzel.codes +gap id:site-migration pr:https://github.com/edheltzel/GAP/pull/107 cutover 2026-11-06 19:05Z: gap.edheltzel.codes on Pages gap-website at a28f0973`
- 2026-11-06: gap-site README updated, no open issues/PRs, repo archived.
  `x 2026-11-06 2026-11-06 After cutover verified: update gap-site README, close its issues/PRs, archive edheltzel/gap-site +gap id:site-archive dep:site-migration gap-site README 0fb9ded3, 0 open issues/PRs, archived 2026-11-06`

## Up next

- Handoff `## Next step`: add `--list-installed` to Echo's `scripts/install.sh` and tag a release; this is Echo work with no task line in this source.
- Then pin Echo at that tag and confirm `gap doctor` writes adapters to `~/.gap/state.json`: blocked on that Echo release (`dep:echo-list-installed`).
  `2026-11-05 Pin Echo at the first tag with install.sh --list-installed, then confirm gap doctor records adapters +gap dep:echo-list-installed note:/Users/ed/Developer/Atlas/GAP/docs/plans/2026-10-22-gap-breadcrumb-ledger-plan.md`

## Remaining

- Decide jcode support (publish to `~/.jcode/prompt-overlay.md`): not started; the line is unchanged from the earlier 2026-11-08 snapshot, which also listed it as not started, and the source records no decision.
  `2026-10-27 Decide jcode support: gap sync could publish to ~/.jcode/prompt-overlay.md (jcode reads ~/AGENTS.md, which GAP keeps as a pointer) +gap`
- Drop the skills.sh and Node path from README Manage skills once `global/skills.json` no longer needs npx: blocked on that condition.
  `2026-11-05 Provisioner section 9 step 5: drop the skills.sh and Node path from README Manage skills once global/skills.json no longer needs npx +gap note:/Users/ed/Developer/Atlas/GAP/docs/plans/2026-10-20-gap-env-provisioner-plan.md`
