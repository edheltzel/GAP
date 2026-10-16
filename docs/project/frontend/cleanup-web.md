# Web Cleanup Pass

Quickstart:

```bash
npx skills add edheltzel/Do-Skills --skill=do-cleanup-web
```

```bash
npx skills update do-cleanup-web
```

[Source](https://github.com/edheltzel/Do-Skills/tree/master/skills/project/frontend/do-cleanup-web)

## What it does

`cleanup-web` runs an end-of-session cleanup pass over the TypeScript, React, and
web code you changed this session — a holistic review across simplification,
correctness, type safety, React patterns, CSS modernization, and comment
hygiene. The defining constraint is that it proposes, it doesn't apply: every
change comes back with a before/after and the principle behind it, and nothing
lands until you approve.
"No changes needed" is a valid, expected outcome — it won't invent work to look
busy.

## When to reach for it

You invoke this by typing `/do-cleanup-web` — treat it as the thing you run when
you're wrapping up. The agent may also reach for it when you signal you want to
finalize or tidy web work, even without the word "cleanup". Don't use it for
one-off edits, bug fixes, or active feature work.

Reach for it to polish a session's output before you commit. For the
posture-level "keep changes surgical and honest" rules that apply while you're
still writing, use [simplify](../../global/core/simplify.md); this skill is the review at the end.

## Prerequisites

A git working tree with this session's changes still visible — it scopes itself
to files modified during the session via `git status`/`git diff`. If it can't
identify those files with confidence, it stops and asks rather than reviewing the
whole repo.

## Lenses in parallel

The pass dispatches one sub-agent per review lens, each loading its own skill
before reviewing and reporting only through that lens:

- **Simplification** ([simplify](../../global/core/simplify.md)) — dead code, needless
  abstractions, and helpers whose names do not improve the call site.
- **TypeScript** ([write-typescript](../typescript/write-typescript.md)) — sound types, `unknown` over
  `any`, discriminated unions over `as`.
- **Type-driven design** ([parse-dont-validate](../typescript/parse-dont-validate.md)) — push
  checks into types; make invalid states unrepresentable.
- **React effects** ([no-use-effect](./no-use-effect.md)) — derived state over
  effects, event handlers over sync effects, `key`-resets.
- **React performance** — component boundaries, `use client`/`use server`, data
  fetching, memoization.
- **CSS** ([modern-css](./modern-css.md)) — native CSS over JS, logical
  properties, container queries, no legacy hacks.
- **Comment hygiene** ([code-comments](../../global/core/code-comments.md)) — strip "what"
  comments and AI narration; keep "why".
- **Correctness** ([adversarial-review](../../global/core/adversarial-review.md)) —
  dropped guards, edge cases, async timing, stale callers, and other regressions
  introduced by the session.

Findings are aggregated into one batch grouped by file, with inter-lens conflicts
(one lens wants an abstraction, another wants to delete it) flagged for you to
decide.

## Where it fits

A periodic-maintenance skill — the closing bracket on a web coding session, where
the individual review skills it invokes are the per-topic standalones. It reaches
across [simplify](../../global/core/simplify.md),
[adversarial-review](../../global/core/adversarial-review.md),
[write-typescript](../typescript/write-typescript.md), [parse-dont-validate](../typescript/parse-dont-validate.md),
[no-use-effect](./no-use-effect.md), [modern-css](./modern-css.md), and
[code-comments](../../global/core/code-comments.md) so you don't have to run each by hand.
