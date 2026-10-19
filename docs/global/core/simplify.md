# Simplify

Quickstart:

This skill lives in this repo (`edheltzel/agent-config`, being renamed GAP) at `skills/global/core/do-simplify`. `global/skills.json` lists it with `"source": "."`. Install and refresh it with:

```bash
gap sync
```

One skill by exact name:

```bash
npx skills add edheltzel/agent-config -s do-simplify
```

[Source](https://github.com/edheltzel/agent-config/tree/master/skills/global/core/do-simplify)

## What it does

`simplify` refines recently modified code so its intent is easier to see without
changing what it does. A good simplification removes concepts, not merely lines:
behavior, public APIs, side effects, timing, and error handling stay intact.
It will not thin a safety check to get there.

## When to reach for it

Type `/do-simplify`, or the agent reaches for it after code has been written or
modified and a behavior-preserving cleanup would help.

Reach for it when nesting, vague names, duplicate logic, dead branches, or
speculative abstractions make a change harder to understand. Over-engineering
on a diff goes to `ponytail-review`, if that skill is installed; readability
stays here. For a bug, use [debug](../workflow/debug.md). For a wider
TypeScript refactor that may reshape APIs or types, use
[typescript-refactoring](../../project/typescript/typescript-refactoring.md). For an
end-of-session multi-lens review, use
[cleanup-web](../../project/frontend/cleanup-web.md) or
[cleanup-swift](../../project/swift/cleanup-swift.md).

## Three lenses

The pass reads the diff three ways, and applies a finding only when behavior
stays the same.

- **Reuse.** New code that repeats a helper, a standard-library call, or a
  guarantee the platform already keeps. Name the existing symbol.
- **Quality.** Redundant state, copy-paste with a small twist, a leaky
  boundary, dead code that is actually unused. A name that carries a domain
  rule stays.
- **Efficiency.** Repeated work, independent calls run one after another, new
  blocking work on a hot path. Outputs, errors, and order stay put.

Line count is not the score. A safety check at a trust boundary, a data-loss
guard, a security check, or an accessibility affordance is not a
simplification target.

## Fewer concepts, not fewer lines

The skill rejects brevity as a goal by itself. It favors clear control flow,
intent-revealing names, and boundaries that hide meaningful low-level detail.
It stops when the remaining changes are taste preferences, require broader
context, or would need new tests, migrations, or API changes.

Single-use helpers get the same test. A helper is a candidate for inlining when
its body communicates more than its name. It stays when the name captures a
domain rule, makes a predicate read naturally, or keeps branching and boolean
mechanics out of the high-level call site.

## It's working if

- The next reader has fewer concepts to hold in mind.
- The diff stays local to the recently modified code.
- Behavior and boundaries remain unchanged.
- The result is clearer at the call site, not just shorter in the helper file.

## Where it fits

A reach-for-it-anytime cleanup lens, and a step [work](../workflow/work.md)
runs after a build and before [adversarial-review](./adversarial-review.md).
Called that way, it reports and applies, then returns. It does not offer a
next rung. It complements [code-comments](./code-comments.md): simplify until
the code explains the mechanics, then comment only the surviving rationale.
Use [adversarial-review](./adversarial-review.md) when correctness, rather
than clarity, is the remaining question.
