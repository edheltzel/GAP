# Red Team Integration Guide

## Skill integration order

**Use before Red Team:**

- Research/search tools to gather context and precedents.
- First-principles decomposition when the argument rests on implicit assumptions.

**Use during Red Team:**

- Pi subagents for parallel persona analysis when available.
- Inline persona simulation when subagents are unavailable or the request is small.

**Use after Red Team:**

- Extract highest-signal critiques.
- Convert accepted remediation paths into implementation tasks or plan changes.

## First-principles integration

Red Team pairs well with first-principles analysis:

- **Phase 1 enhancement:** deconstruct the argument into constituent parts, fundamental truths, assumed truths, and missing links.
- **Phase 5 enhancement:** classify constraints as hard constraints, soft constraints, or assumptions.
- **Core insight:** the strongest critiques often challenge constraints presented as fixed but actually chosen.

## Pi subagent guidance

If a `subagent`/parallel-agent tool is available:

1. Dispatch persona agents in one parallel batch when safe.
2. Ask each agent for balanced analysis: strongest point for, strongest point against, and overall assessment.
3. Wait for all results before synthesizing.
4. Do not let subagents edit files unless the user explicitly asked for implementation changes.

If no subagent tool is available:

1. Run a compressed in-context version with 8-12 personas.
2. Preserve the same synthesis and steelman/counter-argument structure.
3. State that the analysis used inline persona simulation instead of real parallel dispatch.

## Output format

Default final output:

1. **Position analyzed** — one-sentence summary.
2. **Steelman** — 8 numbered points, ideally 12-16 words each.
3. **Convergent weaknesses** — grouped by severity.
4. **Counter-argument** — 8 numbered points, ideally 12-16 words each.
5. **Remediation paths** — what would make the plan stronger.
6. **Verdict** — proceed, revise, or reject.

## Tone

- Direct, substantive, non-performative.
- Harsh on ideas, respectful toward people.
- Prefer one devastating core critique over many weak objections.
- Avoid nitpicking, strawmanning, generic objections, or vibes-only skepticism.
