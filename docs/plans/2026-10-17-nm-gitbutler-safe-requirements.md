# nm-gitbutler-safe

## Goal

Make the user's local No Mistakes (NM) fork work in GitButler repos without fighting GitButler. An agent publishes one GitButler stack branch through the NM gate from `gitbutler/workspace`, with no checkout, no side clone, and no NM write to a branch GitButler owns.

## In scope

- Detect a GitButler repo automatically, with a per-repo setting that forces the behavior on or off.
- Start a run for a named branch from any checkout: `no-mistakes axi run --branch <name> --intent "..." [--yes]`. NM pushes that branch to the gate itself and drives the run.
- Drive an existing run by id from any checkout (`axi run --run <id>`, already on local branch `feat/axi-run-selector`; `axi respond --run <id>` is already upstream).
- In a GitButler repo, NM never writes `refs/heads/*` and never moves `HEAD` in the user repo. That covers custody return, `axi sync`, `--recover`, `--keep-local`, `--adopt-published`, and the wizard.
- In a GitButler repo, a dirty working tree does not block `axi run --branch`, because the branch content is committed.
- When the pipeline pushed fix commits, NM reports that the branch was updated upstream and names the GitButler step that brings them back (GitButler's own upstream integration for that branch). NM does not run `but` itself.
- GAP policy and skill updates so agents use this path: the No Mistakes rule in `global/AGENTS.md`, the `no-mistakes` skill text NM ships, and the `do-commit` procedure. The side-clone workaround is removed from guidance.

## Out of scope

- Replacing NM's gate worktrees (`~/.no-mistakes/worktrees/<repo>/<run>`) with clones. They never touch the user repo.
- NM calling the `but` CLI, or GitButler pushing to the gate (`but config push-remote no-mistakes` breaks GitButler tracking, PR creation, and fetch).
- jj or other VCS support.
- Changing behavior in non-GitButler repos.

## Success

- From `gitbutler/workspace` in GAP, with uncommitted changes present, `no-mistakes axi run --branch <stack-branch> --intent "..." --yes` reaches a terminal outcome and opens the PR, with intent recorded on the run.
- During and after that run, `refs/heads/<stack-branch>` and `HEAD` in the user repo are unchanged by NM (reflog shows no NM entry), and `but status` reports no conflicts caused by the run.
- When the pipeline pushed fix commits, NM's output says so and names the GitButler integration step; after that step, the local branch contains every fix commit.
- `axi sync`, `--recover`, and the other custody commands refuse to write branch refs in a detected GitButler repo and say why.
- Forcing the setting off restores today's behavior in that repo; in a plain git repo, NM's existing test suite passes unchanged.
- Each change is a small self-contained commit that could be offered upstream as its own PR.

## Constraints

- Work lives in `/Users/ed/Developer/Contributions/NoMistakes` on top of `upstream/main` (https://github.com/kunchenguid/no-mistakes); keep syncing with upstream.
- The installed binary is a local build in `~/.go/bin` (first on PATH) and `~/.no-mistakes/bin`.
- Follow NM's own `AGENTS.md` and test conventions; NM's safety refusals (nested gate context, protected paths, Test unvalidated work) stay intact.

## Decisions

- User: fix GitButler conflicts and keep NM's private gate worktrees. The premise "NM creates native worktrees in the user repo" was checked and is false: no production path registers a worktree in the user repo.
- User: GitButler brings fix commits back; NM never writes the user's branch refs in a GitButler repo.
- User: auto-detect GitButler, with a per-repo override.
- User: new `axi run --branch <name>` is the entry point; no plain `git push no-mistakes` exception in GAP policy.
- User: fork now, shape each change to offer upstream.

## Open questions

- Which GitButler command is the documented way to integrate a pushed remote branch into a local stack branch (`but pull` or another), so NM's message names the right step.

## todo.txt

- Confirm the GitButler command that integrates remote fix commits into a stack branch

Review complete
