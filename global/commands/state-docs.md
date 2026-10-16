---
description: Write or update docs/state system maps, validate, and open them in the docs browser
argument-hint: "[project path] [focus or 'full sweep']"
---
Document a project in `docs/state` for the local docs browser.

Arguments: $ARGUMENTS
- First argument: absolute project path. If missing, use the git repository root of the current directory.
- Remaining text: what to cover. If empty, update pages affected by recent changes; if no `docs/state/README.md` exists, map the whole project.

Nessa docs root: use `$NESSA_DOCS` if set, else `/Users/ed/Developer/Tools/nessa-docs`.

Steps:
1. Read the nessa docs root's `skills/nessa-docs/SKILL.md` and `references/format.md` beside it, and follow them exactly. Do not copy the skill into the project.
2. Read the project's AGENTS.md chain before reading source. Ground every page in source you read; mark fixtures, proposals and anything unverified via `status`. No em dashes.
3. For a monorepo, keep one system root at `docs/state/README.md` and one service section per app under `docs/state/services/<app>/`. Fan out one subagent per app when there are 3 or more. Edit existing owning pages instead of duplicating them.
4. Validate until clean:
   `cd` to the nessa docs root and run `npm run --silent validate:docs -- --project <project path>`
5. Restart the browser for that project: stop any running `docs-browser` service, then start one named `docs-browser` with `NESSA_DOCS_PROJECT=<project path> PORT=4312 npm start` in the nessa docs root.
6. Report pages written or changed, validation result, what was not visually checked, and the URL http://127.0.0.1:4312. Do not commit.
