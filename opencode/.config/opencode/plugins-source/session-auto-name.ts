/**
 * SessionAutoName.ts - Auto-generate concise session names (OpenCode Plugin)
 *
 * PURPOSE:
 * Generates a 2-3 word session title on the FIRST user prompt in a session,
 * so the status line always shows a meaningful descriptive name.
 *
 * TRIGGER: session.idle (after session finishes processing)
 *
 * PORTED FROM: SessionAutoName.hook.ts (Claude Code)
 *
 * BEHAVIOR:
 * - If session already has a name → skip (no-op)
 * - If no name exists → generate one via fast inference → store it
 * - Names persist across session lifecycle
 */

import { existsSync, readFileSync, writeFileSync, mkdirSync } from 'fs';
import { dirname, join } from 'path';
import { homedir } from 'os';

const CLAUDE_DIR = join(homedir(), '.claude');
const SESSION_NAMES_PATH = join(CLAUDE_DIR, 'MEMORY', 'STATE', 'session-names.json');

interface SessionNames {
  [sessionId: string]: string;
}

const NOISE_WORDS = new Set([
  'the', 'a', 'an', 'i', 'my', 'we', 'you', 'your', 'this', 'that', 'it',
  'is', 'are', 'was', 'were', 'do', 'does', 'did', 'can', 'could', 'should',
  'would', 'will', 'have', 'has', 'had', 'just', 'also', 'need', 'want',
  'please', 'session', 'help', 'work', 'task', 'update', 'new', 'check',
  'make', 'get', 'set', 'put', 'use', 'run', 'try', 'let', 'see', 'look',
  'fix', 'add', 'create', 'build', 'deploy', 'code', 'read', 'write',
  'thing', 'things', 'something', 'going', 'like', 'know', 'think', 'right',
]);

function sanitizePromptForNaming(prompt: string): string {
  return prompt
    .replace(/<[^>]+>/g, ' ')
    .replace(/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/gi, ' ')
    .replace(/\b[0-9a-f]{7,}\b/gi, ' ')
    .replace(/(?:\/[\w.-]+){2,}/g, ' ')
    .replace(/\s+/g, ' ')
    .trim();
}

function extractFallbackName(prompt: string): string | null {
  const words = prompt
    .replace(/[^a-zA-Z\s]/g, ' ')
    .split(/\s+/)
    .filter(w => w.length >= 4 && !NOISE_WORDS.has(w.toLowerCase()));

  if (words.length === 0) return null;

  const topic = words[0].charAt(0).toUpperCase() + words[0].slice(1).toLowerCase();
  return `${topic} Session`;
}

function readSessionNames(): SessionNames {
  try {
    if (existsSync(SESSION_NAMES_PATH)) {
      return JSON.parse(readFileSync(SESSION_NAMES_PATH, 'utf-8'));
    }
  } catch {
    // Corrupted file — start fresh
  }
  return {};
}

function writeSessionNames(names: SessionNames): void {
  const dir = dirname(SESSION_NAMES_PATH);
  if (!existsSync(dir)) {
    mkdirSync(dir, { recursive: true });
  }
  writeFileSync(SESSION_NAMES_PATH, JSON.stringify(names, null, 2), 'utf-8');
}

async function generateSessionName(prompt: string): Promise<string | null> {
  // For OpenCode, we'll use a simple fallback approach
  // since we don't have direct access to an inference API
  const fallback = extractFallbackName(prompt);
  return fallback;
}

export default async function SessionAutoName({ 
  event,
  session_id,
}: {
  event: string;
  session_id: string;
}): Promise<void> {
  if (event !== 'session.idle') return;
  
  const names = readSessionNames();
  
  // Skip if already named
  if (names[session_id]) return;
  
  // For now, just use fallback naming
  // OC doesn't have a direct inference API like CC does
  // In a full implementation, you would call an LLM here
  const prompt = ''; // Would need to extract from session
  
  if (!prompt) return;
  
  const sanitized = sanitizePromptForNaming(prompt);
  const name = await generateSessionName(sanitized);
  
  if (name) {
    names[session_id] = name;
    writeSessionNames(names);
    console.error(`[SessionAutoName] Named session: "${name}"`);
  }
}