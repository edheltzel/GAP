# ARCHITECTURE.md Generator

Quickstart:

```bash
npx skills add edheltzel/Do-Skills --skill=do-architecture-md
```

```bash
npx skills update do-architecture-md
```

[Source](https://github.com/edheltzel/Do-Skills/tree/master/skills/global/core/do-architecture-md)

## What it does

`architecture-md` generates an `ARCHITECTURE.md` that gives a newcomer — human or
agent — a mental map of a codebase, following matklad's principles and modelled on
rust-analyzer's doc. The premise is that the real bottleneck isn't writing code, it's
figuring out *where* to change it. The defining constraint is that the doc is short
and stable: it describes only what won't change often, names files and types instead
of linking to them (links go stale), and is meant to be revisited a couple of times a
year rather than synchronised with every PR.

## When to reach for it

Type `/do-architecture-md`, or the agent reaches for it automatically when you ask to
write an architecture doc, document the codebase structure, or draft a codemap.

Reach for it when you need the bird's-eye map of a whole codebase. For the lean
entry-point file that should *point* to this doc, and for the surrounding agent-first
documentation strategy, use [agent-docs](./agent-docs.md).

## What a good codemap does

- **Bird's eye first.** Open with the problem being solved and how data flows through
  the system at the coarsest level, before any module detail.
- **Answer "where's the thing that does X?"** A per-module section, 1–3 sentences
  each, naming the important types so symbol search lands.
- **Boundaries and invariants are the payload.** Mark API boundaries between layers,
  and call out what's deliberately *absent* — invariants are often expressed as
  absence and are impossible to divine from reading code.
- **Cross-cutting concerns last.** Error handling, testing, config — the things that
  live everywhere and nowhere — after the codemap, not woven through it.

The skill ships a template and a worked example, and targets ~300 lines: shorter docs
are more likely to be read and kept current.

## It's working if

- A newcomer can locate "the thing that does X" from the doc alone.
- API boundaries and deliberate absences are called out explicitly.
- Every section is stable enough to survive six months untouched.

## Where it fits

A run-once setup skill with a light twice-a-year refresh. It owns the architecture
layer of [agent-docs](./agent-docs.md)'s knowledge hierarchy, sitting one
level below the entry-point file that links to it.
