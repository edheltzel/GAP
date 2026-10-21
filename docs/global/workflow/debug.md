# Debug

Quickstart:

See [Manage skills](../../../README.md#manage-skills) for installation and updates.

[Source](../../../skills/global/workflow/do-debug/SKILL.md)

## What it does

`do-debug` finds why something fails, then changes code only after you pick fix, diagnosis only, or rethink. It reproduces the failure before that choice. It does not create a branch, commit, or push.

## When to reach for it

Type `/do-debug`, or the agent reaches for it when something fails or runs slowly.

Reach for it for a bug. For a named plan, use [work](./work.md). For a redesign before a cause is known, use [brainstorm](./brainstorm.md).

## The gate

Reproduction comes first: a named check you have run, that fails on this symptom now and will pass once the cause is fixed. Reading the code is not that check.

Then the chain, from trigger to symptom, with a file and line and no gap. That chain is on screen before the gate. The three choices are fix, diagnosis only, and rethink. Rethink loads [brainstorm](./brainstorm.md) and this skill ends. Diagnosis only stops after the summary. Fix writes a regression test that fails for this cause, applies the smallest change that removes it, and re-runs the check.

## It's working if

- A check failed on the symptom before any edit.
- The chain names a file and line, with no gap.
- The regression test failed before the fix and passes after.
- Nothing was branched, committed, or pushed from this skill.

## Where it fits

This is a way into Build, not a separate loop. A verified fix offers [commit](./commit.md). A fix that grew into a feature offers [work](./work.md). Rethink hands off to [brainstorm](./brainstorm.md).
