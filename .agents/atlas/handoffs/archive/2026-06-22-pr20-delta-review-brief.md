# Delta-Review Brief — PR #20 fix commit (adversarial confirmation)

**Dispatched by:** Themis. You already reviewed PR #20 and wrote the findings. Now confirm the fix commit `0ce4897` genuinely resolves them — adversarially, not just test-green.
**Worktree:** `.claude/worktrees/feat+gh-projects-phase2-reconcile` · **Your prior findings:** `.agents/atlas/artifacts/2026-06-22-pr20-review.md`.
**Static only — do NOT run live-API mutations.** Ground in Read/Grep/git.

Verify each is truly resolved:
- **F1 (blocker):** Does the new id-preserving `--rename` path actually emit the option with its existing `id` + new name (in-place, non-destructive)? Do the NEW tests *assert* assignment preservation (item stays mapped, option id unchanged) rather than rubber-stamp? Is `countAssignments`' false-0 fixed — resolves the real `gh project item-list` field key, or returns an honest "unknown" instead of a confident `0`?
- **F3:** Is the `!s.exists || !s.fieldId` guard present and correct before `applyOptionUnion`?
- **F4:** Is `gh()` injectable, and does a test prove the `--prune`-without-`--confirm` gate throws BEFORE any mutation?
- **F5:** Are `optionLiteral`/`colorLiteral` now pure + tested? Is an unknown live colour a warning, not a mid-map throw?
- **F2:** Is the affected item→option mapping captured before destructive removal?

Also sanity-check no regression was introduced and the §6 core invariants you previously verified still hold.

**Output:** append a short **## Delta Verdict — <date>** section to the review artifact: per-finding RESOLVED/PARTIAL/UNRESOLVED + an overall **CLEAR-TO-MERGE** or **STILL-FIX**. Report the verdict to Themis. Do not modify code or merge.
