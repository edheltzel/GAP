# Pragmatic TypeScript

Quickstart:

```bash
npx skills add edheltzel/Do-Skills --skill=do-write-typescript
```

```bash
npx skills update do-write-typescript
```

[Source](https://github.com/edheltzel/Do-Skills/tree/master/skills/project/typescript/do-write-typescript)

## What it does

`write-typescript` steers code toward TypeScript that is simple, clear, composable, and
soundly typed — small functions that combine, discriminated unions, explicit over
clever. It borrows the good parts of a functional style without the dogma: the
defining constraint is that readability wins over both cleverness and purity, so
it will mutate when mutation is simpler and reach for a plain `switch` before a
nested ternary. It is a style, not a rewrite engine — for restructuring existing
code, that's a different skill (below).

## When to reach for it

Type `/do-write-typescript`, or the agent reaches for it automatically when working in
`.ts`, `.tsx`, `.mts`, or `.cts` files.

Reach for it when you're *writing* TypeScript and want it to read cleanly the
first time. For systematically reshaping an existing codebase — extracting types,
tightening signatures, improving navigability — use
[typescript-refactoring](./typescript-refactoring.md). For keeping bundles and
libraries lightweight and dependency-free, use
[lean-ts-patterns](../tooling/lean-ts-patterns.md). For pushing validation to the type
system so illegal states can't be represented, use
[parse-dont-validate](./parse-dont-validate.md).

## What "pragmatically functional" looks like

- **Simple and clear.** Explicit code that scans easily beats dense code that has
  to be parsed. Chained `.filter().map()` when each step is obvious; a `switch`
  or early returns instead of stacked ternaries.
- **Small functions that compose.** Named predicates, comparators, and mappers so
  the call site reads like a sentence (`users.filter(isActive).sort(byCreatedDesc)`)
  — but trivial one-offs (`u => u.id`) stay inline.
- **Soundly typed, not ceremonially typed.** Lean on inference and discriminated
  unions; be explicit where it aids the reader, not to decorate.
- **Locality.** Keep a utility in the module that uses it; promote it to a shared
  file only when a second consumer actually appears.

Deeper material lives in the skill's `references/` — `modern-features.md`,
`patterns-in-the-wild.md`, and `performance.md`.

## It's working if

- New code favours small named functions and flat control flow over dense
  one-liners and nested ternaries.
- Types come mostly from inference and unions rather than scattered `as` casts.
- Helpers stay local until a real second caller forces them into a shared module.

## Where it fits

A reach-for-it-anytime standalone for day-to-day TypeScript authoring, and the
baseline the other TypeScript skills build on:
[typescript-refactoring](./typescript-refactoring.md) reshapes toward this style,
[lean-ts-patterns](../tooling/lean-ts-patterns.md) applies it to zero-dependency tooling,
and [parse-dont-validate](./parse-dont-validate.md) is the type-driven principle
it leans on at system boundaries.
