# Adversarial Validation Pattern — Battle of Bots

<!-- markdownlint-disable MD013 -->

Follow the notification contract in [`../SKILL.md`](../SKILL.md), then print:

```text
Running the **AdversarialValidation** workflow in the **Red Team** skill to validate decisions...
```

## Purpose

Use genuinely competing proposals plus harsh critique to produce a superior synthesis through adversarial refinement.

## When to use

- Feature specifications that need stress-testing.
- Architectural decisions before implementation.
- Code reviews where multiple approaches exist.
- Content that needs to withstand scrutiny.
- Any output where quality matters more than speed.

Source inspiration: Network Chuck's 2025 prompting video and its observation that critique and editing often expose weaknesses that first-pass generation misses.

## Pi execution model

With Pi subagents, use the complete call shape in [`../Integration.md`](../Integration.md):

1. Launch two or three independent proposal agents in one parallel batch using `Agent` with complete self-contained, read-only prompts.
2. Continue clarifying constraints while they run; do not poll or sleep.
3. When completion notifications arrive, collect each result once with `get_subagent_result`.
4. Give every complete proposal to a critic agent, or run the critic inline if another dispatch adds no value.
5. Synthesize in the main agent. Subagents do not share context or collaborate directly; the main agent is the orchestrator.

When subagents are unavailable or the task is small, use the single-prompt fallback below and disclose inline execution.

---

## Three-round protocol

### Round 1: Competing proposals

Use two or three specialized perspectives that optimize for meaningfully different priorities.

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
| --- | --- |
| Architecture | Engineer, Architect, Security/Skeptic |
| Feature specification | Product, Engineer, QA |
| Content/writing | Subject-matter expert, Audience representative, Editor |
| Strategy | Operator, Optimist, Skeptic |

### Round 2: Brutal critique

A critic reads **all** complete proposals and critiques each.

```text
ADVERSARIAL VALIDATION — ROUND 2: BRUTAL CRITIQUE

You are the Harsh Critic. You have received [N] proposals for [problem].

For each proposal:

1. What they got right:
   - Acknowledge genuine strengths with specifics.
2. What they got wrong:
   - Identify flaws, gaps, blind spots, and ignored constraints.
3. The uncomfortable truth:
   - What real problem are all proposals dancing around?
4. If forced to pick one:
   - Which has the strongest foundation, and why?

Be harsh but fair. The goal is truth, not destruction.
```

### Round 3: Collaborative synthesis

The main agent combines the proposals and critique into one superior result. Optionally ask one integrator subagent for an independent synthesis, but retain responsibility for the final judgment.

```text
ADVERSARIAL VALIDATION — ROUND 3: COLLABORATIVE SYNTHESIS

You have all original proposals and the critic's assessment.

Produce a single unified solution that:
- Addresses valid criticisms.
- Incorporates the strongest elements from each proposal.
- Resolves tensions between perspectives where possible.
- Acknowledges remaining trade-offs honestly.

This is not compromise; it is synthesis. The final output must be better
than any individual proposal, not merely a blend.
```

If no synthesis is clearly superior, report the unresolved choice and what evidence would decide it.

---

## Single-prompt fallback

```text
ADVERSARIAL VALIDATION — FULL PROTOCOL

ROUND 1 — COMPETING PROPOSALS:
Generate three distinct solutions from different perspectives:

Proposal A — Pragmatist: implementation ease and quick wins.
Proposal B — Idealist: best practices and long-term quality.
Proposal C — Skeptic: risk reduction and failure prevention.

Each proposal must be complete and defensible.

ROUND 2 — BRUTAL CRITIQUE:
As a harsh but fair critic, evaluate all three:
- What each got right.
- What each got wrong.
- The uncomfortable truth none addressed.
- Which has the strongest foundation.

ROUND 3 — COLLABORATIVE SYNTHESIS:
Synthesize the strongest elements into a superior solution that addresses
valid criticisms, resolves tensions, and names remaining trade-offs.

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
2. Use [`ParallelAnalysis`](ParallelAnalysis.md) to stress-test the synthesized result.
3. Iterate if critical flaws remain.

## Quality signals

Good adversarial validation:

- Each proposal genuinely represents its perspective rather than serving as a strawman.
- The critic finds real flaws and acknowledges real strengths.
- The synthesis is demonstrably better than any individual proposal.
- Remaining trade-offs are explicit.
- The final output withstands scrutiny.

Bad signs:

- Proposals are too similar.
- The critic is too gentle or destructive.
- “Synthesis” is one proposal with minor edits.
- Trade-offs are handwaved away.
- The process feels performative.

## Example: API authentication decision

**Problem:** Design authentication for a new API.

### Round 1 proposals

- **Pragmatist:** JWTs with 24-hour expiry, refresh tokens, and standard claims. Fast and familiar.
- **Idealist:** OAuth 2.0 with PKCE, 15-minute access tokens, secure storage, and audit logging.
- **Skeptic:** API keys with rate limiting, IP allowlists, and manual rotation. Minimal moving parts.

### Round 2 critique

- The pragmatic option is fast, but 24-hour bearer tokens are difficult to revoke safely.
- The idealist option is robust, but its implementation overhead may exceed current needs.
- The skeptic option is controllable, but it does not scale to a public end-user API.
- **Uncomfortable truth:** none initially defines emergency revocation of all active credentials.

### Round 3 synthesis

Start with JWTs, use short-lived access tokens and audit logging, add an emergency revocation mechanism, and defer full OAuth until the public API requires delegated authorization.

## Key principles

1. **Genuine competition** — proposals represent real alternatives, not strawmen.
2. **Brutal but fair critique** — find real flaws and acknowledge real strengths.
3. **Synthesis over compromise** — seek a better result, not a blended one.
4. **Honest trade-offs** — do not pretend all tensions are resolved.
5. **Permission to fail** — if no synthesis is clearly superior, say so.

## When not to use

- Simple decisions with obvious answers.
- Time-critical situations where the process costs more than it can improve.
- Creative work where multiple valid outputs are the goal.
- Problems with strong expert consensus unless new evidence challenges that consensus.

Adversarial Validation is for decisions where quality matters more than speed.
