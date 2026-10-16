# Frontend Code Review

Quickstart:

```bash
npx skills add edheltzel/Do-Skills --skill=do-review-frontend
```

```bash
npx skills update do-review-frontend
```

[Source](https://github.com/edheltzel/Do-Skills/tree/master/skills/project/frontend/do-review-frontend)

## What it does

`review-frontend` reviews changed React/TypeScript frontend files as one
orchestrator: it records scope, detects the router and libraries in the diff,
then loads matching review siblings as references. The defining constraint is
the Remix fork — `@remix-run/*` loads the Remix v2 umbrella and skips React
Router review; otherwise it loads React Router and shadcn. Siblings stay
references, not extra registered skills.

## When to reach for it

You invoke this by typing `/do-review-frontend` — the agent won't reach for it
on its own (`disable-model-invocation: true`).

Reach for it on a pre-PR pass over frontend changes. For an end-of-session
polish of web code you already wrote, use [cleanup-web](./cleanup-web.md). For
authoring style rather than review, use
[write-typescript](../typescript/write-typescript.md) or
[no-use-effect](./no-use-effect.md).

## Detect, then load

Remix branch: Remix v2 umbrella (routing, data-flow, forms, error boundaries,
SSR, meta/sessions) plus shadcn. Default branch: React Router and shadcn. Both
branches add React Flow, Zustand, Tailwind v4, or Vitest only when detected.

`--parallel` fans one subagent per area when the harness supports it. Sequential
output is the same.

## It's working if

- Remix diffs do not also run React Router review.
- Findings cite opened source, not diff-only context.
- Conditional siblings load only when their detection row matches.

## Where it fits

A user-invoked stack review, not a cleanup pass. The other orchestrators are
[review-ios](../swift/review-ios.md) and
[backend](../backend/backend.md).
