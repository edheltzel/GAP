# Brainstorm

Quickstart:

```bash
npx skills add edheltzel/Do-Skills --skill=do-brainstorm
```

```bash
npx skills update do-brainstorm
```

[Source](https://github.com/edheltzel/Do-Skills/tree/master/skills/global/workflow/do-brainstorm)

## What it does

`do-brainstorm` turns a vague idea into a requirements file. The file records behavior, who it is for, what is out, and a success test. It does not choose libraries, file layout, or a build sequence.

The dialogue asks one question at a time, and only a decision the request and the repo cannot settle. The file is `docs/plans/YYYY-MM-DD-<topic>-requirements.md`, or the directory the project `AGENTS.md` names instead.

## When to reach for it

Type `/do-brainstorm`, or the agent reaches for it when you want to brainstorm or scope an idea. If you have no idea yet, it offers a few grounded options, you pick one, and then the dialogue runs. A feature whose behavior is already settled belongs to [plan](./plan.md).

## Requirements, not a plan

**Requirements** are the behavior, the boundary, and a success test a later reader can apply without guessing. Technology and task order stay out of the file. The file ends with `## todo.txt`. Those lines go onto the repo `todo.txt` with `do-todo`, if that skill is installed.

When the Ready check passes, the skill offers `do-plan` and waits. It does not load the next skill until you pick it.

## It's working if

- The file states behavior, boundary, and a success test someone can fail.
- It has no implementation units.
- The run stopped on the Next block instead of editing code.

## Where it fits

Shape starts here. [plan](./plan.md) writes the how. [doc-review](./doc-review.md) checks the doc. [work](./work.md) executes. Attacking the idea belongs to [redteam](../core/redteam.md). PRD and spec prose belongs to `do-tech-writing`. A UX flow tree belongs to `do-ux-flow-plan`.
