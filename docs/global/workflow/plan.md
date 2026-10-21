# Plan

Quickstart:

See [Manage skills](../../../README.md#manage-skills) for installation and updates.

[Source](../../../skills/global/workflow/do-plan/SKILL.md)

## What it does

`do-plan` writes an implementation plan a later run can execute. The plan is ordered units. Each unit names the change, when it is done, and a check that can fail.

It does not invent product behavior the requirements left open, and it does not implement. There is no research-agent farm, no HTML renderer, and no model-tier switch. After the file is written, the skill runs [doc-review](./doc-review.md) on it, then waits for you to pick the next move.

## When to reach for it

Type `/do-plan`, or the agent reaches for it when you ask to plan, break down implementation, or plan from a requirements file. [brainstorm](./brainstorm.md) hands it the requirements path. A bug, or a feature whose behavior is still unsettled, is not this skill.

## Units and checks

An **Implementation Unit** is one ordered change under `## Implementation Units`. `Depends on` names an earlier unit or none. `Files` lists the paths that unit may create or edit. `Verify` is the check a later run can fail, including any test the plan names. A step with no way to fail is not in the plan. A Blocker line is a blocking question, and build waits while one is open.

The file is `docs/plans/YYYY-MM-DD-<topic>-plan.md`, or the directory the project `AGENTS.md` names instead. Passing an existing plan path revises that file. The requirements file stays put.

When Plannotator is installed, the next move is to approve the plan there. Approval offers [work](./work.md). Without Plannotator, the next move is `do-work` directly.

## It's working if

- Every unit has a Verify line.
- Product behavior matches the requirements, or the plan names the gap and does not guess.
- The run did not edit product code.

## Where it fits

Shape, after [brainstorm](./brainstorm.md) and before [work](./work.md). [doc-review](./doc-review.md) edits the plan in place and leaves `Review complete` as its last line. `do-it` can run this skill with `mode:return-to-caller` and take the status back instead of a menu.
