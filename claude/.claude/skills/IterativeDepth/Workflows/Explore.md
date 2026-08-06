# Explore Workflow: Iterative Depth

## Purpose

Run 2 to 8 structured exploration passes over one problem and return a bounded set of plain Markdown criteria that another person or workflow can evaluate directly.

## Inputs

- **Problem or request:** The original question or desired outcome.
- **Context:** Available facts, constraints, code or project state, and prior work.
- **Existing criteria (optional):** Criteria that may be confirmed or refined.
- **Lens preference (optional):** Requested inclusions or exclusions from `TheLenses.md`.

If the problem is missing, stop and request it. Do not infer a hidden objective from unrelated context.

## Execution

1. Read `TheLenses.md`.
2. Select 2 to 8 lenses that fit the problem.
3. Apply each lens to the same problem.
4. Carry prior findings into later passes so a lens can confirm, refine, or challenge them.
5. Stop when a pass adds no distinct criterion, refinement, anti-criterion, or unresolved question.
6. Deduplicate findings without losing stronger evidence expectations.
7. Keep the final set to a maximum of 12 criteria. Prioritize criteria supported by multiple lenses or tied to severe failure modes.

## Criterion Rules

Each primary criterion must have:

- a stable ID in sequence, starting with `C-01`;
- a status of `NEW`, `REFINED`, `CONFIRMED`, or `OPEN`;
- one binary-testable statement;
- an evidence expectation naming what would prove it;
- the lens or lenses that surfaced it.

Refinements keep the original ID. Anti-criteria use stable IDs starting with `A-01` and state an observable outcome that must not occur.

## Output Contract

Return plain Markdown in this shape:

```markdown
# Iterative Depth Result

## Scope
- Problem: ...
- Context used: ...
- Lenses used: ...
- Stop reason: ...

## Criteria
| ID | Status | Criterion | Evidence expectation | Source lenses |
|----|--------|-----------|----------------------|---------------|
| C-01 | NEW | ... | ... | ... |

## Refined criteria
### C-02
- Previous: ...
- Refined: ...
- Why: ...
- Evidence expectation: ...

## Anti-criteria
| ID | Status | Must not occur | Evidence expectation | Source lenses |
|----|--------|----------------|----------------------|---------------|
| A-01 | NEW | ... | ... | ... |

## Cross-angle insight
...

## Assumptions and unresolved questions
- ...
```

If no criteria are refined, keep the `Refined criteria` heading and state `None`. If no anti-criteria are found, keep the `Anti-criteria` heading and state `None`.

The output is complete when a caller can evaluate every item without access to hidden state or external machinery.
