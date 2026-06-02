---
name: red-team
description: Military-grade adversarial analysis for stress-testing ideas, strategies, plans, specs, and arguments. Use when the user says red team, attack this idea, critique, stress test, devil's advocate, poke holes, find weaknesses, what could go wrong, strongest objection, adversarial validation, or battle of bots. Targets arguments and decisions, not network/system vulnerability testing or AI instruction-set auditing.
---

# Red Team

Adversarial analysis for strengthening ideas. Break arguments into atomic claims, attack them from multiple expert perspectives, synthesize convergent weaknesses, steelman the strongest version, then produce a constructive counter-argument with remediation paths.

## Pi conversion notes

This skill was converted from Claude skill `~/.claude/skills/RedTeam` for Pi.

- Use Pi paths and tools.
- Prefer Pi subagents when available; if not, simulate the personas inline and state the fallback.
- Do not require Claude Code's `Task` tool.
- Notifications are best-effort only; never fail the workflow if the local notifier is absent.

## Customization

Before executing, check for user customizations in this order:

1. `~/.pi/agent/SKILLCUSTOMIZATIONS/red-team/`
2. `~/.claude/PAI/USER/SKILLCUSTOMIZATIONS/RedTeam/` legacy source path

If either directory exists, load and apply relevant `PREFERENCES.md`, configuration, or resources. Pi customizations override legacy Claude customizations.

## Best-effort notification

When a workflow is invoked, you may send a local notification if the notifier exists:

```bash
curl -s -X POST http://localhost:31337/notify \
  -H "Content-Type: application/json" \
  -d '{"message": "Running the WORKFLOWNAME workflow in the Red Team skill to ACTION"}' \
  > /dev/null 2>&1 &
```

Also print:

```text
Running the **WorkflowName** workflow in the **Red Team** skill to ACTION...
```

## Workflow routing

| Trigger | Workflow |
|---|---|
| Stress-test existing content, critique an argument, find objections | `Workflows/ParallelAnalysis.md` |
| Choose/design/create through competing proposals and critique | `Workflows/AdversarialValidation.md` |

## Quick reference

| Workflow | Purpose | Output |
|---|---|---|
| **ParallelAnalysis** | Stress-test existing content | Steelman + counter-argument, 8 points each |
| **AdversarialValidation** | Produce or choose via competition | Synthesized recommendation from competing proposals |

## Five-phase protocol for ParallelAnalysis

1. **Decomposition** — break into 24 atomic claims.
2. **Parallel analysis** — attack strengths and weaknesses from multiple personas.
3. **Synthesis** — identify convergent insights.
4. **Steelman** — strongest version of the original argument.
5. **Counter-argument** — strongest rebuttal that defeats the steelman.

## Context files

- `Philosophy.md` — core principles, success criteria, agent types.
- `Integration.md` — skill integration, output format, related skills.
- `Workflows/ParallelAnalysis.md` — full stress-test protocol.
- `Workflows/AdversarialValidation.md` — battle-of-bots protocol.

## Gotchas

- Red Team attacks **ideas**, not systems. Do not use for network/system vulnerability testing.
- Volume is not truth. Rank findings by severity and discard noise.
- The goal is strengthening, not destruction. Pair weaknesses with remediation paths.
- Avoid performative contrarianism, nitpicking, and strawmanning.

## Execution log

After completing a workflow, append one JSONL entry when practical:

```bash
mkdir -p ~/.pi/agent/memory/skills
echo '{"ts":"'$(date -u +%Y-%m-%dT%H:%M:%SZ)'","skill":"red-team","workflow":"WORKFLOW_USED","input":"8_WORD_SUMMARY","status":"ok|error","duration_s":SECONDS}' >> ~/.pi/agent/memory/skills/execution.jsonl
```
