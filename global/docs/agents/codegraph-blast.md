# Codegraph + blast radius

Before non-trivial edits, know what you might break.

## Rules

- When a project has a codegraph index, use it before writing or editing. Do not guess structure, callers, or conventions it can answer.
- Use it for structural questions: where a symbol is defined, what calls it, what it calls, and what a change would break. Use grep or read for literal text.
- Do not trust the index for unfetched remote state. If the checkout is behind origin, ground truth is the remote tree, not the local index.
- Map callers, shared types, and config surfaces the change touches.
- If you skip a blast-radius pass, say why in one line (truly local, docs-only, or generated).

## Done

You listed the risky neighbors (or explicitly waived) before editing.
