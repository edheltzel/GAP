# Design Patterns (Gang of Four)

Quickstart:

```bash
npx skills add edheltzel/Do-Skills --skill=do-design-patterns-gof
```

```bash
npx skills update do-design-patterns-gof
```

[Source](https://github.com/edheltzel/Do-Skills/tree/master/skills/global/core/do-design-patterns-gof)

## What it does

`design-patterns-gof` is a field guide to the 23 Gang of Four patterns, updated
for modern TypeScript, Python, Rust, and Swift — a map for naming and choosing
designs, not a catalog to apply. The defining constraint is that it is
opinionated against ceremony: in 2026 most GoF class hierarchies dissolve into
language features (closures, discriminated unions, generators, pattern matching,
DI containers), so the skill keeps the pattern *names* as shared vocabulary while
steering you away from the *scaffolding*.

## When to reach for it

Type `/do-design-patterns-gof`, or the agent reaches for it automatically when you
name a shape in review ("this is a Decorator chain"), weigh competing designs,
or question whether an abstraction earns its keep.

Reach for it to put a word on a design in a PR, to sanity-check indirection
before you introduce it, or to map a codebase that leans on patterns. This is a
vocabulary-and-judgment skill; for simplifying code once you've named its shape,
use [simplify](./simplify.md).

## Name the shape, then check if you need it

Two principles are the spine of the whole book: **program to an interface, not an
implementation**, and **favor object composition over class inheritance**. When a
pattern feels wrong it's usually because one of these was violated.

Before committing to any pattern, the skill forces two checks:

- **Does my language already have this?** Strategy is a function, Iterator a
  `for` loop, Command a closure plus data, Observer a pub/sub primitive, Visitor
  a `match` on a union. If the language answers, stop.
- **Have I seen this shape three times?** (Rule of Three.) Two occurrences aren't
  signal — the wrong abstraction costs more than the duplication.

Each pattern's reference file leads with **Modern Relevance** and **When NOT to
Use** — read those before the structure. Swift users get two extra guides:
`swift-idioms.md` (the language-native replacement per pattern) and
`swift-protocol-oriented.md` (why value types and protocols dissolve so many of
them).

## Signs you're over-abstracting

- One concrete implementation hiding behind a factory or interface — delete the
  indirection.
- The pattern appears before the duplication (UML before the second caller).
- Names like `AbstractSingletonProxyFactoryBean`, where the pattern chain has
  become the identity.
- "We'll need it later." You won't.

## Where it fits

A reach-for-it-anytime reference that gives design conversations a shared
vocabulary and a brake on premature abstraction. Its
language-native leanings echo the stack-specific craft skills — for the
TypeScript form of "prefer a union over a hierarchy," see
[write-typescript](../../project/typescript/write-typescript.md).
