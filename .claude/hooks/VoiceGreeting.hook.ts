#!/usr/bin/env bun
/**
 * VoiceGreeting.hook.ts - Speak catchphrase at Session Start (async)
 *
 * PURPOSE:
 * Sends voice notification with the startup catchphrase from settings.json.
 * Runs as an async hook so it doesn't block session startup.
 *
 * TRIGGER: SessionStart (async: true)
 *
 * SIDE EFFECTS:
 * - Sends HTTP request to voice server (fire-and-forget)
 *
 * PERFORMANCE:
 * - Async: Yes (runs in background, doesn't block startup)
 * - Skipped for subagents: Yes
 */

import { readFileSync } from 'fs';
import { join } from 'path';

const CLAUDE_DIR = join(process.env.HOME!, '.claude');

// Skip for subagents
const claudeProjectDir = process.env.CLAUDE_PROJECT_DIR || '';
const isSubagent = claudeProjectDir.includes('/.claude/Agents/') ||
                   process.env.CLAUDE_AGENT_TYPE !== undefined;

if (isSubagent) {
  process.exit(0);
}

try {
  const settings = JSON.parse(readFileSync(join(CLAUDE_DIR, 'settings.json'), 'utf-8'));

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
  // Voice server not running — silent fail
}
