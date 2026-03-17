/**
 * algorithm-state.ts - PAI Algorithm State Management Library
 *
 * PORTED FROM: algorithm-state.ts (Claude Code)
 *
 * Provides state management for the PAI Algorithm tracking system.
 * State is persisted to MEMORY/STATE/algorithms/{sessionId}.json
 */

import { existsSync, readFileSync, writeFileSync, mkdirSync, readdirSync, statSync, unlinkSync } from 'fs';
import { join } from 'path';
import { homedir } from 'os';

const MEMORY_DIR = join(homedir(), '.local', 'share', 'opencode', 'MEMORY');
const STATE_DIR = join(MEMORY_DIR, 'STATE');
const ALGORITHMS_DIR = join(STATE_DIR, 'algorithms');

export type AlgorithmPhase = 'observe' | 'think' | 'plan' | 'build' | 'execute' | 'verify' | 'learn' | 'complete';

export interface AlgorithmCriterion {
  id: string;
  text: string;
  status: 'pending' | 'in_progress' | 'completed' | 'failed';
  taskId?: string;
}

export interface AlgorithmAgent {
  name: string;
  type: string;
  startedAt: string;
  endedAt?: string;
}

export interface AlgorithmState {
  sessionId: string;
  active: boolean;
  phase: AlgorithmPhase;
  phaseHistory: AlgorithmPhase[];
  effort?: 'low' | 'medium' | 'high';
  criteria: AlgorithmCriterion[];
  agents: AlgorithmAgent[];
  capabilities: string[];
  startedAt: string;
  updatedAt: string;
  enrichment?: string;
}

function ensureDirs(): void {
  if (!existsSync(ALGORITHMS_DIR)) {
    mkdirSync(ALGORITHMS_DIR, { recursive: true });
  }
}

export function readState(sessionId: string): AlgorithmState | null {
  ensureDirs();
  const path = join(ALGORITHMS_DIR, `${sessionId}.json`);
  try {
    if (existsSync(path)) {
      const data = JSON.parse(readFileSync(path, 'utf-8'));
      return data as AlgorithmState;
    }
  } catch { /* corrupt — ignore */ }
  return null;
}

export function writeState(state: AlgorithmState): void {
  ensureDirs();
  const path = join(ALGORITHMS_DIR, `${state.sessionId}.json`);
  state.updatedAt = new Date().toISOString();
  writeFileSync(path, JSON.stringify(state, null, 2), 'utf-8');
}

export function initState(sessionId: string): AlgorithmState {
  const state: AlgorithmState = {
    sessionId,
    active: true,
    phase: 'observe',
    phaseHistory: ['observe'],
    criteria: [],
    agents: [],
    capabilities: [],
    startedAt: new Date().toISOString(),
    updatedAt: new Date().toISOString()
  };
  writeState(state);
  return state;
}

export function phaseTransition(sessionId: string, phase: AlgorithmPhase): void {
  let state = readState(sessionId);
  if (!state) state = initState(sessionId);
  
  if (state.phase !== phase) {
    state.phaseHistory.push(phase);
    state.phase = phase;
    writeState(state);
  }
}

export function criteriaAdd(sessionId: string, criterion: AlgorithmCriterion): void {
  let state = readState(sessionId);
  if (!state) state = initState(sessionId);
  
  const existing = state.criteria.find(c => c.id === criterion.id);
  if (!existing) {
    state.criteria.push(criterion);
    writeState(state);
  }
}

export function criteriaUpdate(sessionId: string, criterionId: string, status: AlgorithmCriterion['status']): void {
  const state = readState(sessionId);
  if (!state) return;
  
  const criterion = state.criteria.find(c => c.id === criterionId);
  if (criterion) {
    criterion.status = status;
    writeState(state);
  }
}

export function agentAdd(sessionId: string, agent: AlgorithmAgent): void {
  let state = readState(sessionId);
  if (!state) state = initState(sessionId);
  
  const existing = state.agents.find(a => a.name === agent.name && !a.endedAt);
  if (!existing) {
    state.agents.push(agent);
    writeState(state);
  }
}

export function agentEnd(sessionId: string, agentName: string): void {
  const state = readState(sessionId);
  if (!state) return;
  
  const agent = state.agents.find(a => a.name === agentName && !a.endedAt);
  if (agent) {
    agent.endedAt = new Date().toISOString();
    writeState(state);
  }
}

export function effortLevelUpdate(sessionId: string, level: 'low' | 'medium' | 'high'): void {
  const state = readState(sessionId);
  if (!state) return;
  
  state.effort = level;
  writeState(state);
}

export function algorithmEnd(sessionId: string, enrichment?: string): void {
  const state = readState(sessionId);
  if (!state) return;
  
  state.active = false;
  state.phase = 'complete';
  if (enrichment) state.enrichment = enrichment;
  writeState(state);
}

export function sweepStaleActive(currentSessionId: string): void {
  ensureDirs();
  const files = readdirSync(ALGORITHMS_DIR);
  const now = Date.now();
  const staleThreshold = 24 * 60 * 60 * 1000; // 24 hours
  
  for (const file of files) {
    if (!file.endsWith('.json')) continue;
    const sessionId = file.replace('.json', '');
    if (sessionId === currentSessionId) continue;
    
    const path = join(ALGORITHMS_DIR, file);
    const stat = statSync(path);
    
    if (now - stat.mtimeMs > staleThreshold) {
      const state = readState(sessionId);
      if (state && state.active) {
        state.active = false;
        writeState(state);
      }
    }
  }
}

export function getActiveSessions(): AlgorithmState[] {
  ensureDirs();
  const files = readdirSync(ALGORITHMS_DIR);
  const active: AlgorithmState[] = [];
  
  for (const file of files) {
    if (!file.endsWith('.json')) continue;
    const state = readState(file.replace('.json', ''));
    if (state && state.active) {
      active.push(state);
    }
  }
  
  return active;
}