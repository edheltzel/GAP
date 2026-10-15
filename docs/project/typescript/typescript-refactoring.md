# TypeScript Refactoring

Quickstart:

```bash
npx skills add edheltzel/Do-Skills --skill=do-typescript-refactoring
```

```bash
npx skills update do-typescript-refactoring
```

[Source](https://github.com/edheltzel/Do-Skills/tree/master/skills/project/typescript/do-typescript-refactoring)

## What it does

`typescript-refactoring` turns a messy TypeScript codebase into clean code
through small, verified steps — assess first, then transform one smell at a time,
codebase-wide. Refactoring changes structure without changing behaviour, and the
defining constraint enforces exactly that: a commit changes structure *or*
behaviour, never both, and tests pass after every single step. Where
[write-typescript](./write-typescript.md) defines what good code looks like, this skill
defines how to get there from bad code without breaking it.

## When to reach for it

Type `/do-typescript-refactoring`, or the agent reaches for it automatically when a
task says "refactor this", "clean this up", "improve code quality", or you inherit
a tangled project.

Reach for it when the goal is *reshaping* existing code — extracting types,
killing `any`, renaming for intent, splitting God files. For writing new code in
the target style, use [write-typescript](./write-typescript.md) instead; for the type-driven
principle it leans on at boundaries, see
[parse-dont-validate](./parse-dont-validate.md).

## The iron rules and the priority ladder

- **Never mix behaviour and structure in one commit**, and never refactor without
  tests — write characterisation tests first if none exist.
- **One smell at a time, across the whole codebase.** Fix every `any`, then every
  long function, then every enum. Consistent, reviewable diffs beat one heavily
  reworked file.
- **Boring transformations only.** Rename, extract, inline, move. Clever
  restructuring is where behaviour quietly changes.
- **Fix in leverage order:** type safety → dead code → naming → structure →
  patterns → API boundaries. Each level makes the next safer.

For risky changes the skill uses the strangler-fig pattern — build the new path
alongside the old, migrate callers one at a time, delete the old when empty.

## It's working if

- Every commit is labelled `refactor:`, `chore:`, or `feat:` and does only that
  one kind of work — structure changes never ride along with a feature.
- Diffs are narrow and repetitive (the same fix applied across many files), and
  tests pass at each checkpoint.
- The agent presents an assessment — file counts, smells, hot paths, recommended
  priority — before touching anything.

## Where it fits

A project-level workhorse you reach for when quality has slipped, not a
run-once setup. It sits directly on top of [write-typescript](./write-typescript.md) (its
target state), leans on [parse-dont-validate](./parse-dont-validate.md) at Level
6 boundaries.
