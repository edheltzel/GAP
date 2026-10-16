# Agent docs

Quickstart:

```bash
npx skills add edheltzel/Do-Skills --skill=do-agent-docs
```

```bash
npx skills update do-agent-docs
```

[Source](https://github.com/edheltzel/Do-Skills/tree/master/skills/global/core/do-agent-docs)

## What it does

`do-agent-docs` makes a repository legible to coding agents: the entry file, the docs tree that file routes through, the repo shape agents can work in, and a DOX install when that contract is the ask. The defining constraint is that it is a router, not one manual. Each mode loads one reference and leaves the others unread, because a giant instruction file crowds out the task.

## When to reach for it

Type `/do-agent-docs`, or the agent reaches for it when you write or improve `AGENTS.md`, onboard an agent, shape an agent-first repo, route `CLAUDE.md` or a docs tree, or install DOX.

Reach for it when the work is how the repo explains itself to an agent. For the architecture codemap that entry file should point at, use [architecture-md](./architecture-md.md). For typed boundaries, use [parse-dont-validate](../../project/typescript/parse-dont-validate.md).

## Four modes

- **Map, not manual.** `AGENTS.md` stays a short table of contents: commands, boundaries, non-obvious rules, pointers.
- **Catalogue.** Pointers over payload, one home per fact, section-scoped links. CI can check that links resolve. It cannot check that a doc still describes the code.
- **If the agent can't see it, it doesn't exist.** Knowledge lives in the repo. Rules that matter become lints and tests, not prose.
- **DOX install.** Merge the DOX contract into the root `AGENTS.md` and add a child file only where a folder has its own purpose.

## It's working if

- An agent can build, test, and lint from the commands in `AGENTS.md`.
- The entry file points instead of carrying the payload.
- Architectural rules that agents violate are enforced by CI, not only by a paragraph.

## Where it fits

A setup skill you revisit when conventions drift. It owns the agent-facing docs and repo shape. [architecture-md](./architecture-md.md) owns the codemap one level below the entry file.
