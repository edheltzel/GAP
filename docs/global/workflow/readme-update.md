# README Update

Quickstart:

```bash
npx skills add edheltzel/Do-Skills --skill=do-readme-update
```

```bash
npx skills update do-readme-update
```

[Source](https://github.com/edheltzel/Do-Skills/tree/master/skills/global/workflow/do-readme-update)

## What it does

`do-readme-update` brings any repository's README back in line with the code:
install steps, commands, flags, config, structure, and links. It reads the
truth from manifests, scripts, `--help` output, and the git history since the
README last changed, then fixes only what drifted. The defining constraint is
that it never hand-edits a generated region: if the repo has a README
generator, it runs that instead.

## When to reach for it

Type `/do-readme-update`, or the agent reaches for it automatically when a
README is stale or a change adds, removes, or renames something the README
describes.

Reach for it to keep an existing README accurate. To write a new doc page from
scratch, use [docs](./docs.md); for tone and terseness, use
[tech-writing](./tech-writing.md).

## Drift, not rewrite

The skill compares each README claim against the repo and sorts it as wrong,
missing, or dead. It keeps the author's headings, order, and voice, copies
commands verbatim from the manifest, and runs any changed command that is safe
to run locally before calling the README done.

## It's working if

- Every command in the README exists and runs as written.
- Every relative link resolves.
- Generated regions changed only through their generator.

## Where it fits

Periodic maintenance you reach for after shipping changes. It sits in Workflow
next to [commit](./commit.md): update the README, then commit both together. A new doc page beyond the README belongs to [docs](./docs.md).
