# Red Team

Quickstart:

See [Manage skills](../../../README.md#manage-skills) for installation and updates.

[Source](../../../skills/global/core/do-redteam/SKILL.md)

## What it does

`do-redteam` attacks an idea, judges one named option against the project's evidence, or develops competing alternatives and synthesizes a choice. It does not implement the winner.

A verdict has to be earned from the code, tests, docs, and recorded decisions. A claim in the chat is a pointer to check, not proof. Choosing between alternatives is in scope once those alternatives exist, or once this skill has developed them. Attacking a code diff is not.

## When to reach for it

Type `/do-redteam`, or the agent reaches for it automatically when a plan, strategy, or argument needs a deliberate challenge, a verdict on one named option, or a choice that still needs alternatives built out.

Reach for it to poke holes in a proposal, surface its strongest objection, judge one named candidate, or run a competition when the options are still too thin to compare. For rebuilding a problem from fundamental truths, use [first-principles](./first-principles.md). For bugs in a diff, use [adversarial-review](./adversarial-review.md). For CLAUDE.md and other instruction-file bloat, use `do-bitter-pill`.

## Three ways in

**Parallel analysis** stress-tests an existing argument. The deliverable is an eight-point steelman and an eight-point counter-argument that beats that steelman, not a strawman, with a remediation for each real weakness.

**Judge** is a verdict on one named candidate. It does not invent rival options and it does not implement. If a missing fact would flip the verdict, it returns blocked and names that fact.

**Adversarial validation** develops at least two independent alternatives to one shared brief, or compares ones the caller already supplied, then has a critic assess them and synthesizes one coherent approach. A hard-constraint miss cannot be outvoted. If the field is too small, or only the user can supply the deciding preference, it returns incomplete or unresolved instead of inventing a winner.

## Where it fits

A model-invoked challenge for plans and arguments. It can draw on [first-principles](./first-principles.md) to class constraints as hard, soft, or assumption before the attack. [adversarial-review](./adversarial-review.md) owns correctness bugs in a change set. `do-bitter-pill` owns instruction-file bloat.
