# Lean TypeScript Patterns

Quickstart:

```bash
npx skills add edheltzel/Do-Skills --skill=do-lean-ts-patterns
```

```bash
npx skills update do-lean-ts-patterns
```

[Source](https://github.com/edheltzel/Do-Skills/tree/master/skills/project/tooling/do-lean-ts-patterns)

## What it does

`lean-ts-patterns` is a playbook for building lightweight, zero-dependency
TypeScript tools and libraries — CLIs, loggers, fetch wrappers, config mergers,
string utilities. The defining constraint is that you inline the handful of lines
you need instead of pulling in a package: 22 lines of ANSI codes rather than
chalk, `node:util.parseArgs` rather than commander, a native `fetch` wrapper
rather than axios. It is distilled from studying exemplary repos in the unjs and
antfu ecosystems, so the patterns are ones that ship in real libraries.

## When to reach for it

Type `/do-lean-ts-patterns`, or the agent reaches for it automatically when building
a CLI, logger, or utility from scratch, or when refactoring to shed dependencies.

Reach for it when weight and dependency count matter — a published library, a CLI
you don't want to bloat, infrastructure code that should be self-contained. For
general day-to-day TypeScript style, use [write-typescript](../typescript/write-typescript.md); for
reshaping an existing project's structure, use
[typescript-refactoring](../typescript/typescript-refactoring.md).

## The seven principles

- **Zero dependencies by design.** Prefer `node:` builtins and small inlined
  helpers; vendor at build time if you must.
- **Identity functions as type helpers.** `defineCommand`/`defineConfig` return
  their argument unchanged — their whole job is inference, and a `const` generic
  keeps literal types from widening.
- **One core primitive, compose everything.** Each library has a single core
  function (`splitByCase`, `normalizeWindowsPath`); the rest are thin wrappers.
- **Factories over classes.** Closures that capture config and return composable
  instances, with a `.create()` for layered defaults.
- **`Resolvable<T>` for lazy/async values**, smart defaults with escape hatches,
  and types that mirror runtime — if the code switches on a discriminant, the
  types should be conditional on the same one.

The skill ships copy-paste implementations (ANSI colours, `isPlainObject`,
`callHooks`, `normalizeWindowsPath`) and a reference table pointing at deeper
recipes in `references/`.

## Where it fits

A specialist you reach for when building tooling, not a whole-codebase pass. It
applies the [write-typescript](../typescript/write-typescript.md) style to the narrow world of
zero-dependency infrastructure, and complements
[typescript-refactoring](../typescript/typescript-refactoring.md) when the refactor in
question is "remove this dependency and inline what we actually use".
