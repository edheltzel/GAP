---
name: do-work
description: >-
  Execute a named plan, or the accepted list from a PR feedback file, unit by
  unit in the current checkout. Use when implementing from a plan or spec path,
  when triage hands over accepted review items, or when do-it needs the build.
  NOT FOR open-ended bugs (do-debug), shaping the plan (do-plan), or committing
  (do-commit).
---

# Work

Finish the named work in the checkout you are already in. A unit is one
Implementation Unit from the plan, or one accepted item from the feedback file.

Done when every in-scope unit is implemented and verified, simplify and review
have run or been skipped on purpose, accepted findings are applied and
re-verified, and the run either returns to its caller or offers the Next block.

## Stay here

Work in the current checkout. Do not create a branch, add a worktree, run a
unit-workspace script, commit, push, or open a PR. GitButler owns branches.
Commits happen only when the user picks `do-commit`.

Edit only files the unit owns. Leave other dirty files alone. If a unit needs
a file that already has unrelated edits, stop that unit and return
`status: blocked` with the path. Do not stash, reset, or include that file.

A project `AGENTS.md` may override artifact locations. Use that, then the
defaults below.

## Input

1. A plan path. Default name: `docs/plans/YYYY-MM-DD-<topic>-plan.md`. Units are
   the `### U1. <name>` packets under `## Implementation Units`. Each packet's
   `Change:` is the work, `Files:` lists the only paths that unit may create or
   edit, `Done when:` is the bar, `Verify:` is the check, and `Depends on:`
   names earlier units only.
2. A PR feedback file. Default name:
   `docs/plans/YYYY-MM-DD-pr-<number>-feedback.md`. Each `###` item under
   `## Accepted` is one unit, and its `Change:` is the work. `Reply:` lines and
   everything under `## Held` are not units.
3. `mode:return-to-caller` plus a path, from `do-it`. Strip the token. It
   changes only how the run ends.

A plan with no Implementation Units, or with a blocking question still open,
is not executable. Return `status: needs-human` and name `do-plan`, if
installed. Do not invent units. A feedback file with no `## Accepted` list is
the same: return `status: needs-human` and name `do-git-pr-review-triage`, if
installed.

Before editing, read a unit's `Change:`, `Files:`, `Done when:` (plans), and
`Verify:`. Run units in listed order, one at a time. A unit may touch only its
`Files:` list. A feedback item without `Files:` may touch only the file it
names.

## Each unit

1. Implement that unit only. Match patterns already in the files it owns. A
   settled decision in the plan stays settled. Remove code from approaches
   that did not pan out before calling the unit done.
2. Verify before the next unit. Run the unit's `Verify:` checks; for feedback,
   run the narrowest tests that cover the changed behavior. Run the app or
   command when the behavior is observable that way. A failure stops the unit.
   Fix it or return `status: blocked` with the command and failure. Do not
   weaken a test to go green.
3. UI a person can see: drive the page with `chrome-devtools-axi`, if
   installed. If it is not installed, say so and record what you could not see.

A unit is done when its checks pass and its requirements show up in the diff.

## After the last unit

Skip both passes when the diff is docs-only, or about ten lines of mechanical
change (format, rename, lockfile, generated file). Name the skip and why.

Otherwise, as caller:

1. Load `do-simplify`, if installed, with `mode:return-to-caller`. It reports
   and applies. If it is missing, do a short pass for duplicate or dead code
   in the diff and say you did.
2. Load `do-adversarial-review`, if installed, with `mode:return-to-caller`.
   It reports and does not edit. If it is missing, say so. A self-review is
   not that skill.
3. Apply accepted findings. Interactive: ask which findings to apply, with the
   harness's blocking question tool when it has one. `mode:return-to-caller`:
   apply a finding that names a concrete fix and does not need a product
   decision. A finding that needs a decision returns `status: needs-human`
   with that finding. Do not guess.
4. Re-run the verification from the units you touched.

`mode:return-to-caller` then returns `status: done | blocked | needs-human`
plus the plan or feedback path, and stops. Do not rename those status words.
`done` means the units are verified and review findings are applied or named
in the return. Any other ending uses the Next block.

## Next

End with this block. Run the next skill only when the user picks it. When a caller started this run (`mode:return-to-caller`, used by `do-it` and `do-work`), skip the block and return `status: done | blocked | needs-human` plus the artifact path.

Next: `do-commit` - local commit of the verified change, passing the plan or feedback path.

Other moves:
1. `do-commit cpp` - commit, push, and open a PR when the change should leave the machine.
2. `do-behavioral-testing` - the change lacks tests, if installed.
3. Review in Plannotator - annotate or approve the plan or feedback file first. (Only when that artifact is a file and Plannotator is installed.)
4. Stop - the change is already in the checkout.

Reply with a number, the label, or what you want instead.

Ask with the harness's blocking question tool when it has one, and drop the last line. Otherwise print the block and wait. A pick loads that skill now, in this session.

Review in Plannotator: if a `plannotator` tool exists, call it with `{action: "annotate", target: <path>, gate: true}`, end the turn, and wait for the feedback message. Otherwise run `plannotator annotate <path> --gate --json` with a long or no timeout (it blocks until the user acts). `approved`: offer the Next rung. `annotated`: revise from the feedback, then offer review again. `dismissed`: show the block again.
