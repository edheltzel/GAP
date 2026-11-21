# Comments

Adapted from pstack's [no-comments](https://github.com/cursor/plugins/blob/main/pstack/skills/no-comments/SKILL.md) skill.

## Rules

- Write no comments in code. Names, types, and tests carry the intent.
- A comment that explains a workaround or a surprise in our own code is a smell. Reshape the code or fix the root cause, then delete the comment. Do not add a symptom guard instead.
- Keep a comment only when it is about something we cannot change: an external API or spec quirk, a doc comment the language or a public API requires, a license header, a build or tool directive, a generated-file marker. The kept comment names that thing.
- Lint and type suppressions are comments too: `eslint-disable`, `@ts-ignore`, `@ts-expect-error`, `//nolint`, `# noqa`, `# type: ignore`. A suppression that hides a correctness or safety problem gets the real fix.
- Constraint comments say `do not remove`, `do not change wording`, or `talk to X before changing`. Offer the cheapest check that enforces it in scope: a type, a runtime check, a test, or a CI lint. Ask first. If approved, add the check and delete the comment. If not, delete the comment and report the constraint as open.
- Scope is the diff or the files you were given. Do not sweep other files.

## Done

You reported the comments you deleted, the ones you kept and why, the checks you added or offered, and any constraint left unenforced.
