# TypeScript Coding Standards

Quickstart:

This skill lives in this repo (`edheltzel/agent-config`, being renamed GAP) at `skills/project/typescript/do-ts-standards`. `global/skills.json` lists it with `"source": "."`. Install and refresh it with:

```bash
gap sync
```

One skill by exact name:

```bash
npx skills add edheltzel/agent-config -s do-ts-standards
```

[Source](https://github.com/edheltzel/agent-config/tree/master/skills/project/typescript/do-ts-standards)

## What it does

This skill provides a correct-by-construction method for changing TypeScript:
establish local rules, trace caller-visible behavior, design public types and
services, implement the complete change, then verify through public interfaces.

Its defining constraint is that expected failures are explicit values and
external data is parsed into meaningful application or domain types before it
reaches inner code.

## When to reach for it

Type `/do-ts-standards`, or the agent reaches for it automatically when a
TypeScript change needs the project's engineering standards.

Reach for it when a change crosses types, errors, services, schemas, or tests
and needs a coherent end-to-end design. For Effect code, use
[backend](../backend/backend.md) (its Effect guide). For a narrower boundary-first approach to
untrusted data, use [parse-dont-validate](./parse-dont-validate.md); for general
TypeScript authoring style, use [write-typescript](./write-typescript.md).

## The changed-behavior loop

The work starts at the observable operation rather than at a convenient file.
It maps inputs, decisions, effects, failures, state transitions, external
representations, and test surfaces to their owners. Public inputs, outputs,
expected errors, and service interfaces are then made explicit before the
implementation is filled in.

The skill keeps unrelated legacy behavior contained at the nearest existing
edge. New abstractions must pass a deletion test: removing one should spread
meaningful complexity into callers, not merely remove a name.

## Where it fits

This is the broad engineering baseline for TypeScript changes. Effect code also
uses [backend](../backend/backend.md) (its Effect guide). [typescript-refactoring](./typescript-refactoring.md)
focuses on reshaping existing TypeScript code.
