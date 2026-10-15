# UX Flow Plan

Quickstart:

```bash
npx skills add edheltzel/Do-Skills --skill=do-ux-flow-plan
```

```bash
npx skills update do-ux-flow-plan
```

[Source](https://github.com/edheltzel/Do-Skills/tree/master/skills/project/frontend/do-ux-flow-plan)

## What it does

UX Flow Plan creates an architecture plan from the user's experience outward.
It first describes the current and desired journeys as flow trees, then attaches
concrete functions, files, reusable abstractions, and tests to those flows.

Its defining constraint is that architecture boundaries must be clear before
implementation anchors appear: function names and file paths support the plan;
they do not drive it.

## When to reach for it

You invoke this by typing `/do-ux-flow-plan` — the agent will not reach for it on
its own.

Reach for it when planning a feature whose user journey, system behavior, and
ownership boundaries need to be understood before proposing code changes. For a
design-system implementation rather than a feature-flow plan, use
[design-system](./design-system.md).

## Two flow trees, then anchors

The plan separates the current flow from the desired flow. Each tree follows the
same path: user action, system behavior, architectural layer, and finally a
relevant function or file. This makes changed behavior visible without treating
line numbers as the architecture.

After the trees are established, the plan assigns where a condition is detected,
where side effects live, which layer updates interface or status, and where
state is persisted or mutated. It also states whether the feature extends an
existing concept or deliberately remains independent.

## Decision-oriented output

The result ends with a small decision list: the recommended architecture,
rejected alternatives, and open questions or assumptions. It uses product and
architecture language unless framework terminology names a real boundary.

## Where it fits

UX Flow Plan is a user-invoked planning step before implementation work. It
makes the user-visible flow and architectural ownership explicit; once the
change is scoped, [coding-standards](../typescript/ts-standards.md) guides
the TypeScript or Effect implementation, while
[design-system](./design-system.md) covers the reusable UI foundation and
optional `DESIGN.md` spec.
