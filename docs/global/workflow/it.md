# LFG

Quickstart:

See [Manage skills](../../../README.md#manage-skills) for installation and updates.

[Source](../../../skills/global/workflow/do-it/SKILL.md)

## What it does

`do-it` is the LFG runner: one explicit end-to-end ask, no menus, from the request to a shipped artifact. A feature goes plan, then build, then ship. A bug skips the plan and starts on the fix path.

It stops only for a decision the user has to make: missing plan input, a debug rethink, or a no-mistakes ask-user gate. It never lands or merges. `/do-it commit` stops at a local commit instead of opening a PR.

## When to reach for it

Type `/do-it`, or the agent reaches for it when you say LFG or let's go on autonomous end-to-end work. It will not fire on a vague "continue".

Reach for this when you want the whole loop run without a stop at each skill. For a step you review before the next, use [plan](./plan.md), [work](./work.md), or [commit](./commit.md). A bug you want diagnosed before anyone edits belongs in `do-debug`, not here.

## The run

**LFG** means the route is already chosen. Feature: `do-plan`, then `do-work`, then `do-commit cpp` (commit, push, PR, CI green). Bug: `do-debug` on the fix path, then the same ship step. Children report back instead of offering their own menus.

The run ends with a short report: what ran, artifact paths, PR URL if one exists, and why it stopped if it did. Then a Next block. After a PR, next is review triage when comments land, or land only when you say go. After `/do-it commit`, next is publishing that commit.

Leaving mid-run is `/do-handoff`, not a substitute summary.

## It's working if

- You were not asked to pick a skill in the middle of the run.
- A feature produced a plan path, a build, and either a PR URL or a stated stop.
- A bug did not grow a new plan.
- Nothing merged.

## Where it fits

This is the runner across the workflow, not a stage of it. Shape, build, and ship stay in [plan](./plan.md), `do-work`, and [commit](./commit.md). Parking the session for someone else is `do-handoff`. Review comments that arrive later go to [git-pr-review-triage](./git-pr-review-triage.md).
