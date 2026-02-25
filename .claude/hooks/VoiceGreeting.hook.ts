#!/usr/bin/env bun
/**
 * VoiceGreeting.hook.ts - Speak catchphrase at Session Start (async)
 *
 * PURPOSE:
 * Sends voice notification with the startup catchphrase from settings.json.
 * Runs as an async hook so it doesn't block session startup.
 *
 * For named subagents (Intern, Engineer, etc.), announces with THAT agent's
 * voice settings instead of Atlas's — giving each agent a distinct audio identity.
 *
 * TRIGGER: SessionStart (async: true)
 *
 * SIDE EFFECTS:
 * - Sends HTTP request to voice server (fire-and-forget)
 * - Writes a 0-byte flag file to MEMORY/STATE/voice-greeted/{session_id}
 *   to prevent double-fire within the same session.
 *
 * SUBAGENT DETECTION (layered):
 * 1. CLAUDE_AGENT_TYPE === "loop-worker" → skip (algorithm.ts workers)
 * 2. CLAUDE_PROJECT_DIR contains /.claude/Agents/ → skip
 * 3. CLAUDE_AGENT_TYPE is a named agent (Intern, Engineer, etc.) → agent voice
 * 4. session_id flag file already exists → skip (dedup within same session)
 *
 * PERFORMANCE:
 * - Async: Yes (runs in background, doesn't block startup)
 * - Uses Bun.file().json() for native file reads (faster than readFileSync + JSON.parse)
 */

import { existsSync, mkdirSync, readFileSync } from 'fs';
import { join } from 'path';

const CLAUDE_DIR = join(process.env.HOME!, '.claude');

// ── Layer 1: Skip loop-workers (algorithm.ts spawned workers) ─────────────────
const agentType = process.env.CLAUDE_AGENT_TYPE;

if (agentType === 'loop-worker') {
  process.exit(0);
}

// ── Layer 2: Skip for Agents/ path-based subagents ────────────────────────────
const claudeProjectDir = process.env.CLAUDE_PROJECT_DIR || '';
if (claudeProjectDir.includes('/.claude/Agents/')) {
  process.exit(0);
}

// ── Named agent types that get their own voice announcement ──────────────────
const NAMED_AGENT_TYPES = [
  'Algorithm', 'Architect', 'Artist', 'ClaudeResearcher', 'CodexResearcher',
  'Designer', 'Engineer', 'GeminiResearcher', 'GrokResearcher', 'Intern',
  'Pentester', 'PerplexityResearcher', 'QATester',
];

const isNamedAgent = agentType != null && NAMED_AGENT_TYPES.includes(agentType);

// ── Read session_id from stdin (Claude Code passes hook input as JSON) ────────
let sessionId: string | null = null;
try {
  const stdinText = await Bun.stdin.text();
  if (stdinText.trim()) {
    const hookInput = JSON.parse(stdinText);
    sessionId = hookInput.session_id ?? null;
  }
} catch {
  // stdin parse failed — proceed without session_id (no dedup, still greet)
}

// ── Layer 3: Session dedup — only announce once per session_id ────────────────
// Writes a 0-byte flag file before fetching to close the race window.
if (sessionId) {
  const greetedDir = join(CLAUDE_DIR, 'MEMORY', 'STATE', 'voice-greeted');
  const flagFile = join(greetedDir, sessionId);

  if (await Bun.file(flagFile).exists()) {
    process.exit(0); // Already greeted this session
  }

  try {
    if (!existsSync(greetedDir)) mkdirSync(greetedDir, { recursive: true });
    // Write flag BEFORE the voice fetch — prevents a second fire from sneaking through
    // during the async fetch (which can take 100ms+ for TTS)
    await Bun.write(flagFile, '');
  } catch {
    // Non-fatal — flag write failed, still play greeting
  }
}

// ── Parse simple YAML frontmatter from an agent .md file ──────────────────────
interface AgentFrontmatter {
  voiceId?: string;
  voice?: {
    stability?: string;
    similarity_boost?: string;
    style?: string;
    speed?: string;
    use_speaker_boost?: string;
    volume?: string;
  };
  persona?: {
    name?: string;
    title?: string;
  };
}

function parseAgentFrontmatter(content: string): AgentFrontmatter {
  const match = content.match(/^---\n([\s\S]*?)\n---/);
  if (!match) return {};

  const yaml = match[1];
  const result: Record<string, any> = {};
  const lines = yaml.split('\n');
  let currentKey = '';
  let currentObj: Record<string, any> | null = null;

  for (const line of lines) {
    const indent = (line.match(/^(\s*)/)?.[1].length) ?? 0;
    const trimmed = line.trim();
    if (!trimmed || trimmed.startsWith('#')) continue;

    const colonIdx = trimmed.indexOf(':');
    if (colonIdx === -1) continue;

    const key = trimmed.slice(0, colonIdx).trim();
    const val = trimmed.slice(colonIdx + 1).trim();

    if (indent === 0) {
      if (val) {
        // Simple scalar — strip surrounding quotes
        result[key] = val.replace(/^["']|["']$/g, '');
        currentKey = '';
        currentObj = null;
      } else {
        // Nested object
        result[key] = {};
        currentKey = key;
        currentObj = result[key] as Record<string, any>;
      }
    } else if (currentObj) {
      if (val) {
        currentObj[key] = val.replace(/^["']|["']$/g, '');
      }
    }
  }

  return result as AgentFrontmatter;
}

// ── Agent voice announcement ───────────────────────────────────────────────────
if (isNamedAgent && agentType) {
  try {
    // Read Atlas voice ID as fallback (from settings.json)
    const settings = await Bun.file(join(CLAUDE_DIR, 'settings.json')).json();
    const atlasVoiceId: string =
      settings.daidentity?.voices?.main?.voiceId ||
      settings.daidentity?.voiceId ||
      '';

    // Read agent .md frontmatter
    const agentFile = join(CLAUDE_DIR, 'agents', `${agentType}.md`);
    let fm: AgentFrontmatter = {};
    try {
      const agentContent = readFileSync(agentFile, 'utf-8');
      fm = parseAgentFrontmatter(agentContent);
    } catch {
      // Agent file not found — still announce with Atlas fallback voice
    }

    // Resolve voice ID: use agent's own if real, else fall back to Atlas
    const PLACEHOLDER = 'YOUR_VOICE_ID_HERE';
    const voiceId = (fm.voiceId && fm.voiceId !== PLACEHOLDER)
      ? fm.voiceId
      : atlasVoiceId;

    // Build announcement message using agent's persona name if available
    const personaName = fm.persona?.name;
    const message = personaName
      ? `${personaName}, standing by`
      : `${agentType} online`;

    // Build voice_settings from agent frontmatter (gives each agent a distinct sound)
    const voice = fm.voice;
    const voice_settings = voice ? {
      stability: parseFloat(voice.stability ?? '0.5'),
      similarity_boost: parseFloat(voice.similarity_boost ?? '0.8'),
      style: parseFloat(voice.style ?? '0.5'),
      speed: parseFloat(voice.speed ?? '1.0'),
    } : undefined;

    const body: Record<string, unknown> = { message };
    if (voiceId) body.voice_id = voiceId;
    if (voice_settings) body.voice_settings = voice_settings;

    await fetch('http://localhost:8888/notify', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });
  } catch {
    // Voice server not running — silent fail
  }

  process.exit(0);
}

// ── Atlas greeting (main session) ─────────────────────────────────────────────
try {
  // Bun.file().json() is faster than readFileSync + JSON.parse:
  // uses Bun's native C++ file I/O and SIMD JSON parser
  const settings = await Bun.file(join(CLAUDE_DIR, 'settings.json')).json();

  const daName = settings.daidentity?.displayName || settings.daidentity?.name || 'Atlas';
  const catchphrase = (settings.daidentity?.startupCatchphrase || `${daName} standing by`)
    .replace(/\{name\}/gi, daName);
  const personality = settings.daidentity?.personality;

  const url = personality?.baseVoice
    ? 'http://localhost:8888/notify/personality'
    : 'http://localhost:8888/notify';

  const body = personality?.baseVoice
    ? {
        message: `[🎯 focused] ${catchphrase}`,
        personality: {
          name: daName.toLowerCase(),
          base_voice: personality.baseVoice,
          enthusiasm: personality.enthusiasm,
          energy: personality.energy,
          expressiveness: personality.expressiveness,
          resilience: personality.resilience,
          composure: personality.composure,
          optimism: personality.optimism,
          warmth: personality.warmth,
          formality: personality.formality,
          directness: personality.directness,
          precision: personality.precision,
          curiosity: personality.curiosity,
          playfulness: personality.playfulness,
        },
      }
    : { message: catchphrase, play: true };

  await fetch(url, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  });
} catch {
  // Voice server not running or settings unreadable — silent fail
}
