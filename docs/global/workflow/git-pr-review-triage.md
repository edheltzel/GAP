# PR Review Triage

Quickstart:

```bash
npx skills add edheltzel/Do-Skills --skill=do-git-pr-review-triage
```

```bash
npx skills update do-git-pr-review-triage
```

[Source](https://github.com/edheltzel/Do-Skills/tree/master/skills/global/workflow/do-git-pr-review-triage)

## What it does

`do-git-pr-review-triage` pulls the review comments on a PR, classifies each one, and drafts a reply. Accepted items go into a feedback file that `do-work` applies. It does not edit the code itself.

Replies post and threads resolve only after the fixes are pushed and you say go. A caller cannot authorize that post.

## When to reach for it

Type `/do-git-pr-review-triage`, or the agent reaches for it when you mention PR comments, review feedback, or triage, and when you ask to post the drafted replies.

Reach for it when a PR comes back with a wall of comments and you want a split between what to change and what to answer. Not for writing a PR description ([tech-writing](./tech-writing.md)), reviewing a diff before comments exist ([adversarial-review](../core/adversarial-review.md)), or committing the fixes ([commit](./commit.md)).

## The triage

- **Pull.** Via `gh`, gather review-level and inline comments: author, `file:line`, thread id, resolution, and the commit SHA the comment was made against.
- **Classify.** Substantive (`bug`, `design`, `domain`) or noise (`bikeshed`, `stale`, `misread`, `vague`, `ai-slop`, `scope-creep`, `unfounded`). The test: would the change leave the code meaningfully better, or just different?
- **Recommend.** Address, Push back, Defer, Clarify, or Ignore, plus a direct draft reply. No apologetic preamble.

Accepted items are the Address set, unless you move items first. The labels are a draft; it defers when you push back.

## The feedback file

Before the next rung, it writes `docs/plans/YYYY-MM-DD-pr-<number>-feedback.md` (or the location project AGENTS.md names). Each accepted item has `file:line`, the change, and the drafted reply. Held comments keep their drafts too. The file ends with `## todo.txt`.

The next rung is `do-work` with that file. The other moves are post the replies (only after the fixes are pushed and you say go), or stop.

## It's working if

- You have a feedback file whose accepted items name `file:line`, and no code was edited by this skill.
- Replies are still unposted until you say go after the push.
- A thread with a failed post is still open, and is not reported resolved.

## Where it fits

This is the respond stage. It sits after a PR from [commit](./commit.md). `do-work` applies the accepted list, ship publishes the fixes, and this skill posts the drafts only when you say go.
