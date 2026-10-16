# Execute — run an existing plan against the board

Lifecycle for implementing an **already-written** plan (a plan file in the repo or a project plan) while keeping the board in sync. Adapted from `execute.md`. **Runner-agnostic** — a dispatched worker or the operator directly can follow it. **Plan-aware:** it *consumes* an existing plan; it never derives one. (Deriving the plan is the planning phase, not this recipe's.)

## 0. Link to the issue + move to In Progress

```bash
gh issue view NUMBER --repo OWNER/REPO            # read the issue + its AC

# move the board item to In Progress (IDs from `gh project field-list PROJECT_NUM --owner OWNER --format json`)
ITEM_ID=$(gh project item-list PROJECT_NUM --owner OWNER --format json \
  --jq '.items[] | select(.content.number == NUMBER) | .id')
gh project item-edit --project-id PROJECT_ID --id "$ITEM_ID" \
  --field-id STATUS_FIELD_ID --single-select-option-id IN_PROGRESS_OPTION_ID
```

## 1. Pre-flight checks

Use the package manager the repo's lockfile names (`bun.lock` -> bun, `pnpm-lock.yaml` -> pnpm, `package-lock.json` -> npm, and so on). Skip the dependency check for repos without one.

```bash
git status --porcelain        # working tree should be clean
git branch --show-current
```

- Dirty tree → stop and ask: stash, commit, or abort.
- Missing deps → run the repo's install command and continue.
- Env file the plan or README requires is missing → stop and ask.

## 2. Execute tasks in order

For each task in the plan: read related files first, implement following existing patterns, verify syntax/imports/types as you go. Then create and run the tests the plan specifies, and run every validation command the plan lists — fix-and-rerun until each passes.

## 3. The 4-tier deviation rules

Deviations from the plan will happen. Decide how to handle each:

1. **Fix bugs inline** *(OK, no ask)* — spot a bug in existing code while implementing? Fix it in the same change; note it in the report.
2. **Add validation at boundaries** *(OK, no ask)* — plan omits input validation / type guards / error handling at a system boundary? Add it.
3. **Fix blockers pragmatically** *(OK, but document)* — task can't be done as specified (missing dep, wrong assumption)? Take the simplest working alternative and record what changed and why.
4. **STOP for architecture changes** *(NOT OK to deviate)* — would completing the task require new dependencies, a schema restructure beyond spec, or auth/middleware changes? **Stop and ask before proceeding.**

## 4. Update the issue + board

```bash
# check off completed AC in the issue body
gh issue edit NUMBER --repo OWNER/REPO --body "…updated body with [x] AC…"
# comment a summary
gh issue comment NUMBER --repo OWNER/REPO \
  --body "Implementation complete. Files changed: … Ready for commit."
```

- All AC met → note the issue is ready to close (close **after** the commit lands).
- Partially done → leave In Progress.

## 5. Hand off to commit

Confirm: all plan tasks done, all validations pass, issue updated. The actual commit/close is a separate step (the repo's commit flow, or `do-commit` if installed). Closing the issue happens with the commit, not here.

## Anti-scope

This recipe runs a plan; it does **not** write one. If there is no plan yet, stop — go derive it first (a planning agent). Keep this a thin executor, not a second planning engine.
