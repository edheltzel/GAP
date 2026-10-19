---
name: do-git-worktree
description: "Isolate parallel work as GitButler lanes in the primary checkout, never as native git worktrees. Use when asked for a worktree, a separate workspace, or parallel branches. Lists, opens, and parks lanes with `but`."
alwaysAllow: ["Bash"]
---

# Parallel work without worktrees

The global policy forbids native git worktrees: no `git worktree add`, and no harness option that isolates work in a worktree. A project contract that names plain git does not change that. Parallel work is a GitButler lane in the primary checkout.

## Check GitButler

GitButler is in use when `command -v but` succeeds **and** `but status` exits 0.

If it is not, ask the user before `but setup` and hold the branch work until they answer. If they decline, work on one plain branch at a time in the primary checkout. Do not create a worktree as a fallback.

## Commands

Run every command from the primary checkout.

- **New lane:** `but branch new <branch>`, then commit to it with `but commit -b <branch> -m "<msg>" <ids>`. Take the file and hunk IDs from `but diff`. Pass IDs whenever more than one lane has uncommitted work.
- **Open an existing branch as a lane:** `but apply <branch>`.
- **Park a lane:** `but unapply <branch>`. The branch ref stays.
- **List lanes:** `but status`.

Tell the user the work lives in the primary checkout, in that lane, not in a second directory.

## Existing worktrees

`git worktree list` is read-only and fine. If an old linked worktree holds a branch, `but apply` of that branch fails until the worktree is gone. Do not remove a worktree with uncommitted changes. Name it and ask the user.

## Related skills

- Dependent PRs as a stack: `do-gh-stack`, if installed.
- Sync, conflicts, and undo: `do-git-safe-pr-workflow`, if installed.
