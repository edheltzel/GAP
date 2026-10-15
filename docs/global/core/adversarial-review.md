# Adversarial Review

Quickstart:

```bash
npx skills add edheltzel/Do-Skills --skill=do-adversarial-review
```

```bash
npx skills update do-adversarial-review
```

[Source](https://github.com/edheltzel/Do-Skills/tree/master/skills/global/core/do-adversarial-review)

## What it does

`adversarial-review` hunts for correctness bugs and regressions a change
introduced. It reports and does not edit. The caller applies. Every finding
names a severity, a `file:line`, why it breaks, a suggested fix, and the
concrete input that reaches the changed line. One reproducible bug beats ten
plausible suspicions.

## When to reach for it

Type `/do-adversarial-review`, or the agent reaches for it when reviewing a diff
before finalizing, pressure-testing recent changes, or answering “what could
break?”

Reach for it when the risk is incorrect behavior: missing guards, boundary
errors, stale callers, async ordering, swallowed failures, resource leaks, or a
half-finished refactor. For readability, use [simplify](./simplify.md). For
ideas and plans, use [redteam](./redteam.md). For feedback already left on a
pull request, use [git-pr-review-triage](../workflow/git-pr-review-triage.md).

## The concrete-trigger gate

The review starts from changed lines, traces affected callers, and states each
changed function's contract before judging its implementation. For every
candidate bug it constructs a failing input, follows that path through the
surrounding guards and sanitizers, and then tries to disprove its own finding.
Anything that cannot survive that self-attack is dropped.

The hunt emphasizes failure modes that fresh code commonly misses: empty,
zero, negative, huge, or absent inputs; external dependency failure; forgotten
`await`; check-then-act races; mutation visible to callers; changed return
shapes; and one branch updated without its twin.

## The report

Severity runs P0 (breaks on common input, or loses data) through P3 (narrow,
the caller's discretion). `file:line` points at a changed line. "Why it
breaks" says what fails and who hits it, not what the line already says. The
suggested fix is the smallest change that makes the trigger pass. The skill
proposes that fix and leaves the edit to the caller.

## It's working if

- Every finding points to a changed line and names a reproducible trigger.
- Callers are checked when signatures, units, nullability, or return shapes move.
- Speculative findings disappear during self-verification.
- A clean result is reported plainly instead of padded with weak concerns.

## Where it fits

A read-only correctness lens at the end of a change. [work](../workflow/work.md)
calls it after [simplify](./simplify.md) and applies the findings you accept.
In that caller mode it returns the report and does not edit. It complements
[simplify](./simplify.md), [code-comments](./code-comments.md), and
[parse-dont-validate](../../project/typescript/parse-dont-validate.md), each of
which owns a different concern. It also runs as the correctness lens inside
[cleanup-web](../../project/frontend/cleanup-web.md) and
[cleanup-swift](../../project/swift/cleanup-swift.md).
