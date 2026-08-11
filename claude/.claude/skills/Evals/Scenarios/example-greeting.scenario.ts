/**
 * Minimum scenario demonstrating ExplicitAgentAdapter.
 *
 * The user simulator and judge use the configured API provider. The agent under
 * test consumes EVALS_AGENT_RESPONSE as an explicit supplied result.
 */

import { anthropic } from '@ai-sdk/anthropic';
import scenario, { type ScenarioConfig } from '@langwatch/scenario';
import { ExplicitAgentAdapter } from '../Tools/LifeosAgentAdapter.ts';

if (process.env.EVALS_ALLOW_API_BILLING !== '1') {
  throw new Error(
    'Set EVALS_ALLOW_API_BILLING=1 to opt in to API-billed user simulation and judging.',
  );
}

const suppliedResponse = process.env.EVALS_AGENT_RESPONSE;
if (!suppliedResponse) {
  throw new Error('Set EVALS_AGENT_RESPONSE to the explicit agent-under-test output.');
}

const judgeModel = anthropic('claude-sonnet-4-6');

const config: ScenarioConfig = {
  name: 'polite greeting',
  description:
    'A user greets a general-purpose assistant. The assistant should respond politely, in English, and keep the response concise.',
  agents: [
    new ExplicitAgentAdapter({
      name: 'supplied-response-agent',
      systemPrompt: 'You are a concise, polite assistant. Keep replies under 40 words.',
      respond: async () => suppliedResponse,
    }),
    scenario.userSimulatorAgent({ model: judgeModel }),
    scenario.judgeAgent({
      model: judgeModel,
      criteria: [
        'Assistant responds in English',
        'Response is polite',
        'Response is under 40 words',
      ],
    }),
  ],
  script: [scenario.user(), scenario.agent(), scenario.judge()],
  maxTurns: 4,
};

export default config;
