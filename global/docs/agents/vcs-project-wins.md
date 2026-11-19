# Git workflow, project wins

## Rules

- Default: plain git. Branches, commits, pushes, and undo go through `git`.
- Parallel or isolated work goes in a git worktree under `.worktrees/` in the main checkout, one branch per worktree. From the main checkout: `git worktree add -b <branch> .worktrees/<name>`, where `<name>` is the branch with `/` replaced by `-`. Keep the primary checkout on the default branch.
- Keep `.worktrees/` out of git without touching tracked files: before the first worktree, add the line `.worktrees/` to `$(git rev-parse --git-common-dir)/info/exclude` if it is not there. Do not edit the project's `.gitignore` for it.
- Repo-level files that are not part of the branch's change, such as `todo.txt` and `done.txt`, live in the main checkout (the clone itself), never in a linked worktree. `git worktree list` shows both; a linked worktree's `.git` is a file that points into the main checkout's git dir.
- Remove a worktree with `git worktree remove` once its branch is merged or abandoned, then delete the branch. Never delete a worktree folder by hand.
- Other agents may share a checkout. Stage files by name. Never run a whole-tree command (`git stash`, `reset --hard`, `clean`, `checkout -- .`, `add -A`) in a shared checkout.
- The nearest project `AGENTS.md` wins on version-control tooling, including a repo that uses GitButler or forbids worktrees.
- No force-push without asking. Do not push directly to a protected branch. Prefer a PR into the integration branch.

## Done

The work landed through git (or the tooling the project named), on its own branch, and any worktree made for it was removed after merge.
