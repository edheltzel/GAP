# iOS Code Review

Quickstart:

```bash
npx skills add edheltzel/Do-Skills --skill=do-review-ios
```

```bash
npx skills update do-review-ios
```

[Source](https://github.com/edheltzel/Do-Skills/tree/master/skills/project/swift/do-review-ios)

## What it does

`review-ios` reviews changed Swift files as one orchestrator: it records scope,
runs SwiftLint first, detects which Apple frameworks the diff actually uses, then
loads those review siblings as references. The defining constraint is that
siblings stay references — SwiftUI, SwiftData, Combine, and the rest are not
extra registered skills. Empty scope stops with no issues; it will not invent
findings.

## When to reach for it

You invoke this by typing `/do-review-ios` — the agent won't reach for it on its
own (`disable-model-invocation: true`).

Reach for it on a pre-PR or pre-push pass over `.swift` changes. For an
end-of-session polish of Swift you already wrote, use
[cleanup-swift](./cleanup-swift.md). For a stack-agnostic correctness lens, use
[adversarial-review](../../global/core/adversarial-review.md).

## Detect, then load

Always loads `swift-code-review` and `swiftui-code-review`. Then loads a sibling
only when the diff shows it: SwiftData, Swift Testing, Combine, URLSession,
CloudKit, WidgetKit, App Intents, HealthKit, WatchKit, animation.

`--parallel` fans one subagent per detected area when the harness supports it.
Sequential output is the same. Do not re-flag what SwiftLint already owns.

## It's working if

- No Swift files in scope produces one line and no Issues.
- Findings cite file and line from the opened source, not diff-only context.
- Style that SwiftLint already passed is not repeated.

## Where it fits

A user-invoked stack review, not a cleanup pass. The other orchestrators are
[review-frontend](../frontend/review-frontend.md) and
[backend](../backend/backend.md).
