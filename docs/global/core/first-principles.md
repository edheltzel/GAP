# First Principles

Quickstart:

See [Manage skills](../../../README.md#manage-skills) for installation and updates.

[Source](../../../skills/global/core/do-first-principles/SKILL.md)

## What it does

`first-principles` breaks a problem into irreducible facts, challenges the constraints wrapped around it, and rebuilds a solution from what cannot change. It produces a parts breakdown, a constraint table, and a reconstructed solution.

Its defining distinction is that only hard constraints are immutable. Policies, conventions, market prices, and unvalidated beliefs are soft constraints or assumptions to challenge—not fundamental truths to inherit.

## When to reach for it

Type `/do-first-principles`, or the agent reaches for it automatically when inherited assumptions may be restricting the solution space.

Reach for it when you need to question what is truly required, escape an inherited form, or reason from fundamentals rather than analogy. For an adversarial critique of an existing idea or plan, use [redteam](./redteam.md) instead.

## Deconstruct, challenge, reconstruct

Deconstruct the problem into constituent parts and actual values. Challenge every stated constraint by classifying it as hard, soft, or an assumption. Then reconstruct an optimal solution from the hard constraints alone, optimizing function rather than the traditional form.

The output should make the limiting assumption visible: state the fundamental truths, explain the resulting solution, and identify whether the current approach is optimizing the right thing.

## Analysis before implementation

This is a reasoning skill, not an instruction to implement a fix. Use it to analyze a problem; when the task is simply to make a change, do the work directly. When assumptions prove wrong, rebuild from hard constraints rather than patching the inherited solution.

## Where it fits

A standalone analysis skill for decisions, designs, and stuck problems. [Redteam](./redteam.md) can use its constraint challenge to attack assumed boundaries, while this skill remains focused on rebuilding from the facts that survive scrutiny.
