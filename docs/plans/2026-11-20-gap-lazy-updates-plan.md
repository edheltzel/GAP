# GAP updates the way lazy.nvim does

Date: 2026-11-20. Planning base: master `962ae92f`. Status: built on branch `feat/lazy-update` by the DoSkills tab, with the GAP tab's handover; steps 1-6 done, jcode refresh left as a warning. Ed asked to build it 2026-11-20.

## Goal

Get the latest component skills with one command, never hand-edit a pin, and keep every machine on the same versions. Copy lazy.nvim's model, which LazyVim ships by default.

## How lazy.nvim does it

Source: lazy.nvim `doc/lazy.nvim.txt`, `lua/lazy/manage/init.lua`, `lua/lazy/manage/task/git.lua`, `lua/lazy/core/loader.lua`, and the LazyVim starter `lua/config/lazy.lua` (all read 2026-11-20).

| Concept | lazy.nvim | Notes |
| --- | --- | --- |
| Spec separate from lock | The spec takes `branch`, `tag`, `commit`, or `version` (a semver range); `lazy-lock.json` stores the commit actually installed | LazyVim sets `defaults.version = false`, meaning the latest git commit, because many plugin releases are stale |
| Install follows the lock | On startup, missing plugins install with `install({ lockfile = true })` | Install never moves a plugin |
| Update writes the lock | `:Lazy update` runs fetch, then status, then checkout at the spec tip, then rewrites `lazy-lock.json` | The lock is meant to be committed to the config repo |
| Restore | `:Lazy restore` puts every plugin back on the lock | Run on another machine after pulling the config |
| Check without writing | `:Lazy check` fetches and shows pending commits; `checker.enabled` runs it every hour (LazyVim: on, `notify = false`) | Check never changes anything |
| Hold a plugin | `pin = true` leaves the plugin out of update; `checker.check_pinned` controls whether check still looks at it | |
| Local edits block update | Update refuses a plugin with local changes: "You have local changes ... Please remove them to update." | lazy.nvim never overwrites edits silently |
| Log | `:Lazy log` shows the commits each update brought in | |
| Local dev | `dev = true` or `dev.patterns` use a local checkout instead of the remote | |

## Mapping to GAP

| lazy.nvim | GAP today | GAP after this plan |
| --- | --- | --- |
| spec | `gap.toml` `ref` | Same; add `pin = true` |
| `lazy-lock.json` | `gap.lock` (distro), `~/.config/gap/gap.lock` (utilities) | Same files |
| install | `gap sync` installs from the lock | Unchanged; sync never moves a pin |
| `:Lazy update` | `gap update` handles utilities only; distro components need `gap pin <id>@<ref>` by hand | `gap update [id]` also handles distro components |
| `:Lazy restore` | `gap restore <id>` | Unchanged |
| `:Lazy check` + checker | none | `gap check [id]`; `gap status` shows `N new` |
| `pin = true` | none | `pin = true` in `gap.toml` |
| local changes block update | drift prompt (keep, take source, skip) | Untouched copies upgrade without asking (#83); edited copies still prompt |
| `:Lazy log` | none | `gap update` prints the commits it brought in |

### Distro lock: option (a), agreed with the GAP tab

`gap update` writes the distro `gap.lock` in the checkout, the same way `gap pin` does. It prints the commits and tells Ed to commit `gap.lock` and open a PR. The merge gate keeps refusing sync until that PR lands. This is lazy.nvim's committed lockfile with a review step on top.

Rejected: (b) a machine-only override of the distro lock creates a second source of truth that other machines never see. (c) Opening the PR automatically is a push action, which needs Ed's go every time.

## Steps

1. `pin = true` in `[components.<id>]`. `gap update` and `gap check` skip a pinned component unless it is named explicitly. Tests: parse, and skip during update.
2. `gap update [id]` for distro components. For every component without `pin = true` (or only `id`), resolve the tip of its `gap.toml` ref with `resolveRef`, and when the commit moved, write the lock with `pinCommit`/`writeLock`. Utilities keep their current path. Print `id old..new` plus `git log --oneline` for each moved component. End with the commit and PR instruction for `gap.lock`. Tests: moves the lock, honors `pin = true`, no change when already at the tip.
3. `gap check [id]`. Resolve each tip without writing anything and list the pending commits. `gap status` adds `N new` to a component line when the tip is ahead. Reuse the fetch cache; any network failure is reported, never fatal. Tests: no file writes, and the count is correct.
4. #83: during sync and update, upgrade copies that match their recorded baseline (`skillCopies`, the skills.sh store source, a recorded jcode copy) without asking. Only real edits prompt. This is lazy.nvim's local-changes rule, but softer: edits still get keep or take source.
5. Docs. Update the README Components section, `gap update` / `gap check` / `pin = true`, `cmd/gap/AGENTS.md`, and the GAP site page. Replace "move the pin with `gap pin do-skills@master`" with `gap update do-skills`.
6. Prove it in a throwaway HOME: `gap update do-skills` moves the lock from `0bab2b0` to the DoSkills tip, `gap sync` after merge installs it, the 9 formerly drifted skills upgrade silently, and a hand-edited skill still prompts.

## The no-pin rule

Ed said "don't pin doSkills with master or a version", then asked for lazy.nvim's model, which keeps a lock file. Reading used here: nobody pins do-skills by hand. `gap.toml` keeps `ref = "master"`, `gap update` records the tip commit in `gap.lock` the way lazy.nvim writes `lazy-lock.json`, and do-skills does not set `pin = true`.

## Built

- `check.go`: `gap check`, `updateDistro`, `~/.gap/check.json`, the blobless fetch cache `~/.gap/cache/<id>.git`.
- `pin = true` parsed in `parseComponent`; update and check skip it unless named.
- `gap status` prints `new=N` (piped) or `+N new` (terminal) from the last check while the pin is unchanged.
- `upgradeUntouched` in sync replaces copies equal to an earlier verified store of the same component, before drift review. It needs no new `state.json` field, so `do-skill-drift` keeps its jq paths.
- Test: `TestLazyUpdateCycle` runs sync, check, status, update, sync after merge, and `pin = true` end to end.
- Real machine: `gap check do-skills` listed #66 and #67 after `0bab2b0`, and `gap status` printed `do-skills ... new=2`.

## Todo

```
2026-11-20 Build lazy-style updates: pin = true, gap update for distro components, gap check and N new in status, per docs/plans/2026-11-20-gap-lazy-updates-plan.md +gap id:gap-lazy-update
```
