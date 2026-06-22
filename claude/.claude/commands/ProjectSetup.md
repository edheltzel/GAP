---
name: atlas:ProjectSetup
description: Bootstrap a GitHub Projects v2 board for the current repo — create the board, Status/Phase/Priority fields, views, and capture field IDs. Wraps the github-projects skill's ProjectSetup recipe.
argument-hint: [optional — project title or owner/repo overrides]
---

Use the **github-projects** skill and follow its `references/ProjectSetup.md` recipe to bootstrap a GitHub Projects v2 board for the current repository.

$ARGUMENTS

Follow the recipe exactly: create the board, add the Status/Phase/Priority fields, create the working views, and capture the field/option IDs into a project-local notes block. Use bun for any tooling — no CLAUDE.md generator, no Husky. See `references/Hierarchy.md` for the issue model and `references/Labels.md` for the label policy (use the repo's existing labels; Phase/Priority are board fields, not labels).
