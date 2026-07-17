---
name: red-team
description: "Adversarial analysis that stress-tests ideas, strategies, plans, specs, arguments, and decisions by decomposing claims, attacking them from diverse expert perspectives, then steelmanning and counter-arguing with severity-ranked remediation. Use for red team, attack this idea, counterarguments, critique, stress test, devil's advocate, break this, poke holes, find weaknesses, what could go wrong, strongest objection, adversarial validation, or battle of bots. Not for collaborative debate to find the best path, network/system penetration testing, or AI instruction-set auditing."
metadata:
  source-version: "1.1.16"
---

<!-- markdownlint-disable MD013 -->

# Red Team

Attack ideas, strategies, and plans so their weak points surface while they are still cheap to fix. Break the position into atomic claims, challenge it from diverse expert perspectives, synthesize convergent findings, steelman its strongest honest form, then produce the strongest surviving counter-argument and practical remediation.

## Why it exists

Commitment creates confirmation bias: people search for reasons their plan works, while polite or aligned collaborators often skip the hardest objections. Red Team deliberately attacks the position many ways at once so flawed assumptions fail during analysis instead of in production, the market, or a high-stakes decision.

## Pi compatibility

This Pi-managed adaptation preserves the authoritative RedTeam concepts while using Pi's actual conventions:

- Use Pi's `Agent` tool for parallel specialists; never emit Claude-only `Task` syntax.
- Use Pi-relative skill paths and the Pi runtime directory.
- If subagent dispatch is unavailable or disproportionate, simulate a representative persona set inline and disclose that fallback.
- Treat notifications and execution logging as required attempts, but never discard useful analysis because a local optional service is unavailable.

## Customization

Before executing, check `~/.pi/agent/SKILLCUSTOMIZATIONS/red-team/`. If it exists, load and apply relevant `PREFERENCES.md`, configuration, or resources. User customizations override this skill's defaults.

## Notification

Before substantive workflow analysis, send a short notification with Pi's `themis_notify` tool when it is available:

```text
Running the WORKFLOWNAME workflow in the Red Team skill to ACTION
```

If `themis_notify` is unavailable, reports inactive mode, or fails, and the local Echo service exists, make a best-effort POST to Pi's configured Echo endpoint:

```bash
curl -s -X POST http://localhost:8888/notify \
  -H "Content-Type: application/json" \
  -d '{"message":"Running the WORKFLOWNAME workflow in the Red Team skill to ACTION"}' \
  > /dev/null 2>&1 &
```

Also print:

```text
Running the **WorkflowName** workflow in the **Red Team** skill to ACTION...
```

Notification failure must not block the workflow.

## Workflow routing

| Workflow | Trigger | File |
| --- | --- | --- |
| **ParallelAnalysis** | Stress-test existing content, critique an argument, find objections | [`Workflows/ParallelAnalysis.md`](Workflows/ParallelAnalysis.md) |
| **AdversarialValidation** | Design, decide, or create through competing proposals and critique | [`Workflows/AdversarialValidation.md`](Workflows/AdversarialValidation.md) |

## Quick reference

| Workflow | Purpose | Output |
| --- | --- | --- |
| **ParallelAnalysis** | Stress-test existing content | Steelman plus strongest counter-argument, eight points each |
| **AdversarialValidation** | Produce or choose through competition | Synthesized recommendation from competing proposals |

The default ParallelAnalysis deliverable is the strongest steelman and the strongest counter-argument that survives it. Each is an eight-point story with 12–16 words per point. Findings attack real weaknesses, never strawmen, and are ranked by severity.

## Core operating principles

- Attack arguments and decisions, not people or computer systems.
- Decompose first; vague positions produce vague criticism.
- Demand balanced analysis from every perspective: real strengths and real weaknesses.
- Rank findings by convergence, severity, and consequence; volume is not truth.
- Prefer one structural flaw over many performative objections.
- Pair weaknesses with remediation when remediation is possible.
- If no synthesis or counter-argument clearly wins, say so.

## Progressive disclosure

Load only the material needed for the routed workflow:

- [`Philosophy.md`](Philosophy.md) — core philosophy, success criteria, and persona families.
- [`Integration.md`](Integration.md) — Pi subagent orchestration, first-principles procedure, and output contract.
- [`Workflows/ParallelAnalysis.md`](Workflows/ParallelAnalysis.md) — full 32-perspective stress-test protocol and persona roster.
- [`Workflows/AdversarialValidation.md`](Workflows/AdversarialValidation.md) — battle-of-bots competition, critique, and synthesis protocol.

## Examples

### Attack an architecture proposal

```text
User: "Red team this microservices migration plan."
Route: ParallelAnalysis
Output: Steelman, convergent weaknesses, strongest counter-argument, remediation, verdict.
```

### Devil's advocate on a business decision

```text
User: "Poke holes in my plan to raise prices twenty percent."
Route: ParallelAnalysis
Output: The core issue that could collapse the plan, plus ways to reinforce it.
```

### Adversarial validation for a decision

```text
User: "Battle of bots: which approach is better for this feature?"
Route: AdversarialValidation
Output: Competing proposals, harsh critique, and a superior synthesized recommendation.
```

## Gotchas

- Red Team is for attacking **ideas**, not network or system vulnerabilities.
- For neutral collaborative deliberation, use an available council or decision workflow instead.
- Thirty-two perspectives generate volume; discard noise and rank what survives.
- The goal is strengthening, not destruction. Be harsh on ideas and respectful toward people.
- Avoid performative contrarianism, nitpicking, generic objections, and strawmen.

## Execution log

After completing a workflow, attempt to append one JSONL entry to Pi runtime state:

```bash
mkdir -p ~/.pi/agent/memory/skills
echo '{"ts":"'$(date -u +%Y-%m-%dT%H:%M:%SZ)'","skill":"red-team","workflow":"WORKFLOW_USED","input":"8_WORD_SUMMARY","status":"ok|error","duration_s":SECONDS}' >> ~/.pi/agent/memory/skills/execution.jsonl
```

Replace the placeholders with the workflow, a brief input summary, status, and approximate duration. Logging failure must not replace or invalidate the analysis.
