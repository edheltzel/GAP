# Create Scenario Workflow

Create a multi-turn `@langwatch/scenario` definition without importing a hidden model runtime.

## Required Inputs

1. Scenario name and description
2. System prompt for the agent under test
3. A caller-supplied `respond` adapter
4. Narrow, testable judge criteria
5. Maximum turns, default 6

## Scaffold

```ts
import { anthropic } from '@ai-sdk/anthropic';
import scenario, { type ScenarioConfig } from '@langwatch/scenario';
import { ExplicitAgentAdapter } from '../Tools/LifeosAgentAdapter.ts';

const judgeModel = anthropic('claude-sonnet-4-6');

const config: ScenarioConfig = {
  name: '<scenario-name>',
  description: '<what happens in plain English>',
  agents: [
    new ExplicitAgentAdapter({
      name: '<agent-name>',
      systemPrompt: '<system prompt>',
      respond: async ({ systemPrompt, userPrompt }) => {
        // Call the current harness adapter explicitly here.
        // Throw if no approved adapter is available.
        throw new Error(`No respond adapter configured for ${systemPrompt.length + userPrompt.length} prompt characters`);
      },
    }),
    scenario.userSimulatorAgent({ model: judgeModel }),
    scenario.judgeAgent({
      model: judgeModel,
      criteria: ['<criterion 1>', '<criterion 2>'],
    }),
  ],
  script: [scenario.user(), scenario.agent(), scenario.judge()],
  maxTurns: 6,
};

export default config;
```

The example throws deliberately. Replace `respond` with an explicit current adapter supplied by the caller. Do not invent a model command or import a removed inference module.

## Smoke Test

```bash
bun run ~/.claude/skills/Evals/Tools/ScenarioRunner.ts \
  --scenario ~/.claude/skills/Evals/Scenarios/<name>.scenario.ts
```

Set `EVALS_ALLOW_API_BILLING=1` only after confirming that the selected simulator and judge providers bill the intended account.

## Authoring Guidance

- Keep criteria narrow and observable.
- Treat `maxTurns` as a ceiling, not a target.
- Use a deterministic starting message when repeatability matters.
- Fail closed when the approved agent-under-test adapter is unavailable.
- Run at least three trials before treating a scenario as regression evidence.
