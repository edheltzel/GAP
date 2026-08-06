# Oscillate Workflow: Aperture Oscillation

## Purpose

Answer one problem through a tactical lens and a strategic lens, then return a self-contained synthesis of agreements, tensions, and recommended scope.

## Invocation

The workflow may be requested directly by a user or called by another skill that needs a scope-coherence analysis. The caller must provide the full input packet and receives the complete Markdown result.

## Required Inputs

- **Problem or decision:** The exact question held constant across both passes.
- **Context:** Relevant facts, constraints, existing decisions, and boundaries.
- **Tactical lens:** Immediate success, component logic, local constraints, and near-term risks.
- **Strategic lens:** System success, long-term consequences, adjacent dependencies, and broader risks.

If the problem or either lens is missing, stop and request it. If the two lenses are effectively identical, explain why oscillation would add no value and suggest a multi-angle exploration instead.

## Execution

### Pass 1: Tactical

Answer the problem with the tactical lens primary. Cover internal logic, practical interfaces, immediate tradeoffs, implementation constraints, and what good looks like locally.

### Pass 2: Strategic

Answer the same problem with the strategic lens primary. Cover system fit, long-term consequences, dependencies, organizational or product constraints, and what good looks like globally.

### Pass 3: Synthesis

Compare the two passes point by point:

1. Record agreements.
2. Record every material divergence.
3. Explain the consequence of ignoring each side.
4. Recommend a resolution or explicit tradeoff.
5. State the appropriate scope and coherence verdict.

Do not manufacture divergence. When the passes align, explain the shared reasoning and return `ALIGNED`.

## Output Contract

Return plain Markdown in this shape:

```markdown
# Aperture Oscillation

## Inputs
- Problem or decision: ...
- Context: ...
- Tactical lens: ...
- Strategic lens: ...

## Tactical analysis
...

## Strategic analysis
...

## Agreements
- ...

## Divergences and resolutions
### D-01: ...
- Tactical position: ...
- Strategic position: ...
- Consequence: ...
- Recommendation: ...

## Scope recommendation
...

## Coherence
- Verdict: ALIGNED | TENSIONED | DIVERGENT
- Reason: ...

## Assumptions and unresolved questions
- ...
```

The result must stand on its own. It must not require hidden project state, external criteria systems, or a later orchestration phase to be useful.
