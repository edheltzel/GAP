# No Comments

Quickstart:

See [Manage skills](../../../README.md#manage-skills) for installation and
refresh instructions. The bundled source is
[`do-no-comments`](../../../skills/global/core/do-no-comments/SKILL.md).

Upstream: [backnotprop/pstack](https://github.com/backnotprop/pstack/tree/main/skills/no-comments), a mirror of Lauren Tan's [pstack](https://github.com/cursor/plugins/tree/main/pstack)

## What it does

`do-no-comments` is a deletion pass over a diff. It uses Comment Sicko to
review comments, then the calling agent checks the report, rejects bad calls,
and fixes the code the deleted comments were apologizing for. Without a
subagent tool, the calling agent runs the same prompt in a fresh pass.

The [Comment Sicko prompt](../../../skills/global/core/do-no-comments/references/comment-sicko.md)
defines the keep exceptions, suppression rules, and `MUST KILL` flags.

## When to reach for it

Type `/do-no-comments`. It never runs on its own.

Run it on a finished diff before review, or on files that collected
narration, commented-out code, and `do not remove` warnings nobody can
explain. With no files named, it takes the current diff against the default
branch, working tree included.

## It's working if

- The report gives a deletion count, one line per `MUST KILL` flag, and the
  comments it skipped.
- License headers and real vendor workarounds survive. Narration and
  commented-out code do not.
- No application code changed inside Comment Sicko's pass. The calling agent
  makes the fixes, smallest root cause first.
- Constraint comments follow the skill's
  [approval and reporting rules](../../../skills/global/core/do-no-comments/SKILL.md#steps).

## Where it fits

A stricter deletion pass alongside [code-comments](./code-comments.md).
Code-comments guides comment authoring; no-comments applies Comment Sicko's
keep exceptions, so even a why-comment about our own code can be removed.
Run [simplify](./simplify.md) after it when the `MUST KILL` fixes leave code to tidy.
