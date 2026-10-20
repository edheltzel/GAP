---
name: do-no-comments
description: "Spawn Comment Sicko, fix accepted findings, and offer encodings for claimed constraints."
disable-model-invocation: true
author: Lauren Tan
license: MIT
---

# No comments

Spawn Comment Sicko. Act on accepted findings.

Defer to Comment Sicko's fresh perspective.

## Scope

Use the caller's files or diff. Otherwise use the current diff against the base branch, default the repo's default branch (`origin/HEAD`), including the working tree.

## Steps

1. Spawn Comment Sicko. Pass the scope. Pick its reasoning level and fast or standard speed for the job. Do not restate its rules. Where `comment-sicko` is a registered subagent, spawn it by name: `Task` with `subagent_type: "comment-sicko"` in Cursor with the pstack plugin, `task` with agent `comment-sicko` in OMP with PStack-for-OMP. Otherwise spawn a general subagent: `Agent` in Claude Code (`subagent_type: general-purpose`), `task` in OpenCode (`subagent_type: general`), `spawn_agent` in Codex, `dispatch` in Empryo (detect it per GAP `global/docs/agents/harness-detection.md`). Its prompt is [`references/comment-sicko.md`](references/comment-sicko.md) verbatim, then the scope. If your harness has no subagent tool, as in Pi without an extension, run that prompt yourself in a fresh pass over the scope.
2. Inspect its report and diff. Reject application-code edits, scope escapes, exception-protected deletions, misstated `MUST KILL` reasons, and flags that treat kept intentional code as guilty. Reshape flags on our-code surprises stay actionable. Do not restore those comments. A keep survives only with proof it is about something we cannot change. Audit missed scoped lint and TypeScript suppressions. Correctness or safety suppressions stay actionable `MUST KILL`s. Restore deletions only with exact exceptions and scoped proof. Before accepting thin `IMPORTANT` or `do not remove` kills or keeps, trace their symbol: the how or why skill (`ps-how`, `ps-why`) if installed, otherwise its callers and `git log -L` history yourself. If a kill is ambiguous, do not restore. If a keep is refuted or still ambiguous, delete it. Revert and rerun one rejected report with the failure named. Reject a second, report it open, and fail `/do-no-comments`.
3. Fix trivial accepted flags directly by deleting a dead path, dropping a parameter, or using the real API. If any fix needs a shape, sketch it once for the accepted set and surrounding code, with `ps-architect` if installed. Stop at the sketch. The sketch shapes. Step 4 implements.
4. Implement the smallest root-cause fix in scope. Remove every named workaround. If the root cause is out of scope, land the smallest in-scope fix and report the rest open. Fix real causes and redesign as if the requirements always existed (`ps-principle-fix-root-causes` and `ps-principle-redesign-from-first-principles` go deeper, if installed). That guides intent only. It never authorizes widening the fence nor fixing instances outside it. Never bolt on symptom guards.
5. Constraint comments say `do not remove`, `do not change wording`, or `talk to X before changing`. Leave keeps about things we cannot change. Offer the cheapest in-scope type, runtime, test, or CI lint. Wait for interactive approval. Unattended and eval require caller pre-approval. If approved, encode then delete. Otherwise delete, report the constraint open, and sketch out-of-scope work.
6. Report the deletion count, restored comments, reruns, architect sketch, fixes, encoding offers, encodings, unenforced constraints, and other open work.

## Related skills

- Writing the why-comments that survive this pass: `do-code-comments`, if installed.
- Simplifying code once its comments are gone: `do-simplify`, if installed.
