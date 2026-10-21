# Git Worktree Management

Quickstart:

See [Manage skills](../../../README.md#manage-skills) for installation and updates.

[Source](../../../skills/global/workflow/do-git-worktree/SKILL.md)

## What it does

The global policy forbids native git worktrees. When you ask for a worktree, a
separate workspace, or parallel branches, this skill gives you a GitButler lane in
the primary checkout instead. It never runs `git worktree add`.

## When to reach for it

Type `/do-git-worktree`, or the agent reaches for it when you ask for a worktree or
want to work a second branch in parallel: a review, a hotfix, a long-running
feature.

## Operations

- **New lane**: `but branch new <branch>`, then `but commit -b <branch>` with the
  file and hunk IDs from `but diff`.
- **Open an existing branch**: `but apply <branch>`.
- **Park a lane**: `but unapply <branch>`. The branch ref stays.
- **List lanes**: `but status`.

If the repo is not in GitButler mode, the agent asks before `but setup`. If you
decline, it works one plain branch at a time. It never falls back to a worktree.
`git worktree list` stays allowed, and an old worktree with uncommitted changes is
named to you, never removed.

## Where it fits

A reach-for-it-anytime standalone for parallel branches.
It pairs with the other Git workflow skills:
[git-safe-pr-workflow](./git-safe-pr-workflow.md) governs how a branch syncs
and merges once you're working in it, and
[git-pr-review-triage](./git-pr-review-triage.md) handles the review comments
that come back on its PR.
