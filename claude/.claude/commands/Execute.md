---
name: atlas:Execute
description: Run an existing plan or ISA against the GitHub Projects board — pre-flight checks, move to In Progress, 4-tier deviation rules, inline verify, AC/close updates. Wraps the pm-tools skill's Execute recipe.
argument-hint: [path to plan/ISA and/or the issue number]
---

Use the **pm-tools** skill and follow its `references/Execute.md` recipe to implement an existing plan or ISA while keeping the board in sync.

Plan / issue: $ARGUMENTS

Follow the recipe exactly: link the issue and move it to In Progress, run the pre-flight checks (bun), execute the plan's tasks in order, apply the 4-tier deviation rules (fix inline / add validation / fix blockers+document / STOP for architecture), verify inline, then check off acceptance criteria and note close-readiness.

This command **runs an existing plan — it does not derive one.** If there is no plan or ISA to execute, stop and say so (deriving the plan is the Algorithm's job, not this recipe's).
