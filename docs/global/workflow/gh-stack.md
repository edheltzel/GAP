# gh-stack

Quickstart:

```bash
npx skills add edheltzel/Do-Skills --skill=do-gh-stack
```

```bash
npx skills update do-gh-stack
```

[Source](https://github.com/edheltzel/Do-Skills/tree/master/skills/global/workflow/do-gh-stack)

## What it does

`do-gh-stack` manages an ordered stack of branches and pull requests. It includes two backends: GitButler (`but commit` with per-layer IDs, `but branch new --above`, `but pr new -t`) when `but` is on PATH and `but status` succeeds, otherwise the `gh stack` GitHub CLI extension. Do not mix them. Each branch is based on the branch below it, so each pull request shows only that layer's diff.

Its defining constraint is linearity: a branch has one parent and at most one child, with foundational work at the bottom and dependent work above it.

## When to reach for it

Type `/do-gh-stack`, or the agent reaches for it automatically when work needs stacked branches, dependent pull requests, or a checked-out stack.

Reach for it before beginning multi-part work, when editing the layer that owns a change, or when creating, submitting, syncing, rebasing, merging, or checking out a stack. For one local commit, use [Commit](./commit.md); to commit and publish a single branch without a stack, use `/do-commit push`.

## Prerequisites

GitButler path needs `but`. The `gh stack` path needs the `github/gh-stack` extension, Git `rerere`, and `remote.pushDefault` when the repository has more than one remote.

## Non-interactive stack control

`gh stack` changes behavior when stdout is a TTY, so this workflow uses explicit non-interactive commands rather than bare interactive ones. Read stack state with `gh stack view --json`; submit with `gh stack submit --auto`; and use named targets for operations such as checkout and merge.

When a stack changes below the current layer, rebase the branches above it with `gh stack rebase --upstack`. Use `gh stack sync` to reconcile the local stack with GitHub, rebase it, push it, and refresh pull-request state.

## Where it fits

Use this to shape related changes into reviewable PR layers, not as a replacement for an ordinary commit workflow. [Commit](./commit.md) covers a local Conventional Commit, and with `push` it also publishes a single branch.
