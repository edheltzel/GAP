# GitHub Projects CLI

Quickstart:

```bash
npx skills add edheltzel/Do-Skills --skill=do-gh-pm
```

```bash
npx skills update do-gh-pm
```

[Source](https://github.com/edheltzel/Do-Skills/tree/master/skills/global/workflow/do-gh-pm)

## What it does

`gh-pm` drives GitHub Projects (v2) from the `gh` CLI — creating boards,
adding items, defining custom fields, and editing field values — and layers a set
of opinionated PM recipes on top: board bootstrap, an Epic→Feature→Task issue
hierarchy with real sub-issue linking, a label policy, running a plan against the
board, picking next work, and status reporting. The defining constraint is that
it works entirely through `gh project` commands and GraphQL, so there's no web
clicking and every board change is a scriptable, repeatable command.

## When to reach for it

Type `/do-gh-pm`, or the agent reaches for it automatically when a task involves
GitHub Projects — adding issues or PRs to a project, creating fields,
bootstrapping a board, decomposing work into epics and features, executing a plan
with board updates, or reporting progress.

The recipes are also exposed as named entry points — `ProjectSetup`, `Execute`,
`Continue`, `Status` — for jumping straight to one phase of the workflow.

## Prerequisites

The `gh` CLI, authenticated with the `project` scope. Check with `gh auth status`
and add the scope if missing:

```bash
gh auth refresh -s project
```

## Two layers: primitives and recipes

- **gh primitives.** The mechanical vocabulary — `project create`, `item-add`,
  `item-edit`, `field-create`, and the `--format json | jq` patterns for pulling
  the project, field, and option IDs that field edits require. This is the
  bottom layer everything else composes from.
- **PM recipes.** Opinionated flows in `references/`: `ProjectSetup` (one-time
  board bootstrap — fields, views, automations), `Hierarchy` (the
  Epic→Feature→Task model with `addSubIssue` linking), `Labels` (Phase and
  Priority live as board fields, not labels), `Execute` (run a plan against the
  board with tiered deviation rules), `Continue` (pick next work by status →
  priority → dependencies, after reading the latest handoff in `docs/plans/handoffs/` when one exists), and `Status` (progress report).

The through-line: **the board is the source of truth**, and every item's state is
set by editing a field, not by dragging a card.

## Where it fits

A standalone you reach for whenever work is tracked on a GitHub Projects board —
part one-time setup (`ProjectSetup`), part daily driver (`Continue`, `Execute`,
`Status`). It sits at the coordination layer above the code-craft skills: it
decides and tracks *what* gets worked on, while the engineering skills govern how
the work itself is done.

For a status check across several projects, including this board and a todo.txt list, use `do-regroup`. Personal tasks that never touch a board belong in `do-todo`.
