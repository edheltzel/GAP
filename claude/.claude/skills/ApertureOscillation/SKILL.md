---
name: ApertureOscillation
version: 1.1.0
description: "Three-pass scope analysis that holds a problem constant while applying tactical and strategic lenses, then synthesizes the tension between them. USE WHEN aperture oscillation, oscillate scope, zoom in and out, tactical vs strategic, scope framing, design tension, system coherence check, local vs global design, wrong scope, scope negotiation. NOT FOR lens rotation across many analytical angles (use IterativeDepth)."
effort: medium
---

# ApertureOscillation

## What It Does

ApertureOscillation examines one problem at two scope levels, then reconciles the results:

1. **Tactical pass:** Treat the immediate component, decision, or action as primary.
2. **Strategic pass:** Treat the wider system, goal, and consequences as primary.
3. **Synthesis:** Identify agreement, tension, and required tradeoffs.

The problem stays constant. Only the scope changes. This exposes local designs that conflict with system needs and broad strategies that ignore practical constraints.

## Inputs

The caller must provide or confirm:

- **Problem or decision:** The exact question to analyze.
- **Context:** Relevant facts, constraints, and known decisions.
- **Tactical lens:** What success means at the immediate or component level.
- **Strategic lens:** What success means at the system or long-term level.

Do not invent missing inputs. Ask for the missing distinction when the tactical and strategic lenses are not meaningfully different.

## Deliverable

Return a self-contained Markdown synthesis containing:

- the confirmed inputs;
- a tactical analysis;
- a strategic analysis;
- agreements and divergences;
- a resolution recommendation for each divergence;
- a scope recommendation;
- a coherence verdict of `ALIGNED`, `TENSIONED`, or `DIVERGENT`;
- assumptions and unresolved questions.

A result with no divergence is valid. State why the two views align instead of manufacturing tension.

## Workflow Routing

| Workflow  | Trigger                                                                           | File                     |
| --------- | --------------------------------------------------------------------------------- | ------------------------ |
| Oscillate | "aperture oscillation", "oscillate scope", "zoom in/out", "tactical vs strategic" | `Workflows/Oscillate.md` |

## How It Differs from IterativeDepth

| Dimension   | IterativeDepth                    | ApertureOscillation                            |
| ----------- | --------------------------------- | ---------------------------------------------- |
| What varies | Analytical angle                  | Scope level                                    |
| Pass count  | 2 to 8                            | 3 fixed passes                                 |
| Input       | One problem plus context          | One problem plus tactical and strategic lenses |
| Output      | Testable criteria and blind spots | Scope tensions and coherence recommendation    |

Use IterativeDepth to explore a problem from multiple angles. Use ApertureOscillation to determine whether a local approach fits its wider context.

## Gotchas

- Keep the question identical in both passes.
- Make each lens explicit rather than silently inferring it.
- Do not make the strategic pass a vague restatement of the tactical pass.
- Do not claim access to integrations, project state, or retained user data that the caller did not provide.
- Optional retained user material may be read only when the user supplies or identifies it. Never modify it.
