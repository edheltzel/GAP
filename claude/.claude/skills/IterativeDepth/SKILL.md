---
name: IterativeDepth
version: 1.2.0
description: "Structured multi-angle exploration running 2-8 passes over the same problem, each through a different scientific lens, to surface hidden testable criteria and edge cases. USE WHEN iterative depth, explore deeper, multi-angle analysis, surface hidden requirements, blind spot check, what am I missing. NOT FOR scope and zoom analysis (use ApertureOscillation)."
effort: high
---

# IterativeDepth

## What It Does

IterativeDepth examines one problem through several structurally different lenses, such as literal, stakeholder, failure, temporal, adversarial, and experiential. Each pass must add a distinct finding or refine an earlier one.

The method is standalone. It does not require an external task tracker, hidden criteria system, or orchestration phase.

## Inputs

- **Problem or request:** The exact subject to explore.
- **Context:** Relevant facts, constraints, known decisions, and existing criteria.
- **Lens preference (optional):** Lenses the caller wants included or excluded.

Read `TheLenses.md` for the lens catalog and `ScientificFoundation.md` for the method's grounding.

## Deliverable

Return a bounded plain Markdown criteria set with:

- stable criterion IDs;
- status for every criterion;
- a binary-testable statement;
- evidence expectations;
- source lenses;
- refined criteria with their previous wording and reason for change;
- anti-criteria describing outcomes that must not occur;
- the most useful cross-angle insight;
- assumptions and unresolved questions.

Use 2 to 8 lenses and return no more than 12 primary criteria. Stop early when another pass adds no distinct criterion or refinement.

## Workflow Routing

| Workflow | Trigger                                               | File                   |
| -------- | ----------------------------------------------------- | ---------------------- |
| Explore  | "iterative depth", "explore deeper", "multi-angle"    | `Workflows/Explore.md` |

## Quality Rules

- Criteria are plain testable statements, not commands to another system.
- Keep IDs stable when refining a criterion.
- Prefer observable evidence over vague judgments.
- Deduplicate semantic repeats.
- Record anti-criteria separately from desired outcomes.
- Never invent missing context. Surface it as an unresolved question.
- Optional retained user material may be read only when the user identifies it. Never modify it.
