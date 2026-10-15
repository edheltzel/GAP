---
name: do-debug
description: >-
  Reproduce a failure, confirm the root cause, then fix, diagnose, or rethink.
  Use when asked to debug or fix failing or slow behavior. NOT FOR implementing
  a named plan (do-work) or redesigning before a cause is known (do-brainstorm).
---

# Debug

Find the cause, then change code only after the gate. Done when the chain from
trigger to symptom has no gaps, and either a verified fix is ready for the
Next block or a diagnosis-only summary has been delivered.

Work in the current checkout. Do not create a branch, add a worktree, commit,
push, or open a PR. Commits happen only through `do-commit`.

`mode:return-to-caller` skips the gate and the Next block. Apply a fix that
matches the confirmed cause. A fix that would reverse a deliberate behavior
is `status: needs-human`, not an edit. Return `status: done | blocked |
needs-human` plus the absolute path to the current checkout as the change
artifact. Do not rename those words.

No credential, token, or connection string goes into a command, a file, or
the summary. Redact with `<REDACTED>`. If redacting hides the cause, say so
and ask. Do not un-redact.

## Reproduce first

Build a named check you have run: a test, a command, or a measured number for
a slowdown. It fails on this symptom now, and it will pass once the cause is
fixed. Reading the code and seeing the problem is not reproduction.

If you cannot build that check, say what you tried and what is missing, and
ask. Do not edit. Uncommitted work that can reach the failure is a hypothesis.
Test it by reading those edits. Do not stash or commit them.

Trace backward from the symptom to the first place valid state became invalid.
Observed values, not assumed ones. One hypothesis at a time, each with a
`file:line` and one observation that supports it. A failed fix invalidates
that hypothesis before the next try. Three failed fixes, or three hypotheses
with no confirmation: stop and say the cause was wrong. Do not shotgun.

## The chain, then the gate

Do not edit until you can state the chain, trigger through every step to the
symptom, with `file:line` and no "somehow". Write that chain, the files a fix
would touch, and where the regression test should live. Put the test where
existing tests already own that behavior. It must fail on this bug before the
fix and pass after. A test that fails because the change reverses intended
behavior is a rethink, not a test to rewrite.

Write that block in the same turn, then ask. If the user already said fix,
diagnosis only, or rethink, that is the answer. Otherwise ask with the
harness's blocking question tool when it has one:

1. Fix it now
2. Diagnosis only
3. Rethink the design - load `do-brainstorm`, if installed, and end this skill

## Fix

Diagnosis only: write the summary and stop. Do not offer `do-commit`.
Rethink: this skill ends when `do-brainstorm` loads.

On fix: write or strengthen the regression test and see it fail for this
cause. Then the smallest change that removes the cause. No drive-by cleanup.
Re-run the regression test, then the original reproduction check, then the
narrowest suite around the files you changed. A red check that the fix caused
is a wrong cause: revert that attempt and go back to the chain. Temporary
logs and asserts come out before the summary.

Then join Build: run `do-simplify`, then `do-adversarial-review`, each with
`mode:return-to-caller` and if installed, on the fix diff. Skip both for a
one-line or test-only fix. Apply the accepted findings and re-run the
regression test.

Summary, always: cause (`file:line`), what changed, the check that now passes,
and what you did not change.

## Next

End with this block. Run the next skill only when the user picks it. When a caller started this run (`mode:return-to-caller`, used by `do-it` and `do-work`), skip the block and return `status: done | blocked | needs-human` plus the artifact path.

Next: `do-commit` - local commit of the verified fix, passing the path this run was given, or the checkout when there was no file.

Other moves:
1. `do-work` - the fix grew into a feature, if installed.
2. Review in Plannotator - annotate or approve the artifact first. (Only when the artifact is a file and Plannotator is installed.)
3. Stop - the fix is already in the checkout.

Reply with a number, the label, or what you want instead.

Ask with the harness's blocking question tool when it has one, and drop the last line. Otherwise print the block and wait. A pick loads that skill now, in this session.

Review in Plannotator: if a `plannotator` tool exists, call it with `{action: "annotate", target: <path>, gate: true}`, end the turn, and wait for the feedback message. Otherwise run `plannotator annotate <path> --gate --json` with a long or no timeout (it blocks until the user acts). `approved`: offer the Next rung. `annotated`: revise from the feedback, then offer review again. `dismissed`: show the block again.
