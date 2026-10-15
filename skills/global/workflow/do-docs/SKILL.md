---
name: do-docs
description: "Author or improve one Diataxis page, or write one short learning when a verified fix's reasoning is not in the code, tests, or docs. Use when writing a tutorial, how-to, reference, or explanation, when asked to improve an existing doc, or when a hard-won fix would be lost if only the diff remained. NOT FOR a README that drifted from the code (`do-readme-update`, if installed) or commits, issues, and specs (`do-tech-writing`, if installed)."
---

# Docs

Three modes. Load only the references the mode needs. Learning loads none of them.

## Write

Author one page of one Diataxis type. Do not mix types.

1. Pick the type with [diataxis-compass](references/diataxis-compass.md).
2. Apply [docs-style](references/docs-style.md).
3. Load the matching type reference and follow it, including its gates and checklist:
   - Tutorial: [tutorial-docs](references/tutorial-docs.md). Worked example: [example-weather-api](references/example-weather-api.md).
   - How-to: [howto-docs](references/howto-docs.md)
   - Reference: [reference-docs](references/reference-docs.md)
   - Explanation: [explanation-docs](references/explanation-docs.md)

Reach for write when authoring docs, including a tutorial, getting-started, onboarding, beginner, or learn-by-doing guide, a how-to, a reference, or an explanation.

## Improve

Runs only when the user asks to improve or review an existing doc. Invoke as `/do-docs improve <path>`.

Load [improve](references/improve.md) and follow its two-phase analyze/refine workflow and gates. Do not start improve on your own.


## Learning

Write one short learning when a verified fix's reasoning is not carried by the final code, tests, or existing docs. If this note disappeared, would a later reader of the implementation repeat the mistake or redo the investigation? If no, write nothing and say why. Effort and diff size do not qualify. An explicit ask still has to pass that test.

One learning per run. If a note already covers this problem, update it. Do not add a second.

Path: the solutions location the project's AGENTS.md names, if it names one. Otherwise `docs/solutions/YYYY-MM-DD-<topic>.md`.

State the problem, the obvious fix that failed, what worked, and how to tell next time. Ground every claim in the current tree. Do not add a concepts glossary, and do not audit older notes.

## Related skills

- A README that drifted from the code: `do-readme-update`, if installed.
- Commits, issues, PRDs, and specs: `do-tech-writing`, if installed.
