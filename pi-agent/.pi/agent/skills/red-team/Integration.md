# Red Team Integration Guide

## Skill Integration Order

**Use BEFORE RedTeam:**
- `brainstorming` skill - Gather context, explore alternatives
- `web-search` skill - Find precedents and counterexamples

**Use DURING RedTeam:**
- `think` tool with `first_principles` mode - Decomposition methodology (Phase 1 &5)
- `subagent` tool - Parallel agent deployment (Phase 2)

**Use AFTER RedTeam:**
- `web-search` skill - Find additional counterexamples
- `notify` tool - Alert user when analysis complete

## Think Tool Integration

RedTeam deeply integrates with the `think` tool's `first_principles` mode:

- **Phase1 Enhancement:** Use `think` with `first_principles` to break arguments into fundamental parts
- **Phase 5 Enhancement:** Use `think` with `first_principles` to classify constraints as hard/soft/assumption
- **Core Insight:** The most devastating critiques come from challenging hidden assumptions
- **Invocation:** 
  ```
  think(problem="[THE ARGUMENT]", mode="first_principles")
  ```

## Subagent Tool Integration

Use the `subagent` tool to dispatch multiple agents in parallel:

**For 32-Agent ParallelAnalysis:**
```json
{
  "agentScope": "both",
  "tasks": [
    { "agent": "worker", "task": "[EN-1 prompt]" },
    { "agent": "worker", "task": "[EN-2 prompt]" },
    // ... all 32 agents
  ]
}
```

**For Adversarial Validation:**
```json
{
  "agentScope": "both",
  "tasks": [
    { "agent": "worker", "task": "[Proposal A prompt]" },
    { "agent": "worker", "task": "[Proposal B prompt]" },
    { "agent": "worker", "task": "[Proposal C prompt]" }
  ]
}
```

## Output Format

- **Format:** Steelman + Counter-argument, each with 8 numbered points
- **Length:** 12-16 words per point (strict discipline)
- **Tone:** Direct, substantive, non-performative
- **Must Include:** First-principles analysis, convergence identification
- **Must Avoid:** Nitpicking, strawmanning, generic objections

## Related Pi Skills

| RedTeam Concept | Pi Equivalent |
|-----------------|---------------|
| FirstPrinciples/Deconstruct | `think` tool with `first_principles` mode |
| FirstPrinciples/Challenge | `think` tool with `first_principles` mode |
| Task tool (parallel agents) | `subagent` tool with `tasks` array |
| VoiceServer notification | `notify` tool |

## Notification Pattern

Send notifications at workflow start:

```
notify(message="Running the ParallelAnalysis workflow in the RedTeam skill to red team arguments", title="RedTeam")
```

---

**Last Updated:** 2025-03-12