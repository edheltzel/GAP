# GitHub Projects CLI

Quickstart:

This skill lives in this repo (`edheltzel/agent-config`, being renamed GAP) at `skills/global/workflow/do-gh-pm`. `global/skills.json` lists it with `"source": "."`. Install and refresh it with:

```bash
gap sync
```

[Source](https://github.com/edheltzel/agent-config/tree/master/skills/global/workflow/do-gh-pm)

## What it does

`do-gh-pm` drives GitHub Projects v2 from the `gh` CLI and layers opinionated PM workflows over the primitives: interview-driven board scaffolding, an Epic→Feature→Task hierarchy with native sub-issues and dependencies, label policy, plan execution, next-work selection, and status reporting.

The defining constraint is **safe convergence**. Board fields, options, and repository labels are declared as typed archetypes and reconciled in dry-run-first passes; views and workflows remain explicit manual residuals because GitHub exposes no API for creating them.

## When to reach for it

Type `/do-gh-pm`, or the agent reaches for it automatically when work involves GitHub Projects: creating or linking a board, adding issues or pull requests, managing fields, decomposing an epic, running a plan against board state, or reporting progress.

Named recipes expose the major workflows directly: `ProjectSetup`, `Execute`, `Continue`, and `Status`.

## Prerequisites

Use `gh` 2.94 or newer, authenticated with the `project` scope:

```bash
gh --version
gh auth status
gh auth refresh -s project
```

The bundled TypeScript tools run with Bun.

## Interview-driven setup

`ProjectSetup` asks eight explicit questions: work kind, visibility, contributor model, cadence, estimation, roadmap, feedback intake, and Priority vocabulary. Those answers map deterministically to one of five archetypes plus ordered overlays.

`Tools/Scaffold.ts` then composes the board:

1. Print a no-write plan.
2. Copy a supplied golden project or create an empty project.
3. Set visibility and link the repository.
4. Run the reconcile engine first as a plan and then as a safe apply.
5. Install the optional feedback-agent files without overwriting differing files.
6. Print the views, workflows, iteration field, and secret configuration that remain manual.

The same answers always produce the same archetype, overlay order, and Priority scheme. `--apply` never implies prune: destructive option, field, and label removals stay behind the reconcile engine's separate confirmation gate.

## Archetypes and overlays

The five archetypes cover solo software, content pipelines, research tracking, Life OS tasks, and open-source repositories. They define Status vocabulary, fields, labels, and intended views.

Overlays add contributor triage, iterations, estimation, roadmap dates, or external feedback intake. Priority can use `P0–P3`, `High/Medium/Low`, or be omitted.

The feedback overlay consumes a machine-readable four-file manifest and installs Issue Forms plus routing and `/pm:*` workflows into a local repository checkout. Existing files with different content stop the scaffold before any GitHub project is created.

## Reconcile safety

The reconcile engine snapshots live fields, options, and labels, then separates changes into create, update, and delete buckets. Plan mode is the default.

GitHub replaces a single-select field's full option list on update. The engine therefore preserves live options and their IDs, never infers renames, and gates removals behind `--apply --prune --confirm`. Explicit renames preserve assignments. A board apply records resolved IDs and the spec hash under the repository's gitignored `.agents/atlas/artifacts/` directory.

## Daily board work

- `Hierarchy` creates parent/child issue structure and native dependencies.
- `Execute` moves a selected item through implementation while keeping acceptance criteria and board status aligned.
- `Continue` chooses work by board status, Priority, issue number, and open blockers, after reading the latest handoff in `docs/plans/handoffs/` when one exists.
- `Status` reports phase progress, status counts, recent closes, and what is next.

The board owns status. Memory and local planning context may explain work, but they do not override the board's current field values.

## It's working if

- Repeating the same interview yields the same resolved configuration.
- Dry-run mode executes no command and writes no file.
- Applying a scaffold creates or copies one project, links the intended repository, and converges safe board and label changes.
- Re-running reconcile after convergence reports no safe changes.
- Existing feedback workflow files are never overwritten silently.
- Manual views and automations are reported rather than falsely claimed complete.

## Where it fits

`do-gh-pm` is the coordination layer above implementation skills. `ProjectSetup` establishes a board once; `Continue`, `Execute`, and `Status` operate it day to day. Engineering skills govern how selected work is built, while `do-gh-pm` governs what is selected and how progress is represented.

For a status check across several projects, including this board and a todo.txt list, use `do-regroup`. Personal tasks that never touch a board belong in `do-todo`.
