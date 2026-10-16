# Cleanup Swift

Quickstart:

```bash
npx skills add edheltzel/Do-Skills --skill=do-cleanup-swift
```

```bash
npx skills update do-cleanup-swift
```

[Source](https://github.com/edheltzel/Do-Skills/tree/master/skills/project/swift/do-cleanup-swift)

## What it does

`cleanup-swift` runs an end-of-session cleanup pass over the Swift code you
touched — a holistic review across simplification, correctness, type-driven
design, macOS conventions, and comment hygiene, aimed at leaving the session's
work in a pristine state. It proposes, it does not apply: every change comes
with a before/after, the principle behind it, and any trade-off, and nothing
lands until you approve. "No changes" is a valid outcome — if the code is already
clean, it says so and stops rather than inventing work.

## When to reach for it

Type `/do-cleanup-swift`, or the agent reaches for it automatically when you signal
you're wrapping up Swift work — "clean up", "polish", "tidy", "finalize". It is
explicitly *not* for one-off edits, bug fixes, or active feature work.

Reach for it at the end of a Swift session, not during it. For the same
end-of-session pass on TypeScript, React, and web code, use
[cleanup-web](../frontend/cleanup-web.md) — this is its Swift analogue, same shape,
different review lenses.

## The parallel-lens review

Scope is strictly the files modified this session (found via `git status` /
`git diff`); if it can't identify them with confidence, it stops and asks.
It then dispatches one sub-agent per review lens, all in parallel, each loading
its own skill first and reviewing only through that lens:

- **Simplification** ([simplify](../../global/core/simplify.md)) — dead code, needless
  abstractions, and helpers whose names do not improve the call site.
- **Type-driven design** ([parse-dont-validate](../typescript/parse-dont-validate.md)) — push
  checks into types; make invalid states unrepresentable.
- **Design patterns** ([design-patterns-gof](../../global/core/design-patterns-gof.md)) —
  patterns only where they earn their weight.
- **Platform conventions** ([macos-swift-desktop](./macos-swift-desktop.md)) —
  naming, ARC, AppKit/SwiftUI boundaries, threading, main-actor isolation.
- **Comment hygiene** ([code-comments](../../global/core/code-comments.md)) — strip "what"
  comments and AI narration; keep "why" only.
- **Correctness** ([adversarial-review](../../global/core/adversarial-review.md)) —
  dropped guards, edge cases, concurrency hazards, swallowed errors, stale
  callers, and other regressions introduced by the session.

Findings aggregate into a single batch grouped by file, with conflicts between
lenses flagged for you to decide.

## Where it fits

A periodic-maintenance skill you run once, at the end of a session — the Swift
sibling of [cleanup-web](../frontend/cleanup-web.md). It doesn't author code; it composes
several authoring and review skills — including
[simplify](../../global/core/simplify.md),
[adversarial-review](../../global/core/adversarial-review.md), and
[macos-swift-desktop](./macos-swift-desktop.md) — into review lenses and turns
them on the diff you just produced.
