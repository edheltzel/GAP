---
name: atlas:ProjectSetup
description: Bootstrap a GitHub Projects v2 board for the current repo — pick an archetype + overlays, create and link the board, converge fields/options/labels with the reconcile engine, then finish the manual views/automations checklist. Wraps the pm-tools skill's ProjectSetup recipe.
argument-hint: [optional — project title or owner/repo overrides]
---

Use the **pm-tools** skill and follow its `references/ProjectSetup.md` recipe to bootstrap a GitHub Projects v2 board for the current repository.

$ARGUMENTS

Follow the recipe exactly: pick the archetype + overlays, create and link the board, run `Tools/Reconcile.ts` in plan mode first and apply only after reviewing the diff, then complete the manual views/automations checklist. Use bun for any tooling — no CLAUDE.md generator, no Husky. See `references/Hierarchy.md` for the issue model and `references/Labels.md` for the label policy (Status/Phase/Priority are board fields, not labels).
