# Run Scenario Workflow

Run a multi-turn scenario against an explicit agent-under-test adapter. The scenario simulator and judge may use a configured API provider, but Evals does not import a hidden inference runtime.

## Prerequisites

- Bun
- Dependencies installed from `Evals/package.json` and `Evals/bun.lock`
- `ANTHROPIC_API_KEY` when the scenario uses the Anthropic simulator or judge
- `EVALS_ALLOW_API_BILLING=1` after confirming the intended billed account
- A scenario whose `ExplicitAgentAdapter` receives a caller-supplied `respond` function or supplied result

## Run

Start with one trial:

```bash
bun run ~/.claude/skills/Evals/Tools/ScenarioRunner.ts \
  --scenario ~/.claude/skills/Evals/Scenarios/<name>.scenario.ts \
  --trials 1
```

After validating the adapter, criteria, and billing boundary, run multiple trials:

```bash
bun run ~/.claude/skills/Evals/Tools/ScenarioRunner.ts \
  --scenario ~/.claude/skills/Evals/Scenarios/<name>.scenario.ts \
  --trials 3
```

## Result Contract

Treat stdout and the runner's explicitly reported result path as authoritative. Do not assume a historical global results directory exists. Copy a result into a project directory only when the user selected that destination.

## Failure Behavior

- Missing scenario, API key, billing opt-in, or explicit response adapter must fail clearly.
- An empty adapter result is an error.
- Do not substitute a model, invent a command, or claim a scenario ran when setup failed.
- Do not mutate Algorithm ISC state.
