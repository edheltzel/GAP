# No Comments

Quickstart:

This skill lives in this repo (`edheltzel/agent-config`, being renamed GAP) at `skills/global/core/do-no-comments`. `global/skills.json` lists it with `"source": "."`. Install and refresh it with:

```bash
gap sync
```

One skill by exact name:

```bash
npx skills add edheltzel/agent-config -s do-no-comments
```

[Source](https://github.com/edheltzel/agent-config/tree/master/skills/global/core/do-no-comments)

Upstream: [backnotprop/pstack](https://github.com/backnotprop/pstack/tree/main/skills/no-comments), a mirror of Lauren Tan's [pstack](https://github.com/cursor/plugins/tree/main/pstack)

## What it does

`no-comments` is a deletion pass over a diff. It spawns Comment Sicko, a
subagent with fresh eyes and one job: kill every comment that does not earn a
keep. Then the calling agent checks the report, rejects bad calls, and fixes
the code the deleted comments were apologizing for.

Comment Sicko keeps only five kinds of comment:

- Legal or license headers.
- Behavior forced by a dependency, platform, vendor, or protocol we cannot
  change.
- `// prettier-ignore`, and lint suppressions whose rule is faulty, pedantic,
  or style-only.
- Doc comments that define a public API contract.
- Issue or RFC links that explain a constraint code cannot express.

Everything else dies. A comment that explains a surprise in our own code is
deleted, and the code it explained is flagged `MUST KILL` for a rename,
extract, type, or redesign that makes the surprise obvious without prose.

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
- Every `do not remove` claim ends as a type, test, or lint that enforces it,
  or as an open item in the report.

## Where it fits

The opposite pass to [code-comments](./code-comments.md). Code-comments
writes the few comments that carry a why. No-comments deletes the rest and
turns their excuses into code changes. Run [simplify](./simplify.md) after it
when the `MUST KILL` fixes leave code to tidy.
