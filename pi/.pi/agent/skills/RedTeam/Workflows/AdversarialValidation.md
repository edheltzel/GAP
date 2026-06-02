# Adversarial Validation Pattern — Battle of Bots

Running the **AdversarialValidation** workflow in the **Red Team** skill to validate decisions...

## Purpose

Use competing proposals plus harsh critique to produce a stronger synthesized decision, plan, spec, or recommendation.

## When to use

- Feature specifications that need stress-testing.
- Architectural decisions before implementation.
- Code reviews where multiple approaches exist.
- Content that needs to withstand scrutiny.
- Any output where quality matters more than speed.

Do not use for simple decisions with obvious answers or time-critical situations.

---

## Three-round protocol

### Round 1: Competing proposals

Launch 2-3 specialized perspectives. Use Pi subagents if available; otherwise generate the perspectives inline.

```text
ADVERSARIAL VALIDATION — ROUND 1: COMPETING PROPOSALS

You are [PERSONA]. Generate your best solution to this problem.

Your perspective emphasizes:
- [Key concern 1]
- [Key concern 2]
- [Key concern 3]

Produce a complete solution that optimizes for your priorities.
Be specific and detailed; this will be compared against other proposals.

Problem:
[Problem/task description]
```

Suggested persona combinations:

| Domain | Personas |
|---|---|
| Architecture | Engineer, Architect, Security/Skeptic |
| Feature spec | Product, Engineer, QA |
| Content/writing | Subject-matter expert, Audience representative, Editor |
| Strategy | Operator, Optimist, Skeptic |

### Round 2: Brutal critique

A critic reads all proposals and critiques each.

```text
ADVERSARIAL VALIDATION — ROUND 2: BRUTAL CRITIQUE

You are the Harsh Critic. You have received [N] proposals for [problem].

For each proposal:

1. What they got right:
   - Acknowledge genuine strengths.
2. What they got wrong:
   - Identify flaws, gaps, blind spots, and ignored constraints.
3. The uncomfortable truth:
   - What is the real problem everyone is dancing around?
4. If forced to pick one:
   - Which has the strongest foundation, and why?

Be harsh but fair. The goal is truth, not destruction.
```

### Round 3: Collaborative synthesis

The original perspectives read the critique and produce one superior solution.

```text
ADVERSARIAL VALIDATION — ROUND 3: COLLABORATIVE SYNTHESIS

You are [PERSONA]. You have seen the critic's assessment.

Produce a single unified solution that:
- Addresses valid criticisms.
- Incorporates the best elements from each proposal.
- Resolves tensions between perspectives where possible.
- Acknowledges remaining trade-offs honestly.

This is not compromise; it is synthesis. The final output should be better
than any individual proposal, not merely blended.
```

---

## Single-prompt fallback

When subagents are unavailable or the task is small, run this compressed protocol inline:

```text
ADVERSARIAL VALIDATION — FULL PROTOCOL

ROUND 1 — COMPETING PROPOSALS:
Generate 3 distinct solutions from different perspectives:

Proposal A — Pragmatist: implementation ease and quick wins.
Proposal B — Idealist: best practices and long-term quality.
Proposal C — Skeptic: risk reduction and failure prevention.

ROUND 2 — BRUTAL CRITIQUE:
As a harsh but fair critic, evaluate all three:
- What each got right.
- What each got wrong.
- The uncomfortable truth none addressed.
- Which has the strongest foundation.

ROUND 3 — COLLABORATIVE SYNTHESIS:
Synthesize the best elements into a superior solution that addresses valid criticisms,
incorporates strengths, resolves tensions, and names remaining trade-offs.

OUTPUT:
- Brief summary of the three proposals.
- Key critique points.
- Final synthesized recommendation with rationale.
```

---

## Relationship to ParallelAnalysis

- **ParallelAnalysis** is for depth: attack one existing argument from many angles.
- **AdversarialValidation** is for synthesis: produce or choose a better output through competition.

Combine them when quality stakes are high:

1. Use AdversarialValidation to produce the initial recommendation.
2. Use ParallelAnalysis to stress-test the synthesized result.
3. Iterate if critical flaws remain.

## Quality signals

Good adversarial validation:

- Proposals genuinely differ.
- Critique finds real flaws and real strengths.
- Synthesis is demonstrably better than any individual proposal.
- Trade-offs are honestly acknowledged.

Bad signs:

- Proposals are too similar.
- Critic is too gentle or too destructive.
- Synthesis is just one proposal with minor edits.
- Trade-offs are handwaved away.
