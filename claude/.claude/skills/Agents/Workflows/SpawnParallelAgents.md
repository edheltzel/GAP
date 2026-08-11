# SpawnParallelAgents Workflow

**Launches independent agents for the same kind of work across different inputs.**

## Voice Notification

```bash
curl -s -X POST http://localhost:8888/notify \
  -H "Content-Type: application/json" \
  -d '{"message": "Running the SpawnParallelAgents workflow in the Agents skill"}' \
  > /dev/null 2>&1 &
```

Running **SpawnParallelAgents** in **Agents**...

## When to Use

Use this workflow when the request contains several independent units such as files, URLs, companies, or questions. Use direct tools when one small unit can be handled faster without delegation. Use custom role briefs when the units need different perspectives rather than the same method.

Do not parallelize work whose inputs depend on an earlier result.

## Current Agent Controls

Use the current `Agent` tool exposed by the active Pi or Claude harness. Do not write JavaScript or TypeScript that pretends to invoke the tool.

For every dispatch, provide the fields exposed by the current tool schema:

- `description`: a short label for the unit
- `prompt`: the complete task, context, constraints, and expected output
- `subagent_type`: a verified built-in or custom agent type
- `run_in_background`: set `true` when the parent can continue useful work before results arrive
- `max_turns`: optional for tightly bounded one-shot work when the schema supports it
- `isolation`: set `worktree` only when an editing agent needs file isolation and the schema supports it

Issue one `Agent` tool call per independent unit in the same assistant message. This is the parallel launch mechanism. Each call starts a fresh agent, so every prompt must be self-contained.

Trust harness completion notifications. Do not read or tail agent output files and do not poll. In Claude Code, use `/tasks` to inspect running agents and `TaskStop` to cancel a specific run. In Pi, use the returned agent ID with `get_subagent_result` only when results are needed and no completion result is already present.

## Workflow

### Step 1: Build the Unit List

Create a bounded list of independent inputs. Record ordering dependencies and keep dependent items out of the parallel batch.

### Step 2: Write Complete Prompts

Each prompt must include:

1. Shared context that every worker needs
2. The one input assigned to this worker
3. Scope and explicit exclusions
4. Evidence or verification requirements
5. A consistent output contract
6. The timing or size limit when the user supplied one

Example prompt shape:

```markdown
## Context
We are comparing competitors in the AI security market.

## Assigned Input
Acme AI Security Corp

## Required Result
- Product launches from the last six months with dates and sources
- Target market and positioning
- Partnerships or acquisitions with dates and sources
- Technical approach, with uncertainty labeled

## Scope
Stay under 1,500 words. Do not analyze other companies.
```

### Step 3: Launch the Batch

In one assistant response, issue all independent `Agent` calls. Use the same output contract in every prompt. Set `run_in_background` to `true` only when there is useful coordinator work to do while the batch runs. Otherwise collect foreground results directly.

Choose the agent type from the types actually listed by the active harness. Do not assume a specialized type exists because an older workflow named it.

### Step 4: Continue Useful Coordinator Work

While background agents run, prepare the comparison structure, inspect shared context, or handle work that does not depend on their answers. Trust completion notifications instead of polling.

### Step 5: Validate the Batch

After every required agent has completed:

1. Confirm that every input has a result.
2. Check that every result follows the output contract.
3. Compare claims for conflicts, missing evidence, and inconsistent scope.
4. Launch one fresh `Agent` spotcheck with the collected results embedded in its prompt when the batch is too large or specialized to validate directly.
5. Resolve or clearly report any failed or incomplete unit before claiming completion.

### Step 6: Synthesize

Return the shared findings, meaningful differences, evidence gaps, and required follow-up. Do not paste raw worker output without coordinator review.

## Failure Rules

- If one worker fails, retry that unit once with a corrected complete prompt or handle it directly.
- If a required unit remains incomplete, report the gap and do not claim full batch coverage.
- If an agent edits files, verify the actual diff before trusting its completion message.
- If agents would edit overlapping files, use worktree isolation when available or execute those units serially.

## Related Workflows

- **CreateCustomAgent**: distinct personas or perspectives
- **SpawnTeam**: a fixed set of complementary role briefs
- **ListTraits**: inspect the optional trait composition surface
