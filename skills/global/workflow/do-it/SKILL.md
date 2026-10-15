---
name: do-it
description: >-
  LFG runner. Let's go: take an explicit end-to-end ask from request to done
  with no menus. Use when the user invokes do-it, says LFG, or says let's go
  on autonomous end-to-end work. A feature runs do-plan, do-work, then
  do-commit cpp. A bug starts at do-debug's fix path. `/do-it commit` stops
  at a local commit. Do NOT use for step-by-step review; use `do-plan`,
  `do-work`, `do-debug`, or `do-commit` for those.
---

# LFG

One run, no menus. The user already chose end-to-end. Fire only on an explicit LFG, "let's go", `/do-it`, or an ask to take this all the way. A vague "continue" is not a trigger.

Pass `mode:return-to-caller` to every child. Children return `status: done | blocked | needs-human` plus the artifact path. Do not print their Next blocks. Do not land, merge, or post review replies.

## Route

1. **Bug.** A concrete report of failing or wrong behavior. Invoke `do-debug`, if installed, on the fix path (not diagnosis-only). A verified fix is the work source: skip `do-plan` and `do-work`. `rethink` or any `needs-human` stops the run.
2. **Feature.** Anything else that changes the product. Invoke `do-plan`, if installed, with the request. Missing plan input (`blocked` or `needs-human`, including no requirements to plan from) stops. Do not search `docs/plans/` for a plan you were not given. A plan path the user named is the work source; do not replan over it.
3. **Build.** On the feature route only, invoke `do-work`, if installed, with the plan path from step 2. `do-work` owns simplify and review. Anything but `done` stops.
4. **Ship.** Invoke `do-commit`, if installed. Default is `cpp` (commit, push, PR, CI green). `/do-it commit` is a local commit and stops there. A no-mistakes `ask-user` gate, or any `needs-human`, stops. Never pass `land`.

A missing child skill stops the run. Say which one.

If the user stops mid-run, the exit is `/do-handoff`. Do not invent a substitute summary.

## Stop

Stop, say why, and leave nothing newly pushed, when:

- plan input is missing, or planning returns `blocked`
- debug returns `rethink` or will not fix
- a child returns anything but `done`
- no-mistakes parks on an ask-user gate

## Report

Short. What ran, each artifact path, the PR URL when one exists, and why a stop happened. Then the Next block. On `/do-it commit`, Next is `do-commit cp` (publish the local commit). Otherwise use the block below.

## Next

End with this block after the report. Run the next skill only when the user picks it.

Next: `do-git-pr-review-triage` - when review comments land, passing the PR URL.

Other moves:

1. `do-commit land` - only when the user says go.
2. Stop - the PR is already open.

Reply with a number, the label, or what you want instead.

Ask with the harness's blocking question tool when it has one, and drop the last line. Otherwise print the block and wait. A pick loads that skill now, in this session.
