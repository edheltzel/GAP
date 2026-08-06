# CreateCustomAgent Workflow

Create custom agents with distinct trait combinations, prompts, colors, voices, and prosody.

## Notification

```bash
curl -s -X POST http://localhost:8888/notify \
  -H "Content-Type: application/json" \
  -d '{"message":"Running the CreateCustomAgent workflow"}' \
  > /dev/null 2>&1 &
```

Echo notification is best-effort and does not determine workflow success.

## When to Use

Use this workflow when the user asks for custom or specialized agents whose perspectives should differ. Use `SpawnParallelAgents.md` when workers should apply the same method to different inputs.

## Workflow

### 1. Define Bounded Workers

Determine the agent count, shared task, independent unit for each worker, requested traits, constraints, evidence, and output format. Do not parallelize dependent units.

### 2. Compose Distinct Identities

Run `ComposeAgent.ts` once per worker with different trait combinations:

```bash
bun run ~/.claude/skills/Agents/Tools/ComposeAgent.ts \
  --traits "research,enthusiastic,exploratory" \
  --task "Research quantum computing applications" \
  --output json

bun run ~/.claude/skills/Agents/Tools/ComposeAgent.ts \
  --traits "research,skeptical,systematic" \
  --task "Research quantum computing applications" \
  --output json
```

Each result includes a complete prompt, name, color, voice ID, and voice settings.

### 3. Complete Every Prompt

Treat the composed prompt as a starting brief. Ensure each worker prompt also contains:

1. complete shared context
2. exactly one assigned unit
3. explicit scope and exclusions
4. required evidence or verification
5. expected output format
6. timing or size limits supplied by the user

Fresh agents do not inherit the parent conversation. Never rely on implicit context or prompt caching.

### 4. Dispatch with the Current Tool

Use the harness-exposed `Agent` tool. Never show or write a JavaScript or TypeScript pseudo invocation.

Issue one tool call per independent worker in the same assistant message for genuine parallelism. For each call, provide a short `description`, the complete composed `prompt`, and a `subagent_type` verified from the current schema. Use background, isolation, or model fields only when they are exposed and needed.

Do not assume a static role name is a valid agent type. When no specialized type is verified, use the active schema's verified general-purpose type.

In Pi, trust completion notifications and use `get_subagent_result` with the returned agent ID when a background result is needed. In Claude Code, `/tasks` inspects active agents and `TaskStop` may cancel one.

### 5. Validate Results

After completion:

1. confirm every unit returned a result
2. verify the common output contract
3. inspect evidence and resolve conflicts
4. verify actual diffs from editing workers
5. run one fresh spotcheck only when direct validation is insufficient
6. report any missing unit instead of claiming complete coverage

### 6. Voice Output

The dynamic prompt contains the agent's voice ID and prosody. Voice completion is best-effort. A failed notification must not hide or invalidate the textual result.

## Trait Variation

### Research

- enthusiastic + exploratory
- skeptical + thorough
- analytical + systematic
- creative + synthesizing

### Security

- adversarial + bold
- skeptical + meticulous
- cautious + systematic

### Strategy

- bold + rapid
- analytical + comparative
- pragmatic + consultative

## Timing

`--timing fast`, `--timing standard`, and `--timing deep` control output scope in the generated prompt. Timing does not select a model. Model selection belongs to the active harness schema and caller configuration.

## Common Failures

- Reusing the same traits for every custom agent
- Sending incomplete prompts to fresh agents
- Dispatching dependent work concurrently
- Inventing a model alias or agent type
- Trusting an editing worker's summary without checking the diff
- Treating a saved identity as persistent shared worker state

## References

- `~/.claude/skills/Agents/Data/Traits.yaml`
- `~/.claude/skills/Agents/Templates/DynamicAgent.hbs`
- `~/.claude/skills/Agents/Tools/ComposeAgent.ts`
- `~/.claude/skills/Agents/AgentPersonalities.md`
