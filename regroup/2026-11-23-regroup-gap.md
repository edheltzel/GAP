# Regroup: GAP

Date: 2026-11-23
Sources: GAP `todo.txt` and `done.txt` via `tuxedo lsa --json` (109 done lines, 10 open); Echo `todo.txt` for the two cross-repo dependencies; newest handoff by name and mtime. Compared with `regroup/2026-11-16-regroup-gap.md`.
Resume point: `docs/plans/handoffs/2026-11-05-gap-plans-sync.md` (newest by name and mtime, 2026-11-11 11:31). It is stale: it predates PRs #138 and #139, the gap TUI, and the jcode decision it still lists as open.

## What changed since 2026-11-16

- The one task called "up next" last time is done: v0.5.1 was tagged on 2026-11-16.
- New and finished since then: PR #138 (gap add installs skills from any repo) merged 2026-11-23, PR #139 (gap add reads nothing outside the checkout) merged 2026-11-23, and the gap TUI built in the checkout with review round 3 passing. It is uncommitted.
- Seven open lines are new since the last snapshot. The two blocked lines are unchanged.

## Last completed

- **Merge PR 139 and remove the `.worktrees/gap-discovery-containment` worktree: done (2026-11-23).** `pr:https://github.com/edheltzel/GAP/pull/139`. The same day also closed the review of #139, the os.Root containment fix, the discovery-before-read fix, the gap TUI plan, and the gap TUI build (U1-U7, uncommitted, review round 3 PASS).

## Up next

No line carries a priority, a "next" label, or an in-progress flag. The cue is the dependency chain: `id:gap-tui` is done, so the three tasks that name `dep:gap-tui` are unblocked, and one of them needs no decision.

- **Commit the gap TUI and open its PR: not started.** `dep:gap-tui` is done. The task says to leave `gap.lock`, `.gitignore`, and the PR 138 plan files to their owners.
- **Reinstall gap from a master checkout: not started.** `go install ./cmd/gap`, so the installed binary picks up `gap add` from PR #138.
- **Add gap tui to the site docs: not started.** README, CONCEPTS, the landing-page command table, and the install line in `site/src/site.js`. `dep:gap-tui` is done.

## Remaining

- **Screenshot the gap TUI and put the shots in the README and the landing page: not started.** Items, Status, Projects, a running update, and the delete confirmation. `dep:gap-tui` is done; it follows the commit, since the shots land in the same docs.
- **Refresh jcode skill copies that match a recorded GAP store: not started.** Left out of #132. `dep:gap-drift-baseline` is done (`x` on 2026-11-20, PR #132), so this wait has cleared.
- **Update Do-Skills `do-plan` and `do-regroup` to emit the paired `.human.md`: not started.** Then re-pin with `gap pin do-skills@master`. The line says the lock is `0bab2b0` and master is `2926d08`.
- **Follow up PR 138 review on master: blocked.** Waiting on Ed's go for a follow-up branch. Three items: list `-l` (now `--list`, was `--local`) under Breaking changes, fail `gap add` when a submodule update fails instead of storing a half-filled pin (`components.go:991`), and `go mod tidy` so `go.yaml.in/yaml/v3` is direct.
- **Decide whether the gap TUI offers update-all and per-project restore: blocked.** Waiting on Ed.
- **Pin Echo at the first tag with `install.sh --list-installed`: blocked.** `dep:echo-list-installed` is still open in Echo's own list: PR #270 merged and was promoted in #276, but the release tag is not cut. That line waits on Ed, because the Empryo pre-commit tsc gate fails on 85 pre-existing dev type errors. v0.13.3 lacks the flag.
- **Move GAP's Echo pin to the first release tag that ships `gap-utility.toml`: blocked.** Waiting on an Echo release after v0.13.3. No task with `id:gap-utilities-migrate` exists in the GAP list, the global list, or the Echo list.

## Raw open lines

```
2026-11-05 Pin Echo at the first tag with install.sh --list-installed, then confirm gap doctor records adapters +gap dep:echo-list-installed note:/Users/ed/Developer/Atlas/GAP/docs/plans/2026-10-22-gap-breadcrumb-ledger-plan.md
2026-11-12 Move GAP's echo pin to the first Echo release tag that ships gap-utility.toml and drop its spec install/doctor/names +gap dep:gap-utilities-migrate @waiting Echo release after v0.13.3
2026-11-20 Refresh jcode skill copies that match a recorded GAP store instead of only warning (left out of #132) +gap dep:gap-drift-baseline
2026-11-20 Update Do-Skills do-plan and do-regroup to emit the paired .human.md per global/docs/agents/plans.md, then re-pin do-skills with gap pin do-skills@master (lock is 0bab2b0, master 2926d08) +gap
2026-11-23 Add gap tui to the site docs: the site already serves README and CONCEPTS live at gap.edheltzel.codes/docs, so update README and CONCEPTS, the command table on the landing page, and the install line in site/src/site.js dep:gap-tui +gap +registry
2026-11-23 Commit the gap TUI (do-commit) and open its PR; leave gap.lock, .gitignore and the PR 138 plan files to their owners +gap dep:gap-tui note:/Users/ed/Developer/Atlas/GAP/docs/plans/2026-11-23-gap-tui-plan.md
2026-11-23 Decide whether the gap TUI offers update all and per-project restore +gap @waiting Ed note:/Users/ed/Developer/Atlas/GAP/docs/plans/2026-11-23-gap-tui-plan.md
2026-11-23 Follow up PR 138 review on master: list -l (now --list, was --local) under Breaking changes, fail gap add when submodule update fails instead of storing a half-filled pin (components.go:991), go mod tidy so go.yaml.in/yaml/v3 is direct +gap @waiting Ed's go for a follow-up branch pr:https://github.com/edheltzel/GAP/pull/138
2026-11-23 Reinstall gap from a master checkout (go install ./cmd/gap) so the installed binary gets gap add from PR 138 +gap pr:https://github.com/edheltzel/GAP/pull/138
2026-11-23 Screenshot the gap TUI (Items, Status, Projects, a running update, the delete confirmation) and put the shots in the README and on the site landing page +gap dep:gap-tui note:/Users/ed/Developer/Atlas/GAP/docs/plans/2026-11-23-gap-tui-plan.md
```

## Open decisions

No open decision was recorded in the previous check-in. Two waits are on Ed and are named above: the TUI update-all and per-project restore question, and the go-ahead for the PR #138 follow-up branch.
