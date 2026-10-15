# Docs

Quickstart:

```bash
npx skills add edheltzel/Do-Skills --skill=do-docs
```

```bash
npx skills update do-docs
```

[Source](https://github.com/edheltzel/Do-Skills/tree/master/skills/global/workflow/do-docs)

## What it does

`do-docs` authors or improves one Diataxis page, or writes one short learning when a verified fix's reasoning is not in the code, tests, or docs. A page is one type. Mixing a lesson, a task, a lookup, and a why in the same section weakens all of them.

Write mode picks the type with the compass, then follows that type's template and gates. Improve mode (`/do-docs improve <path>`) classifies an existing markdown file and refines each section with the user before any overwrite. A learning is not a Diataxis page. It exists only when a later reader of the implementation would otherwise repeat the mistake or redo the investigation. The project's AGENTS.md wins if it names a solutions location. Otherwise the note lands in `docs/solutions/`.

## When to reach for it

Type `/do-docs`, or the agent reaches for it automatically when writing a tutorial, getting-started, onboarding, beginner, or learn-by-doing guide, a how-to, a reference, or an explanation, or when a hard-won fix would be lost if only the diff remained.

Improve mode runs only when you ask to improve or review an existing doc. Pass the file path. A README that drifted from the code belongs to [readme-update](./readme-update.md). Commits, issues, and specs belong to [tech-writing](./tech-writing.md).

## One type, then the page

The leading idea is the **compass**: action or cognition, acquisition or application. That choice picks the reference (tutorial, how-to, reference, explanation) and the voice that goes with it. A tutorial is a lesson the teacher walks with the reader; each step ends in something they can see. A how-to assumes a goal. Reference states facts to look up. Explanation discusses why.

Improve is a two-phase loop: analyze and classify, then refine section by section (`yes` / `skip` / `modify`) and write only after every open issue has a terminal choice.

A **learning** is the other case: one note per run, stating the problem, the obvious fix that failed, what worked, and how to tell next time. If the code already carries that, write nothing.

## It's working if

- The draft is one Diataxis type, not a mix.
- A tutorial step says what to do, then what the reader should see.
- Improve does not overwrite the file until every flagged section is applied, skipped, or modified to a close.
- A learning is one short file, or an update to the note that already covers that problem, and only when the diff does not carry the reasoning.

## Where it fits

A writing skill in Workflow, next to [tech-writing](./tech-writing.md) (shared prose posture). A README that drifted from the code belongs to [readme-update](./readme-update.md). Plain-prose cleanup of text that already landed belongs to [humanize](../core/humanize.md). A learning is the note you write when the fix itself does not carry the reason.
