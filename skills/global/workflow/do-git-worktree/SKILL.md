---
name: do-git-worktree
description: "Create, remove, and list git worktrees in a standardized location"
alwaysAllow: ["Bash"]
---

# Git Worktree Management

Manage git worktrees stored in `~/.git-worktrees/<repo>/<branch>`, keeping them out of the user's project directories.

## GitButler

GitButler is in use when `command -v but` succeeds **and** `but status` exits 0. Do not run `but setup`. If either check fails, use the git worktree commands below.

When GitButler is in use, do **not** `git worktree add`. Feature work is an applied lane in the primary checkout:

```bash
cd <primary-repo>
but apply <branch>          # existing branch
# new work: but commit -b <branch> -m "..."
```

- **Create:** apply or commit onto a lane. Tell the user the work lives in the primary checkout, not `~/.git-worktrees`.
- **Remove:** `but unapply <branch>` (keeps the branch ref).
- **List:** `but status`.
- If the user insists on a second directory, warn that a linked worktree makes that ref the worktree target, so `but apply` of the same branch will fail.


## Determining Context

Before any operation, determine the repository name:

```bash
repo=$(basename "$(git rev-parse --show-toplevel)")
```

The worktree base directory is always `~/.git-worktrees/$repo/`.

## Commands

### Create (default)

If the user provides just a branch name, treat it as a create operation.

```bash
repo=$(basename "$(git rev-parse --show-toplevel)")
branch="<branch>"
dir="$HOME/.git-worktrees/$repo/$branch"
mkdir -p "$HOME/.git-worktrees/$repo"
git worktree add "$dir" "$branch"
```

If the branch doesn't exist yet, use `-b` to create it from HEAD:

```bash
git worktree add -b "$branch" "$dir" HEAD
```

After creation, **always prominently display the worktree path** so the user can open a new session pointing to it:

> Worktree created at: `~/.git-worktrees/<repo>/<branch>`

### Remove

```bash
repo=$(basename "$(git rev-parse --show-toplevel)")
git worktree remove "$HOME/.git-worktrees/$repo/<branch>"
```

If the worktree has uncommitted changes, warn the user before using `--force`.

### List

```bash
git worktree list
```

### Prune

Clean up stale worktree references (e.g., after manually deleting a worktree directory):

```bash
git worktree prune
```

## Guidelines

- Always default to **create** if the user's intent is ambiguous
- Never use `--force` without explicit user confirmation
- After creating a worktree, remind the user they can open a new Craft Agent session with that path as the working directory
- When listing, note which worktree is the main working tree vs. linked worktrees
