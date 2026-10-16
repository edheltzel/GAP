# Work

Quickstart:

This skill lives in this repo (`edheltzel/agent-config`, being renamed GAP) at `skills/global/workflow/do-work`. `global/skills.json` lists it with `"source": "."`. Install and refresh it with:

```bash
gap sync
```

One skill by exact name:

```bash
npx skills add edheltzel/agent-config -s do-work
```

[Source](https://github.com/edheltzel/agent-config/tree/master/skills/global/workflow/do-work)

## What it does

`do-work` executes a named plan, or the accepted items from a PR feedback file, one unit at a time. It works in the checkout you already have. It does not create a branch, add a worktree, commit, or push.

## When to reach for it

Type `/do-work`, or the agent reaches for it when a plan is approved, triage hands over accepted items, or `do-it` needs the build.

Reach for it when the work is already specified. For an open-ended bug, use [debug](./debug.md). For shaping the work, use [plan](./plan.md). For the commit, use [commit](./commit.md).

## The unit

A plan unit carries `Change:`, `Files:`, `Done when:`, and `Verify:`. `Files:` lists the only paths it may create or edit. A feedback unit is one `###` item under `## Accepted`; its `Change:` is the work and the item names the file. `Reply:` lines and items under `## Held` are not work.

The skill finishes that unit and runs its check before starting the next. After the last unit it runs [simplify](../core/simplify.md) and [adversarial-review](../core/adversarial-review.md), unless the diff is docs-only or about ten lines of mechanical change. It applies the findings you accept, re-runs the checks, and then offers a local commit.

`do-it` can start this skill with `mode:return-to-caller`. That run skips the menu and returns `done`, `blocked`, or `needs-human` plus the artifact path.

## It's working if

- Each unit's check passed before the next unit started.
- The checkout gained no branch and no push.
- Simplify and review ran, or the skip was named.
- The next offer is a local commit, not a push.

## Where it fits

This is the Build stage. It follows [plan](./plan.md) and precedes [commit](./commit.md). [debug](./debug.md) hands it a fix that grew into a feature. [git-pr-review-triage](./git-pr-review-triage.md) hands it the accepted list. `do-it` calls it and takes the result without the menu.
