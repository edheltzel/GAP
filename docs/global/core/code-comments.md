# Code Comments

Quickstart:

```bash
npx skills add edheltzel/Do-Skills --skill=do-code-comments
```

```bash
npx skills update do-code-comments
```

[Source](https://github.com/edheltzel/Do-Skills/tree/master/skills/global/core/do-code-comments)

## What it does

`code-comments` guides you to write high-signal comments for the readers who
come next — your future self, a teammate without the original context, and a
coding agent editing from local evidence only. The defining constraint is that
structure explains the *what*, so a comment only earns its place when it carries
the *why*, the *why not*, what must stay true, or what already went wrong here —
context that cannot live in the code itself.

## When to reach for it

Type `/do-code-comments`, or the agent reaches for it automatically when adding
inline comments, docstrings, or API documentation.

Reach for it when a comment would explain a non-obvious decision, guard an
invariant, or stop someone from "simplifying" code that is deliberately odd. If
a comment would only translate a weak name into better English, this skill sends
you to rename the thing instead — the comment is not the fix.

## Comment the why, not the what

The order of operations is refactor first, comment second: rename, extract, or
reorder until intent is obvious, and add a comment only for the context that
survives that. What earns a comment:

- **Intent** — why this code exists at all.
- **Constraint** — a product, legal, protocol, or platform rule the code must
  respect.
- **Invariant** — a property that must stay true across future edits.
- **Tradeoff** — why the less-obvious implementation beat the simpler-looking
  one.
- **History** — the incident or bug that shaped this, with the id when one
  exists.
- **Warning** — what not to collapse or "clean up," and the failure if you do.

The skill ships copy-ready templates (decision, invariant, compatibility,
incident breadcrumb, temporary work) and a catalog of anti-patterns to avoid:
narration, name translation, vague intent, fake `TEMP: remove later`, ghost
history, and comment drift.

## It's working if

- Every comment names the thing that would break and the consequence, not just a
  preference.
- Comments survive routine refactors instead of going stale.
- A comment stops an agent from making a wrong cleanup — the load-bearing test
  in the review checklist.

## Where it fits

A reach-for-it-anytime standard for authoring and editing: leave the file with
fewer, better comments than you found. It
pairs with the stack-specific craft skills in [`project/`](../../project/), which
lean on clear naming so most comments never need writing.
