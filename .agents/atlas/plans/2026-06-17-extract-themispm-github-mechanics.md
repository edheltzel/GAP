# Extract atlas-themisPM GitHub Mechanics into the `github-projects` Skill

**Date:** 2026-06-17
**Status:** Implemented — recipes added, PR open
**Owner:** Atlas
**Target:** `skills/github-projects/` (repo-backed skill) — **only.** Themis.md is explicitly out of scope.

---

## Commands folded in (at a glance)

The four template commands Ed flagged became PascalCase recipes inside the skill. Simple names; the SKILL.md description carries the detail.

| Template command | Recipe (PascalCase) | What it carries |
|---|---|---|
| `execute.md` | `references/Execute.md` | board → In Progress, pre-flight (bun), 4-tier deviation rules, AC/close updates, inline-verify |
| `continue.md` | `references/Continue.md` | pick next task from board state (status → priority → deps) |
| `status.md` | `references/Status.md` | board progress report (phase progress, counts, recently-closed, up-next) |
| `init-project.md` | `references/ProjectSetup.md` (+ `references/Labels.md`) | board + Status/Phase/Priority fields + views + field-ID capture; label policy |

Plus two supporting recipes not tied to a single command: `references/Hierarchy.md` (Epic→Feature→Task model + sub-issue plumbing) and `references/Labels.md` (label policy).

---

## Problem

`atlas-themisPM` (clone of `nicholasmartin/claude-workflow-template`) is a complete, opinionated GitHub-driven PM workflow, but adopting it whole would fork PAI's context system: it ships its own PRD format, its own `CLAUDE.md` generator, an npm-based Husky setup, and a `.agents/plans/` path — all of which duplicate or contradict existing PAI conventions (ISA as system of record, global `CLAUDE.md`, bun mandate, `.agents/atlas/plans/` rule).

Meanwhile the `github-projects` skill was a flat `gh`/GraphQL reference (SKILL.md + `references/fields.md` + `references/items.md`) with no model for issue hierarchy, label policy, board+views, sub-issue linking, plan execution, next-work selection, or status reporting. The template's valuable, tedious-to-rebuild layer is exactly those GitHub mechanics. Extract them; leave the rest.

## Namespace convention

- **PascalCase** for both file name and reference name (`Execute.md`, not `execute-plan.md`).
- Simple, single-concept names; the SKILL.md References index supplies the longer description.
- Existing `fields.md` / `items.md` left lowercase (pre-date the convention) — rename later only if desired.

## Out of Scope (NOT extracted)

- **Themis.md** — not modified. Themis manages/delegates/validates/documents; never plans or codes.
- **`/create-prd`** — ISA is PAI's system of record.
- **`/create-rules` CLAUDE.md generator** — clashes with Ed's global `CLAUDE.md`.
- **Husky + lint-staged + Prettier (npm)** — bun-only mandate.
- **`.agents/plans/` path** — PAI uses `.agents/atlas/plans/`.
- **`/prime`** — superseded by `Recall:scout`; harvest its tech-stack/conventions/config-read/`!`-mechanic into scout via a separate issue.
- **The slash-command runner framework** — mechanics extracted as recipes, not the commands as a framework. Algorithm + Themis + Recall are the engine.

## Labels decision (resolved)

Per Ed: **keep the repo's existing labels and adjust to ours.** No template `type:`/`priority:`/`source:` taxonomy. Triage roles per `docs/agents/triage-labels.md`; Phase/Priority are board fields, not labels. `Labels.md` documents this.

## What shipped

1. `references/Hierarchy.md` — Epic→Feature→Task + issue-creation checklist + GraphQL (`addProjectV2ItemById`, `updateProjectV2ItemFieldValue`, `addSubIssue`).
2. `references/ProjectSetup.md` — board bootstrap (fields, views, automations, field-ID capture), bun, no CLAUDE-gen/Husky.
3. `references/Labels.md` — use repo taxonomy; Phase/Priority as fields; triage-roles pointer.
4. `references/Execute.md` — run an existing plan/ISA against the board; pre-flight (bun), 4-tier deviation rules, inline-verify, AC/close updates; ISA-aware, never derives a plan.
5. `references/Continue.md` — board prioritization (status → priority → issue# → deps); session-resume left to Recall.
6. `references/Status.md` — board progress report.
7. `skills/github-projects/SKILL.md` — extended description/triggers + References index (gh primitives + PM recipes).

## Verification

- All six recipes present; `Read` confirms content.
- `grep` confirms no `npm`/`husky` strings in Execute.md / ProjectSetup.md.
- SKILL.md References index lists all eight references.
- Shipped via worktree → feature branch → PR into `master`. Themis.md untouched.
