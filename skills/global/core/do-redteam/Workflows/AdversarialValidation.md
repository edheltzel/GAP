# Adversarial validation

**Purpose:** Develop independent alternatives to one brief, compare them, and synthesize one coherent approach. The caller adopts it and does the later work. This workflow does not implement.

**When to use:**
- A defined goal whose alternatives are still too thin to judge
- An architecture or feature choice that needs more than one concrete approach before a pick
- Supplied sketches, or several already-concrete options, that need a fair comparison

Not a code diff (`do-adversarial-review`, if installed). Not one named option that only needs a verdict (Judge, in SKILL.md). Not a routine choice, and not a decision the project already settled.

**Source:** Network Chuck prompting video (2025) - "AI is normally better at critiquing or editing than original writing"

## Develop, then synthesize

The three rounds below are how this runs. These rules win if a round conflicts with them.

1. Write one brief: goal, constraints, settled decisions, and what concrete means here. Every candidate gets that same brief. Unknowns stay in the brief. A candidate may assume something for itself. Do not push that assumption onto the others. Do not hide a correctness bar in a private rubric, and do not change the criteria to favor an entry.
2. Launch at least two independent candidates in fresh contexts, unless the caller already supplied that many usable ones. If they did, skip launch and start at the critic. Do not play several candidates yourself in one context. A sketch is enough when the mechanism and its real tradeoffs can be judged. Do not claim a sketch proves a runtime result.
3. Read every finished candidate. The critic round assesses them against the shared criteria before you pick. A hard-constraint miss cannot be outvoted by preference. Agreement is not proof.
4. Pick the strongest viable base. Take a piece from another candidate only when the result stays one coherent approach. Keep the rejection reasons.
5. If fewer than two usable candidates came back, or the critic never finished, return incomplete and label any lean as provisional. If only the user can supply the deciding preference, return that dependency. Do not invent a winner.

In Empryo (detect it per GAP `global/docs/agents/harness-detection.md`), step 2 may use Empryo's `council` to develop the competing candidates.

Return selected, unresolved, or incomplete. Include the brief, the chosen approach if any, why, what was taken from whom, what was rejected, and evidence still missing.


---

## The Three-Round Protocol

### Round 1: Competing Proposals

Launch 2-3 specialized agents to produce competing solutions from different perspectives:

```
<instructions>
ADVERSARIAL VALIDATION - ROUND 1 (COMPETING PROPOSALS):

You are [PERSONA]. Generate your best solution to this problem.

Your perspective emphasizes:
- [Key concern 1]
- [Key concern 2]
- [Key concern 3]

Produce a complete solution that optimizes for YOUR priorities.
Be specific and detailed - this will be compared against other proposals.
</instructions>

[Problem/Task description]
```

**Suggested Persona Combinations:**

**For Architecture Decisions:**
- Engineer (implementation ease, maintainability)
- Architect (scalability, patterns, long-term)
- Security (attack surface, vulnerabilities)

**For Feature Specs:**
- Product (user value, simplicity)
- Engineer (feasibility, complexity)
- QA (testability, edge cases)

**For Content/Writing:**
- Subject Matter Expert (accuracy, depth)
- Audience Representative (clarity, engagement)
- Editor (structure, flow)

### Round 2: Brutal Critique

A critic agent reads ALL proposals and "brutally critiques" each:

```
<instructions>
ADVERSARIAL VALIDATION - ROUND 2 (BRUTAL CRITIQUE):

You are the Harsh Critic. You have received [N] proposals for [problem].

For EACH proposal:

1. **What they got RIGHT:**
   - Acknowledge genuine strengths (be specific)

2. **What they got WRONG:**
   - Identify flaws, gaps, weaknesses
   - Point out what they conveniently ignored
   - Highlight where their perspective blinds them

3. **The Uncomfortable Truth:**
   - What none of them want to hear?
   - What's the real problem they're all dancing around?

4. **If forced to pick one:**
   - Which has the strongest foundation?
   - Why?

Be harsh but fair. The goal is truth, not destruction.
</instructions>

[Proposals from Round 1]
```

### Round 3: Collaborative Synthesis

The original agents read the critique and collaborate on a final solution:

```
<instructions>
ADVERSARIAL VALIDATION - ROUND 3 (COLLABORATIVE SYNTHESIS):

You are [PERSONA]. You've seen the Critic's brutal assessment.

Working with your fellow agents, produce a SINGLE UNIFIED solution that:
- Addresses the valid criticisms
- Incorporates the best elements from each proposal
- Resolves the tensions between perspectives
- Acknowledges remaining trade-offs honestly

This is not compromise - it's synthesis. The final output should be BETTER
than any individual proposal, not just a blend.
</instructions>

[Original proposals + Critic's assessment]
```

---

## Complete Adversarial Validation Template

The block below is the brief shape. Run each candidate in its own context. Do not execute this whole protocol as one prompt playing every persona.

```
<instructions>
ADVERSARIAL VALIDATION - FULL PROTOCOL:

ROUND 1 - COMPETING PROPOSALS:
Generate 3 distinct solutions to this problem from different perspectives:

Proposal A (Pragmatist): Prioritizes ease of implementation and quick wins
Proposal B (Idealist): Prioritizes best practices and long-term quality
Proposal C (Skeptic): Prioritizes risk reduction and failure prevention

Each proposal should be complete and defensible.

ROUND 2 - BRUTAL CRITIQUE:
As a harsh but fair critic, evaluate all three proposals:
- What each got right
- What each got wrong
- The uncomfortable truth none addressed
- Which has the strongest foundation

ROUND 3 - COLLABORATIVE SYNTHESIS:
Synthesize the best elements into a single superior solution that:
- Addresses the valid criticisms
- Incorporates strengths from each proposal
- Resolves tensions between perspectives
- Honestly acknowledges remaining trade-offs

OUTPUT:
- Brief summary of the three proposals
- Key critique points
- Final synthesized recommendation with clear rationale
</instructions>

[Problem/Task to validate]
```

---

## Integration with Existing RedTeam

**Relationship to 32-Agent Protocol:**
- The 32-agent parallel analysis (parallelAnalysis.md) is for DEPTH - attacking one argument from 32 perspectives
- Adversarial Validation is for SYNTHESIS - producing better output through competition and refinement

**When to use which:**
- Parallel analysis: stress-test an existing argument
- This workflow: the alternatives still need to be built before a fair choice
- Judge (SKILL.md): one named option, a verdict against evidence, no new candidates

**Combining Both:**
1. Use Adversarial Validation to produce initial solution
2. Use 32-Agent Protocol to stress-test the synthesized result
3. Iterate if critical flaws found

---

## Quality Signals

**Good Adversarial Validation:**
- Each proposal genuinely represents its perspective (not strawmen)
- Critic finds real flaws, not just nitpicks
- Synthesis is demonstrably better than any individual proposal
- Trade-offs are honestly acknowledged
- Final output withstands scrutiny

**Bad Signs:**
- Proposals are too similar (not enough diversity)
- Critic is too gentle or too destructive
- "Synthesis" is just one proposal with minor tweaks
- Trade-offs are handwaved away
- Adversarial process feels performative

---

## Example: API Design Decision

**Problem:** "Design authentication flow for our new API"

**Round 1 Proposals:**

**Proposal A (Pragmatist):** JWT with 24-hour expiry, refresh tokens, standard claims. Quick to implement, well-understood pattern.

**Proposal B (Idealist):** OAuth 2.0 with PKCE, short-lived access tokens (15 min), secure token storage, audit logging. Industry best practice.

**Proposal C (Skeptic):** API keys with rate limiting, IP allowlisting, manual rotation. Minimal attack surface, easy to revoke.

**Round 2 Critique:**

- A is fast but JWT secrets are hard to rotate, 24h is too long
- B is gold standard but massive implementation overhead for current team size
- C is secure but doesn't scale to public API use case

Uncomfortable truth: None addressed the question of "what happens when we NEED to revoke all tokens immediately?"

**Round 3 Synthesis:**

Start with A's JWT approach (pragmatic), add B's short-lived tokens (15 min) and audit logging (security), implement C's emergency revocation list (skeptic's concern). Defer full OAuth until public API launch.

---

## Key Principles

1. **Genuine competition** - Proposals must represent real alternatives, not strawmen
2. **Brutal but fair critique** - Find real flaws, acknowledge real strengths
3. **Synthesis > Compromise** - The goal is BETTER, not BLENDED
4. **Honest trade-offs** - Don't pretend tensions are fully resolved
5. **Permission to fail applies** - If no synthesis is clearly superior, say so

---

## When not to use

- The answer is obvious, or the project already settled it
- One named option only needs a verdict (Judge, in SKILL.md)
- A code diff (`do-adversarial-review`, if installed)
- Fewer than two independent approaches can meet the brief, and the evidence says why: return that, do not invent a field

If only the user can supply the deciding preference, return that dependency. Do not invent a winner.

---

**Related Workflows:**
- `parallelAnalysis.md` - For stress-testing existing arguments

**Last Updated:** 2025-11-27
