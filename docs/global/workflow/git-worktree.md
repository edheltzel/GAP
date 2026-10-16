# Git Worktree Management

Quickstart:

```bash
npx skills add edheltzel/Do-Skills --skill=do-git-worktree
```

```bash
npx skills update do-git-worktree
```

[Source](https://github.com/edheltzel/Do-Skills/tree/master/skills/global/workflow/do-git-worktree)

## What it does

`git-worktree` creates, removes, lists, and prunes git worktrees so you can check
out several branches at once without cloning the repo again or stashing your work.
The defining constraint is location: every worktree lands in a standardized path,
`~/.git-worktrees/<repo>/<branch>`, never inside a project directory — so your
working trees stay tidy and a worktree never gets mistaken for repo content.
If GitButler is in use (`but` on PATH and `but status` succeeds), it does not
`git worktree add`. It applies a lane in the primary checkout instead.

## When to reach for it

Type `/do-git-worktree`, or the agent reaches for it automatically when you ask to
create, remove, or list a worktree. Given a bare branch name, it defaults to
**create**.

Reach for it when you want to work a second branch in parallel — a review, a
hotfix, a long-running feature — without disturbing your current checkout. After
creating one, it prints the worktree path so you can open a fresh session pointed
at it.

## Operations

- **Create** (the default): `git worktree add` into `~/.git-worktrees/<repo>/<branch>`,
  using `-b` to branch from `HEAD` when the branch doesn't exist yet.
- **Remove**: deletes the linked worktree; if it has uncommitted changes, you're
  warned before anything uses `--force`.
- **List**: shows every worktree and flags which is the main working tree.
- **Prune**: clears stale references left by a manually deleted directory.

Safety is built in: `--force` never runs without your explicit confirmation, and
ambiguous intent resolves to the non-destructive create.

## Where it fits

A reach-for-it-anytime standalone for managing where your branches live on disk.
It pairs with the other Git workflow skills:
[git-safe-pr-workflow](./git-safe-pr-workflow.md) governs how a branch syncs
and merges once you're working in it, and
[git-pr-review-triage](./git-pr-review-triage.md) handles the review comments
that come back on its PR.
