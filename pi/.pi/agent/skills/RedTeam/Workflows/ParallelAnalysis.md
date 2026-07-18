# Red Team Parallel Analysis Workflow

<!-- markdownlint-disable MD013 -->

Follow the notification contract in [`../SKILL.md`](../SKILL.md), then print:

```text
Running the **ParallelAnalysis** workflow in the **Red Team** skill to red team arguments...
```

## Deliverable

Attack an argument from many diverse perspectives, then produce the two things that matter:

**Output = the strongest steelman of the argument and the strongest surviving counter-argument against it.** Each is an eight-point story with 12–16 words per point. The counter-argument must defeat the steelman, not a weaker version. A proponent should read the steelman and say, “Yes, that is my argument,” then be unable to dismiss the counter-argument as misunderstanding.

Rank real weaknesses by severity. Do not reward nitpicking or contrarianism. Find the fundamental flaw: the assumption that, if it falls, collapses the structure.

## Five-phase protocol

### Phase 1: Decompose

Perform the first-principles procedure in [`../Integration.md`](../Integration.md):

1. Separate fundamental truths from assumed truths.
2. Break the position into atomic, independently attackable claims.
3. Make every claim self-contained, specific, and challengeable.
4. Surface missing links between premises and conclusion.

Use the natural claim count for the position; never split or combine claims merely to hit a quota.

### Phase 2: Dispatch parallel analysis

Use the Pi orchestration contract in [`../Integration.md`](../Integration.md).

- For the full protocol, launch all 32 personas below as independent, read-only `Agent` calls in one assistant turn with `subagent_type: general-purpose` and `run_in_background: true`.
- Give every agent the full position, claim decomposition, persona, and exact response contract.
- Do not prescribe search steps; ask the analytical question and let the specialist judge what evidence it needs.
- Continue useful decomposition or evidence gathering while agents run. Never poll or sleep.
- Retrieve each completed result once after Pi reports completion.
- If 32 calls are disproportionate or constrained, select 8–12 representatives spanning all four families and disclose the reduced run.
- If `Agent` is unavailable, simulate 8–12 representative personas inline and disclose the fallback.

### Phase 3: Synthesize by convergence and severity

1. Cluster independent findings that identify the same underlying issue.
2. Treat convergence as a confidence signal, not a vote that replaces reasoning.
3. Preserve a lone finding when its evidence and consequence are unusually strong.
4. Rank findings as critical, major, or minor.
5. Discard noise, duplicated wording, and objections without consequence.
6. Decide whether the position is fundamentally sound with fixable execution or fundamentally flawed.

### Phase 4: Steelman

Construct the strongest honest version of the original position before rebutting it. Preserve its legitimate concerns, strongest evidence, and best causal logic.

### Phase 5: Challenge constraints and counter-argue

Classify each constraint as:

- **Hard** — unavoidable reality; do not pretend it can be negotiated.
- **Soft** — policy, convention, choice, or preference; challenge it.
- **Assumption** — unvalidated belief; prioritize it.

Then write the strongest counter-argument that defeats the steelman. Finish with remediation paths and a clear verdict.

---

## Persona library

### Engineers — technical and logical rigor

| Agent | Personality | Attack angle |
| --- | --- | --- |
| EN-1 | **The Skeptical Systems Thinker** — thirty years building distributed systems; trusts nothing. | Where does this break at scale? |
| EN-2 | **The Evidence Demander** — accepts no claim without support. | What evidence or numbers prove this? |
| EN-3 | **The Edge Case Hunter** — finds the one-percent case that destroys assumptions. | What happens when X is not true? |
| EN-4 | **The Historical Pattern Matcher** — recognizes recurring failure modes. | Where has this failed before? |
| EN-5 | **The Complexity Realist** — knows simple proposals hide difficult work. | What is harder than it sounds? |
| EN-6 | **The Dependency Tracer** — follows assumptions to their roots. | Which claim depends on a false upstream belief? |
| EN-7 | **The Failure Mode Analyst** — models how things break. | Which failures are catastrophic or unrecoverable? |
| EN-8 | **The Technical Debt Accountant** — calculates hidden and deferred cost. | What is the real long-term price? |

### Architects — structural and systemic issues

| Agent | Personality | Attack angle |
| --- | --- | --- |
| AR-1 | **The Big Picture Thinker** — sees how pieces connect or fail to connect. | What larger system does this ignore? |
| AR-2 | **The Trade-off Illuminator** — treats every gain as a possible loss elsewhere. | What do we lose, and does it matter more? |
| AR-3 | **The Abstraction Questioner** — challenges category mistakes. | Are these actually the same kind of problem? |
| AR-4 | **The Incentive Mapper** — follows money, power, and motivation. | Who benefits from this being believed? |
| AR-5 | **The Second-Order Effects Tracker** — thinks several moves ahead. | What downstream consequence destroys the intended gain? |
| AR-6 | **The Integration Pessimist** — expects interfaces to fail. | What does not compose with existing reality? |
| AR-7 | **The Scalability Skeptic** — distinguishes ten users from ten thousand. | Which success condition stops scaling? |
| AR-8 | **The Reversibility Analyst** — prices lock-in and one-way doors. | What cannot be undone, and why is that dangerous? |

### Pentesters — adversarial thinking against the argument

| Agent | Personality | Attack angle |
| --- | --- | --- |
| PT-1 | **The Red Team Lead** — thinks like an intelligent opponent. | How would an adversary exploit this logic? |
| PT-2 | **The Assumption Breaker** — searches for the weakest link. | Which dependency is false or unsupported? |
| PT-3 | **The Game Theorist** — models rational strategic responses. | How would a smart opponent adapt? |
| PT-4 | **The Social Engineer** — treats human behavior as part of the system. | How will people route around this? |
| PT-5 | **The Precedent Finder** — maps the proposal to prior patterns. | Is this an old failure in new language? |
| PT-6 | **The Defense Evaluator** — tests whether mitigations work. | Which defense fails under realistic pressure? |
| PT-7 | **The Threat Modeler** — maps exposed assumptions systematically. | Which attack surface remains unaddressed? |
| PT-8 | **The Asymmetry Spotter** — finds mismatched costs and incentives. | Where is one side structurally advantaged? |

These personas attack an idea's logic and incentives. They do not perform computer intrusion or vulnerability testing.

### Interns — fresh eyes and unconventional angles

| Agent | Personality | Attack angle |
| --- | --- | --- |
| IN-1 | **The Naive Questioner** — asks why until the premise breaks. | Why assume X in the first place? |
| IN-2 | **The Analogy Finder** — connects patterns across unrelated fields. | Where did an analogous approach fail? |
| IN-3 | **The Contrarian** — tests whether the opposite may be true. | What evidence supports the reverse position? |
| IN-4 | **The Common Sense Checker** — distrusts cleverness that violates intuition. | What obvious reality is being ignored? |
| IN-5 | **The Zeitgeist Reader** — notices actual behavior on the ground. | Why does nobody do this in practice? |
| IN-6 | **The Simplicity Advocate** — applies Occam's razor. | What simpler explanation or intervention fits? |
| IN-7 | **The Edge Lord** — pushes logic to its extreme consequence. | What absurd result follows if this is universally true? |
| IN-8 | **The Devil's Intern** — states the objection everyone avoids. | What is the uncomfortable truth? |

---

## Agent prompt template

Customize this prompt for each persona:

```text
BALANCED ANALYSIS — [AGENT ID]: [PERSONALITY NAME]

You are [PERSONALITY DESCRIPTION].
Your perspective is: “[ATTACK ANGLE]”

POSITION TO ANALYZE:
[Full original position]

DECOMPOSED CLAIMS:
[Claim breakdown]

MISSION:
Provide an independent, balanced analysis of both the real strengths and real
weaknesses from your assigned perspective. Research only as needed. Do not edit files.

RETURN EXACTLY:

[AGENT ID] ANALYSIS

Strongest point for the position: [Claim number]
[Two or three sentences explaining why it is valid or compelling.]
Take seriously because: [One sentence.]

Strongest point against the position: [Claim number]
[Two or three sentences explaining the flaw.]
Problematic because: [One sentence.]

Overall assessment: [One-sentence independent verdict.]

Be intellectually honest. Find real strengths, not token praise.
Find real weaknesses, not nitpicks or strawmen.
```

---

## Output contracts

### Position analyzed

Open with one sentence stating the strongest fair formulation of what was tested.

### Steelman — eight points, 12–16 words each

```text
# STEELMAN

The position, best version: [One sentence.]

1. [Most compelling opening point.]
2. [Strong supporting evidence.]
3. [Supporting precedent or analogy.]
4. [Legitimate concern addressed.]
5. [What weaker critics misunderstand.]
6. [Real risk if ignored.]
7. [Why informed people believe it.]
8. [Strongest reason to take it seriously.]

Validity assessment: [One sentence on the legitimate core concern.]
```

### Convergent weaknesses

Group findings by severity:

- **Critical** — collapses the position or creates unacceptable consequence.
- **Major** — materially weakens the position but may be remediable.
- **Minor** — worth fixing but not decision-changing.

For each finding, state the affected claim, evidence or reasoning, consequence, and convergence count or independent source.

### Counter-argument — eight points, 12–16 words each

```text
# RED TEAM VERDICT

The position: [One-sentence summary.]

1. [Establish the fundamental flaw.]
2. [Develop the core weakness.]
3. [Give precedent or analogy.]
4. [Expose the hidden assumption.]
5. [Show the counterexample or exception.]
6. [Reveal what is conveniently ignored.]
7. [Expose second-order effects.]
8. [Deliver the strongest surviving conclusion.]

Assessment: [One sentence on fundamental soundness after analysis.]
```

### Remediation and verdict

- Name the smallest changes that materially strengthen the position.
- Distinguish mitigations that reduce likelihood from those that reduce consequence.
- End with **proceed**, **revise**, **reject**, or **unresolved**, plus one-sentence rationale.

Each point must be self-contained, use plain language, attack a real weakness, and build toward the strongest objection.

---

## Example: delayed product launch

**Argument:** “Delay launch six months to add more features.”

**First-principles analysis:**

- **Claim type:** normative — we should do X.
- **Hidden assumptions:** more features mean more success; competitors will not act; timing is flexible.
- **Precedent:** products fail from over-engineering and also from shipping without core value.
- **Logical gap:** delay improves outcomes only if feature value exceeds learning and timing costs.

### Steelman

1. Shipping incomplete products can damage trust in ways that take years to repair.
2. Acquisition spend is wasted when users churn because essential functionality is absent.
3. Deliberate releases can outperform rushed competitors on reliability and customer satisfaction.
4. Planned features address the strongest complaints found in current beta research.
5. Critics may mistake competitive parity requirements for optional product gold-plating.
6. Six months' development could cost less than prolonged support for a broken launch.
7. Teams lose confidence when forced to ship work they know is materially incomplete.
8. The greatest risk may be launching something the company must publicly apologize for.

### Counter-argument

1. The plan assumes internal teams know value before real users reveal actual behavior.
2. Every delayed month gives competitors time to capture customers that may never return.
3. Many category leaders learned by shipping narrow first versions, then iterating rapidly.
4. The argument conflates feature quantity with quality, although complexity often destroys value.
5. Six months assumes accurate estimation despite software schedules routinely expanding under uncertainty.
6. Features can follow launch; lost market timing and learning opportunities cannot be restored.
7. Live customer evidence is more valuable than six additional months of internal speculation.
8. The fundamental error treats product development as one bet instead of iterative learning.

The strongest remediation is to define launch-critical outcomes, ship the smallest credible version, and gate later features on observed user evidence.
