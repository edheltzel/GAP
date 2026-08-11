---
name: Loop
version: 1.1.0
description: "Runs a bounded, supervised iteration protocol that improves a target against explicit exit criteria while recording attempts and evidence. USE WHEN loop, iterate, refine, multiple passes, keep improving, revisit, or rework."
disable-model-invocation: true
effort: medium
---

# Loop

## Purpose

Improve one target through bounded iterations without relying on a particular reasoning framework, instruction system, or named companion skill. Supervision is the default. Each iteration starts from recorded evidence, makes one coherent change, verifies it, and pauses for human review.

## Required inputs

- **Target:** file, directory, artifact, or clearly bounded output.
- **Goal:** observable improvement to pursue.
- **Exit criteria:** explicit checks that can pass or fail.
- **Iteration limit:** default 3, maximum 20.
- **Allowed actions:** tools, files, and side effects permitted by the current harness and repository.

If target, exit criteria, or iteration limit is missing or ambiguous, clarify before changing anything.

## Ledger

Persist state at:

```text
.agents/atlas/artifacts/loops/<target-slug>.md
```

Create parent directories when needed. The ledger is local and must contain:

```text
# Loop: <target>
Goal: <goal>
Exit criteria:
- [ ] <criterion>
Iteration limit: <N>
Current iteration: <N>
Status: running | awaiting-review | passed | failed-closed

## Iterations
### <N>
Hypothesis: <one expected improvement>
Change: <what changed>
Verification: <commands or review evidence>
Result: improved | unchanged | regressed | blocked
Human direction: <approval or redirect>

## Rejected approaches
- <approach>: <reason and evidence>
```

Resume only from an existing ledger whose target and criteria still match current reality.

## Supervised protocol

For each iteration:

1. Read the target, applicable repository instructions, ledger, and current verification evidence.
2. Select one hypothesis that advances a stated exit criterion and does not repeat a rejected approach.
3. Make the smallest coherent change allowed by the scope.
4. Run the focused verification needed to detect improvement and regressions.
5. Append evidence and result to the ledger.
6. Present the result to the human and wait for approval, redirection, or stop before another iteration.

Never silently continue past the review seam in supervised mode.

## Autonomous Agent option

Use autonomous mode only when the user explicitly requests it and the harness exposes an Agent or subagent task tool. Launch one autonomous worker with:

- the exact target and allowed paths
- the complete exit criteria
- the ledger path
- the remaining iteration budget
- required verification commands
- an instruction to stop on ambiguity, regression, missing dependency, or exhausted budget

The worker must return evidence after each bounded attempt. The supervising agent remains responsible for reading the ledger, validating actual changes, and enforcing the same fail-closed criteria. If no Agent facility exists, remain supervised rather than simulating autonomy.

## Fail-closed exit criteria

Mark `passed` only when all stated criteria pass, required verification succeeds, no known regression remains, and the ledger contains evidence for the final state.

Mark `failed-closed` and stop when any of these occurs:

- the iteration limit is reached without all criteria passing
- verification fails or cannot run
- an iteration regresses a previously passing criterion
- required input or a dependency is missing
- the next action exceeds allowed scope or requires an unapproved side effect
- evidence is inconclusive

Do not describe partial improvement as completion. Preserve the ledger and report the exact blocker or failed criterion.

## Status and resume

A status request reads the ledger and reports target, current iteration, passing and failing criteria, last evidence, rejected approaches, and next review decision. A resume request revalidates the target, criteria, repository instructions, and working tree before continuing within the remaining budget.
