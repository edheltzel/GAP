# Red Team Integration Guide

<!-- markdownlint-disable MD013 -->

## Integration order

### Before Red Team

- Gather enough context and evidence with the available research, web, repository, or memory tools.
- Identify precedents and counterexamples instead of relying on generic skepticism.
- Decompose arguments whose conclusions rest on implicit assumptions.

### During Red Team

- Use Pi `Agent` subagents for independent persona analysis when parallelism adds value.
- Use inline persona simulation when subagents are unavailable or the request is small.
- Keep every specialist read-only unless the user explicitly requests implementation.

### After Red Team

- Extract the highest-signal critiques.
- Turn accepted remediation into plan changes, implementation tasks, or decision conditions.
- Preserve unresolved trade-offs instead of hiding them in a false consensus.

## First-principles procedure

The skill does not depend on a Claude-only FirstPrinciples resource. Perform this procedure directly; if a compatible first-principles skill is discovered in the current Pi session, it may assist but is not required.

### Deconstruct before attack

Separate the position into:

1. Fundamental truths supported by evidence or unavoidable reality.
2. Assumed truths that still need validation.
3. Atomic claims connecting premises to conclusion.
4. Missing links where the conclusion does not follow.

### Challenge before the counter-argument

Classify each stated constraint:

- **Hard** — physics, law, or unavoidable reality; do not pretend it is optional.
- **Soft** — policy, convention, budget choice, or reversible preference; challenge it.
- **Assumption** — unvalidated belief presented as a constraint; prioritize it.

The strongest critiques often expose a supposedly hard constraint as soft or assumed.

## Pi subagent orchestration

When Pi's `Agent` tool is available:

1. Prepare self-contained, research-only prompts containing the full position, claim decomposition, persona, and exact output contract.
2. For genuine parallelism, issue independent `Agent` calls together in one assistant turn with this complete shape:

   ```text
   Agent:
     description: "Attack scale assumptions"
     prompt: "<full self-contained research-only brief>"
     subagent_type: "general-purpose"
     run_in_background: true
   ```

   `description` must be a short 3–5-word summary; `prompt` must contain all context the fresh agent needs.
3. Continue independent decomposition or evidence gathering while agents run; do not poll or sleep.
4. As completion notifications arrive, retrieve each result once with the returned identifier:

   ```text
   get_subagent_result:
     agent_id: "<completed agent ID>"
     wait: false
   ```

5. Synthesize all results yourself. Agent summaries are evidence, not the final verdict.

For the full ParallelAnalysis protocol, use the 32-persona roster when stakes and quota justify it. For lower-stakes or constrained work, choose a balanced 8–12 persona cross-section across all four families and disclose the reduced dispatch. Do not silently present inline or reduced analysis as a full 32-agent run.

If subagents are unavailable:

1. Run a compressed in-context pass with 8–12 distinct personas across all four families.
2. Preserve the same decomposition, convergence, steelman, counter-argument, remediation, and verdict structure.
3. State that the workflow used inline persona simulation.

## Default output contract

1. **Position analyzed** — one-sentence summary.
2. **Steelman** — eight numbered points, 12–16 words each.
3. **Convergent weaknesses** — grouped by critical, major, and minor severity.
4. **Counter-argument** — eight numbered points, 12–16 words each.
5. **Remediation paths** — changes that materially strengthen the position.
6. **Verdict** — proceed, revise, reject, or unresolved.

Make the first-principles decomposition and convergence evidence visible enough that a reader can audit how the verdict follows.

## Tone

- Direct, substantive, and non-performative.
- Harsh on ideas and respectful toward people.
- Prefer one devastating structural critique over many weak objections.
- Avoid nitpicking, strawmanning, generic objections, and vibes-only skepticism.
