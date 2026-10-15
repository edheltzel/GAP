---
name: do-doc-review
description: "One review pass over a requirements or plan doc: gaps, contradictions, unverifiable steps, and scope creep, fixed in place. Use when asked to review a requirements file or a plan before anyone builds from it. NOT FOR attacking the idea (do-redteam), writing the plan (do-plan), executing (do-work), PRD or spec prose (do-tech-writing), or UX flow trees (do-ux-flow-plan)."
---

# Doc review

One pass over one requirements or plan doc. No personas, no second model, no subagents. Judge whether the doc guides the agreed work, not how long it is. An adequate doc needs no other edits.

**Done:** the file is updated in place, its last line is `Review complete`, and Next was shown or a caller return was sent.

## Intake

Strip `mode:return-to-caller` if present and remember caller mode. Use the path the invocation names. If none, take the newest `*-requirements.md` or `*-plan.md` by mtime under the plans directory. That directory is the project `AGENTS.md` override, else `docs/plans/`.

The path must be readable before you classify or edit. If it is not, do not guess a replacement. Caller mode returns `status: blocked` and no path. Standalone says the file is missing and stops.

Classify by content, not by the filename. `## Implementation Units`, or any units with verification steps, means a plan. Behavior, scope, and success without those mean requirements. If the file is HTML, edit it as HTML. Do not insert markdown.

## Pass

Read the whole doc. Check only these:

- Gaps. A later reader must invent behavior, a boundary, or a check.
- Contradictions. Two parts that cannot both be true.
- Unverifiable steps. A success line or unit with no way to tell done.
- Scope creep. Work the request or the requirements did not ask for.

Fix in place what the doc and the repo already decide. Do not invent a product decision. Do not add a unit or a mechanism the doc did not already commit to. If a unit already says what done looks like, write the missing Verify line from that. If you would have to invent the check, flag the step as a Blocker.

Move unrequested work to Out of scope. If the user asked for it, leave it.

A gap only the user can close stays in the doc as an open question or a Blocker. That return is `needs-human`.

Keep `## todo.txt` immediately above the last line. Add the heading if this is a requirements or plan file and the heading is missing. Do not invent tasks.

Make `Review complete` the last line of the file. If it is already there, leave one copy, still last. Do not put it in a heading.

## Return

Caller mode, from `do-plan` or `do-it`: skip Next. Return `status: done | blocked | needs-human` plus the absolute artifact path. `done` when the pass finished and nothing is waiting on the user. `needs-human` when an open question or blocker remains. `blocked` when the file could not be read or the pass could not finish.

## Next

Standalone only. End with this block. Run the next skill only when the user picks it. When a caller started this run (`mode:return-to-caller`, used by `do-it` and `do-work`), skip the block and return `status: done | blocked | needs-human` plus the artifact path.

Fill the Next line when you show the block:

- The doc is a plan a later run can execute without inventing behavior: Next is `do-work` - build this plan in the current checkout, passing `<absolute artifact path>`.
- Otherwise: Next is `do-plan` - the doc is requirements, or the plan is not implementable yet, passing `<absolute artifact path>`.

Other moves:
1. The rung you did not put on the Next line, `do-plan` or `do-work`, if installed
2. Review in Plannotator - annotate or approve `<artifact>` first. (Only when the artifact is a file and Plannotator is installed.)
3. Stop - the artifact is already saved.

Reply with a number, the label, or what you want instead.

Ask with the harness's blocking question tool when it has one, and drop the last line. Otherwise print the block and wait. A pick loads that skill now, in this session.

Review in Plannotator: if a `plannotator` tool exists, call it with `{action: "annotate", target: <path>, gate: true}`, end the turn, and wait for the feedback message. Otherwise run `plannotator annotate <path> --gate --json` with a long or no timeout (it blocks until the user acts). `approved`: offer the Next rung. `annotated`: revise the doc from the feedback, then offer review again. `dismissed`: show the block again.
