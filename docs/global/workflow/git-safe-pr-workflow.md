# Git Safe PR Workflow

Quickstart:

```bash
npx skills add edheltzel/Do-Skills --skill=do-git-safe-pr-workflow
```

```bash
npx skills update do-git-safe-pr-workflow
```

[Source](https://github.com/edheltzel/Do-Skills/tree/master/skills/global/workflow/do-git-safe-pr-workflow)

## What it does

`git-safe-pr-workflow` guides you through a GitHub-first pull request flow —
branching, syncing, resolving conflicts, undoing mistakes, and landing changes —
that keeps `main` clean without demanding fluent git. The defining constraint is
that it biases toward safety over elegance: it prefers reversible operations,
merges `origin/main` into your feature branch instead of rebasing it, and reaches
for `git revert` over rewriting history whenever commits may already be pushed.
If GitButler is in use, those writes become `but pull`, `but resolve`, and
`but push`; the git recipes stay as the fallback. When No Mistakes is initialized,
that gate owns push, opening the PR, and catching up a live gated PR.

## When to reach for it

Type `/do-git-safe-pr-workflow`, or the agent reaches for it automatically on
triggers like "update my branch", "resolve git conflicts", "recover from a bad
rebase", "undo a pushed commit", or "keep main clean".

Reach for it when a git operation could lose work or rewrite shared history and
you want a walkthrough that inspects first and explains as it goes — especially if
you're not a confident git user. It teaches the next decision, not the whole tool.

## The safe posture

- **Inspect before acting.** Check current branch, clean vs. dirty tree, upstream
  tracking, and whether the branch is shared before any history-changing command.
  If shared status is unknown, assume shared and take the safer path.
- **Merge, don't rebase, pushed branches.** Sync with `fetch` then merge
  `origin/main`; rebase is offered only for clearly private, unpublished commits.
- **Revert, don't reset, pushed work.** Undo with `git revert`; destructive
  local undo is reserved for unpublished changes.
- **Squash and merge.** Land PRs with a squash so `main` stays readable, treating
  the PR title as the final commit message.
- **Refuse or warn hard** on committing to `main`, force-pushing protected
  branches, or `reset --hard` / `clean -fd` without a clear destructive request.
- **No Mistakes owns publication.** If the repo has a `no-mistakes` remote, do not `git push origin`, `but push`, `but pr new`, or `gh pr create`. Drive `no-mistakes axi`. Do not open a PR early; do not merge `origin/main` onto a live gated PR; do not treat the pipeline's guarded force-push as a user force-push. While a run is active, do not hand-rebase or edit findings yourself.

Deeper playbooks live in the skill's `references/` — `conflict-resolution.md`,
`recovery.md`, and `repo-settings.md`.

## It's working if

- Feature branches sync via a merge of `origin/main`, not a rebase of pushed
  commits.
- Mistakes on shared history get reverted rather than reset away.
- Conflict resolutions are verified against the final code, not just cleared of
  markers, and any behavior change is explained back to you.
- In a gated repo, the pipeline opens the PR after local validation; `checks-passed` means it is ready for review and merge, not that the PR only then appears. Do not open one early with `gh pr create`.

## Where it fits

A reach-for-it-anytime standalone for the branch-to-merge lifecycle. It follows
[git-worktree](./git-worktree.md), which sets up where a branch lives, and
feeds [git-pr-review-triage](./git-pr-review-triage.md), which sorts the
review comments once the PR is open. One-shot commit-and-publish is
[commit](./commit.md) with `push`; both hand publication to No Mistakes when
that gate is initialized.
