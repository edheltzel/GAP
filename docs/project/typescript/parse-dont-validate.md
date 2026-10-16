# Parse, Don't Validate

Quickstart:

```bash
npx skills add edheltzel/Do-Skills --skill=do-parse-dont-validate
```

```bash
npx skills update do-parse-dont-validate
```

[Source](https://github.com/edheltzel/Do-Skills/tree/master/skills/project/typescript/do-parse-dont-validate)

## What it does

`parse-dont-validate` is a type-driven design principle: transform unstructured
input into precise types at the system boundary so that illegal states can't be
represented downstream. The defining distinction is that a validator checks a
property and throws the knowledge away, while a parser checks the same property
and *preserves it in the type* — so a non-empty check returns `[T, ...T[]]`, not
`void`, and every caller downstream gets the proof for free. It is the TypeScript
application of Alexis King's essay of the same name.

## When to reach for it

Type `/do-parse-dont-validate`, or the agent reaches for it automatically when code
validates input, designs data types, defines signatures, or models a domain.

Reach for it whenever data crosses a boundary — request bodies, config, CLI args,
external APIs — and you're deciding how to type it. For applying this as one step
of a broader cleanup, use [typescript-refactoring](./typescript-refactoring.md)
(its Level 6 is this principle); for general authoring style,
[write-typescript](./write-typescript.md) leans on it at boundaries.

## Strengthen the input, don't weaken the output

A partial function becomes total in one of two ways. You can weaken the output
(return `T | undefined`) — easy to write, but every caller re-checks. Or you can
strengthen the input (demand `[T, ...T[]]`) — the check happens once at the
boundary and the type carries the proof thereafter. Always try the second first.

The practical toolkit: make illegal states unrepresentable with precise
structures (a `Map` where duplicates can't exist, a discriminated union instead
of a boolean flag); push parsing to the boundary so business logic never touches
`unknown`; treat `void`-returning validators as a smell; and use branded types
with smart constructors (`parseEmail(input): EmailAddress`) when true
unrepresentability is impractical.

## It's working if

- Functions demand specific types (`EmailAddress`, `[T, ...T[]]`) rather than raw
  `string` or `string[]`, and validators return refined types instead of `void`.
- There are no `// should never happen` comments and no redundant null checks deep
  in business logic — the boundary already guaranteed it.
- The program splits cleanly into a parsing phase (where bad input fails) and an
  execution phase (where input is known-good), never shotgun-parsed in between.

## Where it fits

A principle you reach for anytime you shape data, and one the other TypeScript
skills defer to at the edges: [write-typescript](./write-typescript.md) names it as the
type-driven idea it leans on, and
[typescript-refactoring](./typescript-refactoring.md) makes it the final polish
once structure and naming are clean.
