#!/usr/bin/env bun
/**
 * ModelRouter.hook.ts — Phase-Aware Model Routing (PreToolUse: Task)
 *
 * PURPOSE:
 * Injects a system-reminder recommending the appropriate model tier when Atlas
 * spawns a Task agent, based on the current Algorithm phase.
 *
 * Phase → Model mapping:
 *   OBSERVE, THINK, PLAN  → opus   (deep analysis, design, ISC creation)
 *   BUILD, EXECUTE        → sonnet (implementation, execution)
 *   VERIFY, LEARN         → sonnet (verification runs, learning synthesis)
 *   IDLE, COMPLETE        → sonnet (default)
 *
 * IMPORTANT LIMITATION:
 * PreToolUse hooks cannot mutate tool input parameters — only Claude Code itself
 * can do that. This hook injects a system-reminder that ATLAS READS and uses to
 * set the `model` parameter explicitly in Task calls. It is advisory, not forced.
 * The sub-agent frontmatter and CLAUDE_CODE_SUBAGENT_MODEL env var are the
 * enforcement layers. This hook is the guidance layer.
 *
 * TRIGGER: PreToolUse (matcher: Task)
 * PERFORMANCE: ~5ms (single file read, no network I/O)
 * NON-BLOCKING: Always exits 0 — never blocks agent spawning
 */

import { existsSync, readFileSync } from 'fs';
import { join } from 'path';
import { homedir } from 'os';

const PAI_DIR = process.env.PAI_DIR || join(homedir(), '.claude');
const ALGORITHMS_DIR = join(PAI_DIR, 'MEMORY', 'STATE', 'algorithms');

type AlgorithmPhase =
  | 'OBSERVE' | 'THINK' | 'PLAN'
  | 'BUILD' | 'EXECUTE' | 'VERIFY' | 'LEARN'
  | 'IDLE' | 'COMPLETE';

interface AlgorithmState {
  sessionId: string;
  currentPhase: AlgorithmPhase;
  sla?: string;
}

interface HookInput {
  session_id?: string;
  tool_name: string;
  tool_input: {
    subagent_type?: string;
    description?: string;
    model?: string;
    run_in_background?: boolean;
  };
}

const PLANNING_PHASES: AlgorithmPhase[] = ['OBSERVE', 'THINK', 'PLAN'];
const EXECUTION_PHASES: AlgorithmPhase[] = ['BUILD', 'EXECUTE', 'VERIFY', 'LEARN'];

// Agent types that already have correct model routing — skip them
const SKIP_TYPES = ['Explore']; // haiku hardcoded in Claude Code

async function readStdin(timeout = 1000): Promise<string> {
  return new Promise((resolve) => {
    let data = '';
    const timer = setTimeout(() => resolve(data), timeout);
    process.stdin.on('data', chunk => { data += chunk.toString(); });
    process.stdin.on('end', () => { clearTimeout(timer); resolve(data); });
    process.stdin.on('error', () => { clearTimeout(timer); resolve(''); });
  });
}

function readAlgorithmState(sessionId: string): AlgorithmState | null {
  try {
    const file = join(ALGORITHMS_DIR, `${sessionId}.json`);
    if (!existsSync(file)) return null;
    return JSON.parse(readFileSync(file, 'utf-8')) as AlgorithmState;
  } catch {
    return null;
  }
}

async function main() {
  try {
    const raw = await readStdin();
    if (!raw) process.exit(0);

    const data: HookInput = JSON.parse(raw);
    const { session_id, tool_input } = data;

    // Skip if no session context
    if (!session_id) process.exit(0);

    // Skip if agent already has explicit model set
    if (tool_input.model) process.exit(0);

    // Skip fast-tier agent types
    const agentType = tool_input.subagent_type || '';
    if (SKIP_TYPES.includes(agentType)) process.exit(0);

    // Read current algorithm phase
    const state = readAlgorithmState(session_id);
    if (!state) process.exit(0);

    const phase = state.currentPhase;
    const effortLevel = state.sla || 'Standard';

    let recommendedModel: string;
    let reasoning: string;

    if (PLANNING_PHASES.includes(phase)) {
      recommendedModel = 'opus';
      reasoning = `${phase} is a deep-thinking phase (analysis, design, ISC creation)`;
    } else if (EXECUTION_PHASES.includes(phase)) {
      recommendedModel = 'sonnet';
      reasoning = `${phase} is an execution phase (implementation, verification)`;
    } else {
      // IDLE / COMPLETE — default to sonnet
      recommendedModel = 'sonnet';
      reasoning = 'no active algorithm phase';
    }

    // Only inject reminder if there's a meaningful recommendation
    // (i.e., we actually know the phase and have a recommendation)
    console.log(`<system-reminder>
MODEL ROUTER — Current phase: ${phase} | Effort: ${effortLevel}
Recommended model for this Task spawn: **${recommendedModel}**
Reason: ${reasoning}

Set model: "${recommendedModel}" in this Task call unless the agent's own
frontmatter already specifies a model (Algorithm=opus, Engineer=sonnet, etc.).
Frontmatter overrides this recommendation.
</system-reminder>`);

    process.exit(0);
  } catch {
    // Never block agent execution on hook error
    process.exit(0);
  }
}

main();
