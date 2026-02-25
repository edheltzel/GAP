#!/usr/bin/env bun
/**
 * RatingCapture.hook.ts - Unified Rating & Sentiment Capture (UserPromptSubmit)
 *
 * PURPOSE:
 * Single hook for all rating capture. Handles both explicit ratings (1-10 pattern)
 * and implicit sentiment detection (AI inference). Also outputs the Algorithm
 * format reminder (absorbed from AlgorithmEnforcement.hook.ts).
 *
 * TRIGGER: UserPromptSubmit
 *
 * FLOW:
 * 1. Output algorithm format reminder (instant, stdout — non-blocking)
 * 2. Parse input from stdin
 * 3. Check for explicit rating pattern → if found, write and exit
 * 4. If no explicit rating, run AI sentiment inference (Haiku, ~1s)
 * 5. Write result to ratings.jsonl
 * 6. Capture learnings for low ratings (<6), full failure capture for <=3
 *
 * OUTPUT:
 * - stdout: <user-prompt-submit-hook> algorithm reminder (instant)
 * - exit(0): Normal completion
 *
 * SIDE EFFECTS:
 * - Writes to: MEMORY/LEARNING/SIGNALS/ratings.jsonl
 * - Writes to: MEMORY/LEARNING/<category>/<YYYY-MM>/*.md (for low ratings)
 * - Triggers: TrendingAnalysis.ts update (fire-and-forget)
 * - API call: Haiku inference for implicit sentiment (fast/cheap)
 *
 * INTER-HOOK RELATIONSHIPS:
 * - DEPENDS ON: None
 * - COORDINATES WITH: None (self-contained — replaces ExplicitRatingCapture + ImplicitSentimentCapture)
 *
 * PERFORMANCE:
 * - Algorithm reminder: <1ms (immediate stdout)
 * - Explicit rating path: <50ms (no inference)
 * - Implicit sentiment path: 0.5-1.5s (Haiku inference)
 *
 * HISTORY:
 * - Consolidated from ExplicitRatingCapture + ImplicitSentimentCapture + AlgorithmEnforcement
 * - Removes ~200 lines of duplicated code (shared writeRating, captureLearning, trending)
 */

import { appendFileSync, mkdirSync, existsSync, readFileSync, writeFileSync } from 'fs';
import { join } from 'path';
import { getPrincipalName } from './lib/identity';
import { getLearningCategory } from './lib/learning-utils';
import { getISOTimestamp, getPSTComponents } from './lib/time';
import { captureFailure } from '../skills/PAI/Tools/FailureCapture';

const SENTIMENT_WORKER = join(import.meta.dir, 'workers', 'SentimentWorker.ts');
const MAX_WORKER_PROCS = 3;

// ── Algorithm Format Reminder (absorbed from AlgorithmEnforcement) ──
// Output IMMEDIATELY before any async work — this is blocking stdout injection.
const ALGORITHM_REMINDER = `<user-prompt-submit-hook>
\u{1F6A8} ALGORITHM FORMAT REQUIRED - EVERY RESPONSE \u{1F6A8}

START WITH:
\u{267B}\u{FE0F} Entering the PAI ALGORITHM\u{2026} (v0.4.9 | github.com/danielmiessler/TheAlgorithm) \u{2550}\u{2550}\u{2550}\u{2550}\u{2550}\u{2550}\u{2550}\u{2550}\u{2550}\u{2550}\u{2550}\u{2550}\u{2550}

EXECUTE VOICE CURLS at each phase (OBSERVE, THINK, PLAN, BUILD, EXECUTE, VERIFY, LEARN)

USE TaskCreate for ISC criteria. USE TaskList to display them. NEVER manual tables.

END WITH:
\u{1F5E3}\u{FE0F} {DAIDENTITY.NAME}: [12-24 word spoken summary]

For MINIMAL tasks (pure greetings, ratings): Use abbreviated format but STILL include header and voice line.
</user-prompt-submit-hook>`;

console.log(ALGORITHM_REMINDER);

// ── Shared Types ──

interface HookInput {
  session_id: string;
  prompt?: string;
  user_prompt?: string;  // Legacy field name
  transcript_path: string;
  hook_event_name: string;
}

interface RatingEntry {
  timestamp: string;
  rating: number;
  session_id: string;
  comment?: string;
  source?: 'implicit';
  sentiment_summary?: string;
  confidence?: number;
}

// ── Shared Constants ──

const BASE_DIR = process.env.PAI_DIR || join(process.env.HOME!, '.claude');
const SIGNALS_DIR = join(BASE_DIR, 'MEMORY', 'LEARNING', 'SIGNALS');
const RATINGS_FILE = join(SIGNALS_DIR, 'ratings.jsonl');
const TRENDING_SCRIPT = join(BASE_DIR, 'tools', 'TrendingAnalysis.ts');
const MIN_PROMPT_LENGTH = 3;
const MIN_CONFIDENCE = 0.5;
const ANALYSIS_TIMEOUT = 15000;

// ── Stdin Reader ──

async function readStdinWithTimeout(timeout: number = 5000): Promise<string> {
  return new Promise((resolve, reject) => {
    let data = '';
    const timer = setTimeout(() => reject(new Error('Timeout')), timeout);
    process.stdin.on('data', (chunk) => { data += chunk.toString(); });
    process.stdin.on('end', () => { clearTimeout(timer); resolve(data); });
    process.stdin.on('error', (err) => { clearTimeout(timer); reject(err); });
  });
}

// ── Explicit Rating Detection ──

/**
 * Parse explicit rating pattern from prompt.
 * Matches: "7", "8 - good work", "6: needs work", "9 excellent", "10!"
 * Rejects: "3 items", "5 things to fix", "7th thing"
 */
function parseExplicitRating(prompt: string): { rating: number; comment?: string } | null {
  const trimmed = prompt.trim();
  const ratingPattern = /^(10|[1-9])(?:\s*[-:]\s*|\s+)?(.*)$/;
  const match = trimmed.match(ratingPattern);
  if (!match) return null;

  const rating = parseInt(match[1], 10);
  const comment = match[2]?.trim() || undefined;

  if (rating < 1 || rating > 10) return null;

  // Reject if comment starts with words indicating a sentence, not a rating
  if (comment) {
    const sentenceStarters = /^(items?|things?|steps?|files?|lines?|bugs?|issues?|errors?|times?|minutes?|hours?|days?|seconds?|percent|%|th\b|st\b|nd\b|rd\b|of\b|in\b|at\b|to\b|the\b|a\b|an\b)/i;
    if (sentenceStarters.test(comment)) return null;
  }

  return { rating, comment };
}

// ── Shared: Write Rating ──

function writeRating(entry: RatingEntry): void {
  if (!existsSync(SIGNALS_DIR)) mkdirSync(SIGNALS_DIR, { recursive: true });
  appendFileSync(RATINGS_FILE, JSON.stringify(entry) + '\n', 'utf-8');
  const source = entry.source === 'implicit' ? 'implicit' : 'explicit';
  console.error(`[RatingCapture] Wrote ${source} rating ${entry.rating} to ${RATINGS_FILE}`);
}

// ── Shared: Trigger Trending Analysis ──

function triggerTrending(): void {
  if (existsSync(TRENDING_SCRIPT)) {
    Bun.spawn(['bun', TRENDING_SCRIPT, '--force'], { stdout: 'ignore', stderr: 'ignore' });
    console.error('[RatingCapture] Triggered TrendingAnalysis update');
  }
}

// ── Shared: Capture Low Rating Learning ──

function captureLowRatingLearning(
  rating: number,
  summaryOrComment: string,
  detailedContext: string,
  source: 'explicit' | 'implicit'
): void {
  if (rating >= 6) return;

  const { year, month, day, hours, minutes, seconds } = getPSTComponents();
  const yearMonth = `${year}-${month}`;
  const category = getLearningCategory(detailedContext, summaryOrComment);
  const learningsDir = join(BASE_DIR, 'MEMORY', 'LEARNING', category, yearMonth);

  if (!existsSync(learningsDir)) mkdirSync(learningsDir, { recursive: true });

  const label = source === 'explicit' ? `low-rating-${rating}` : `sentiment-rating-${rating}`;
  const filename = `${year}-${month}-${day}-${hours}${minutes}${seconds}_LEARNING_${label}.md`;
  const filepath = join(learningsDir, filename);

  const tags = source === 'explicit'
    ? '[low-rating, improvement-opportunity]'
    : '[sentiment-detected, implicit-rating, improvement-opportunity]';

  const content = `---
capture_type: LEARNING
timestamp: ${year}-${month}-${day} ${hours}:${minutes}:${seconds} PST
rating: ${rating}
source: ${source}
auto_captured: true
tags: ${tags}
---

# ${source === 'explicit' ? 'Low Rating' : 'Implicit Low Rating'} Captured: ${rating}/10

**Date:** ${year}-${month}-${day}
**Rating:** ${rating}/10
**Detection Method:** ${source === 'explicit' ? 'Explicit Rating' : 'Sentiment Analysis'}
${summaryOrComment ? `**Feedback:** ${summaryOrComment}` : ''}

---

## Context

${detailedContext || 'No context available'}

---

## Improvement Notes

This response was rated ${rating}/10 by ${getPrincipalName()}. Use this as an improvement opportunity.

---
`;

  writeFileSync(filepath, content, 'utf-8');
  console.error(`[RatingCapture] Captured low ${source} rating learning to ${filepath}`);
}

// ── Main ──

async function main() {
  try {
    console.error('[RatingCapture] Hook started');
    const input = await readStdinWithTimeout();
    const data: HookInput = JSON.parse(input);
    const prompt = data.prompt || data.user_prompt || '';

    // ── Path 1: Explicit Rating ──
    const explicitResult = parseExplicitRating(prompt);
    if (explicitResult) {
      console.error(`[RatingCapture] Explicit rating: ${explicitResult.rating}${explicitResult.comment ? ` - ${explicitResult.comment}` : ''}`);

      const entry: RatingEntry = {
        timestamp: getISOTimestamp(),
        rating: explicitResult.rating,
        session_id: data.session_id,
      };
      if (explicitResult.comment) entry.comment = explicitResult.comment;

      writeRating(entry);
      triggerTrending();

      if (explicitResult.rating < 6) {
        // Get last response for context
        let responseContext = '';
        try {
          if (data.transcript_path && existsSync(data.transcript_path)) {
            const content = readFileSync(data.transcript_path, 'utf-8');
            const lines = content.trim().split('\n');
            let lastAssistant = '';
            for (const line of lines) {
              try {
                const e = JSON.parse(line);
                if (e.type === 'assistant' && e.message?.content) {
                  const text = typeof e.message.content === 'string'
                    ? e.message.content
                    : Array.isArray(e.message.content)
                      ? e.message.content.filter((c: any) => c.type === 'text').map((c: any) => c.text).join(' ')
                      : '';
                  if (text) lastAssistant = text;
                }
              } catch {}
            }
            const summaryMatch = lastAssistant.match(/SUMMARY:\s*([^\n]+)/i);
            responseContext = summaryMatch ? summaryMatch[1].trim() : lastAssistant.slice(0, 500);
          }
        } catch {}

        captureLowRatingLearning(explicitResult.rating, explicitResult.comment || '', responseContext, 'explicit');

        if (explicitResult.rating <= 3) {
          try {
            await captureFailure({
              transcriptPath: data.transcript_path,
              rating: explicitResult.rating,
              sentimentSummary: explicitResult.comment || `Explicit low rating: ${explicitResult.rating}/10`,
              detailedContext: responseContext,
              sessionId: data.session_id,
            });
            console.error(`[RatingCapture] Created failure capture for explicit rating ${explicitResult.rating}`);
          } catch (err) {
            console.error(`[RatingCapture] Error creating failure capture: ${err}`);
          }
        }
      }

      process.exit(0);
    }

    // ── Path 2: Implicit Sentiment (detached background worker) ──
    // Spawn SentimentWorker as detached process and exit immediately.
    // This reduces UserPromptSubmit overhead from ~1s to <50ms.
    if (prompt.length < MIN_PROMPT_LENGTH) {
      console.error('[RatingCapture] Prompt too short for sentiment, exiting');
      process.exit(0);
    }

    // Guard: don't accumulate workers if API is slow (Red Team ISC-A3)
    try {
      const check = Bun.spawnSync(['pgrep', '-fc', 'SentimentWorker'], { stdout: 'pipe' });
      const count = parseInt(new TextDecoder().decode(check.stdout).trim(), 10) || 0;
      if (count >= MAX_WORKER_PROCS) {
        console.error(`[RatingCapture] ${count} SentimentWorker procs running, skipping spawn`);
        process.exit(0);
      }
    } catch { /* pgrep unavailable — proceed */ }

    // Serialize payload and spawn detached worker
    const payload = Buffer.from(JSON.stringify({
      session_id: data.session_id,
      prompt,
      transcript_path: data.transcript_path,
    })).toString('base64');

    try {
      Bun.spawn(['bun', SENTIMENT_WORKER, payload], {
        stdout: 'ignore',
        stderr: 'ignore',
        detached: true,
        env: { ...process.env },
      });
      console.error('[RatingCapture] SentimentWorker spawned (detached)');
    } catch (err) {
      console.error(`[RatingCapture] Failed to spawn worker: ${err}`);
    }

    process.exit(0);
  } catch (err) {
    console.error(`[RatingCapture] Error: ${err}`);
    process.exit(0);
  }
}

main();
