/**
 * AlgorithmTracker.ts - PAI Algorithm Phase Tracking (OpenCode Plugin)
 *
 * PURPOSE:
 * Tracks the PAI Algorithm phases and state throughout a session.
 * Monitors for phase transitions, criteria changes, and agent spawns.
 *
 * TRIGGER: tool.execute.after (Bash and Task tools)
 *
 * PORTED FROM: AlgorithmTracker.hook.ts (Claude Code)
 *
 * BEHAVIOR:
 * - Detects phase transitions from voice curl commands
 * - Tracks criteria from Task tool calls
 * - Tracks agent spawns from Task tool calls
 * - Manages algorithm state in MEMORY/STATE/algorithms/{sessionId}.json
 */

import {
  readState,
  writeState,
  initState,
  phaseTransition,
  criteriaAdd,
  criteriaUpdate,
  agentAdd,
  agentEnd,
  algorithmEnd,
  sweepStaleActive,
  type AlgorithmPhase,
  type AlgorithmCriterion,
  type AlgorithmAgent
} from './algorithm-state';

const VOICE_CURL_PATTERN = /curl\s+.*localhost:8888\/notify.*-d\s+['"]?\{/i;
const PHASE_PATTERNS: Record<string, AlgorithmPhase> = {
  'entering the observe phase': 'observe',
  'entering the think phase': 'think',
  'entering the plan phase': 'plan',
  'entering the build phase': 'build',
  'entering the execute phase': 'execute',
  'entering the verify phase': 'verify',
  'entering the learn phase': 'learn',
  'algorithm complete': 'complete'
};

function extractPhaseFromCommand(command: string): AlgorithmPhase | null {
  for (const [pattern, phase] of Object.entries(PHASE_PATTERNS)) {
    if (command.toLowerCase().includes(pattern)) {
      return phase;
    }
  }
  return null;
}



function extractAgentFromTask(toolInput: Record<string, unknown>): AlgorithmAgent | null {
  const subagentType = toolInput.subagent_type as string | undefined;
  const description = toolInput.description as string | undefined;
  
  if (!subagentType) return null;
  
  return {
    name: description || subagentType,
    type: subagentType,
    startedAt: new Date().toISOString()
  };
}

function extractCriteriaFromTask(toolInput: Record<string, unknown>): AlgorithmCriterion | null {
  const prompt = toolInput.prompt as string | undefined;
  if (!prompt) return null;
  
  // Look for ISC-style criteria in the prompt
  const iscMatch = prompt.match(/ISC[-\s]*(\d+):\s*(.+?)(?:\n|$)/i);
  if (iscMatch) {
    return {
      id: `ISC-${iscMatch[1]}`,
      text: iscMatch[2].trim(),
      status: 'pending'
    };
  }
  return null;
}

export default async function AlgorithmTracker({
  event,
  tool_name,
  tool_input,
  tool_result,
  session_id,
}: {
  event: string;
  tool_name?: string;
  tool_input?: Record<string, unknown>;
  tool_result?: unknown;
  session_id: string;
}): Promise<void> {
  if (event !== 'tool.execute.after') return;
  if (!tool_name) return;
  
  // Initialize or get state
  let state = readState(session_id);
  if (!state) {
    state = initState(session_id);
    sweepStaleActive(session_id);
  }
  
  // Handle Bash commands (voice curl detection)
  if (tool_name === 'bash' && tool_input) {
    const command = tool_input.command as string;
    if (command && VOICE_CURL_PATTERN.test(command)) {
      const phase = extractPhaseFromCommand(command);
      if (phase) {
        phaseTransition(session_id, phase);
        
        // Log phase transition
        console.error(`[AlgorithmTracker] Phase transition: ${phase}`);
      }
    }
  }
  
  // Handle Task tool calls (criteria and agent tracking)
  if (tool_name === 'task' && tool_input) {
    // Track agent spawn
    const agent = extractAgentFromTask(tool_input);
    if (agent) {
      agentAdd(session_id, agent);
      
      // If this is a completed task, end the agent
      if (tool_result) {
        agentEnd(session_id, agent.name);
      }
    }
    
    // Track criteria
    const criterion = extractCriteriaFromTask(tool_input);
    if (criterion) {
      criteriaAdd(session_id, criterion);
    }
  }
}