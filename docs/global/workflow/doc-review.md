# Doc review

Quickstart:

```bash
npx skills add edheltzel/Do-Skills --skill=do-doc-review
```

```bash
npx skills update do-doc-review
```

[Source](https://github.com/edheltzel/Do-Skills/tree/master/skills/global/workflow/do-doc-review)

## What it does

`do-doc-review` reads one requirements or plan doc and fixes it in place. The pass looks for gaps, contradictions, unverifiable steps, and scope creep. An adequate doc gets no other edits.

The last line of the file is `Review complete`. That line is how a later skill knows this pass already happened. Called by [plan](./plan.md), it returns a status instead of a menu.

## When to reach for it

Type `/do-doc-review` and pass the file, or the agent reaches for it when you ask to review a requirements file or a plan before anyone builds from it. [plan](./plan.md) runs it automatically with `mode:return-to-caller`. Prose polish of a PRD is not this skill.

## One pass

The review is **one pass**. No personas and no second model. It fixes what the doc and the repo already decide, and it leaves a blocker when only you can close the gap. It does not add a unit the doc never committed to.

Standalone, an implementable plan offers [work](./work.md). Anything else offers [plan](./plan.md). Either way it waits for the pick.

## It's working if

- The file's last line is `Review complete`.
- Settled product decisions are unchanged.
- A caller got `status: done`, `blocked`, or `needs-human`, plus the path, and no menu.

## Where it fits

Shape, immediately after [plan](./plan.md). You can also run it on a [brainstorm](./brainstorm.md) requirements file. Attacking the idea belongs to [redteam](../core/redteam.md). Executing belongs to [work](./work.md).
