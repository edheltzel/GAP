#!/usr/bin/env bun
/**
 * SentimentWorker.ts - Detached implicit sentiment analysis worker
 *
 * PURPOSE:
 * Runs AI sentiment inference in the background, detached from the
 * UserPromptSubmit hook lifecycle. Called by RatingCapture.hook.ts
 * for the implicit sentiment path so the hook can exit immediately.
 *
 * INVOCATION:
 *   bun SentimentWorker.ts <base64_json_payload>
 *
 * PAYLOAD:
 *   { session_id, prompt, transcript_path }
 *
 * SIDE EFFECTS:
 * - Writes to: MEMORY/LEARNING/SIGNALS/ratings.jsonl
 * - Writes learning files for ratings < 6
 * - Triggers TrendingAnalysis.ts (fire-and-forget)
 */

import { appendFileSync, mkdirSync, existsSync, readFileSync, writeFileSync } from 'fs';
import { join } from 'path';
import { inference } from '../../skills/PAI/Tools/Inference';
import { getIdentity, getPrincipal, getPrincipalName } from '../lib/identity';
import { getLearningCategory } from '../lib/learning-utils';
import { getISOTimestamp, getPSTComponents } from '../lib/time';
import { captureFailure } from '../../skills/PAI/Tools/FailureCapture';

const BASE_DIR = process.env.PAI_DIR || join(process.env.HOME!, '.claude');
const SIGNALS_DIR = join(BASE_DIR, 'MEMORY', 'LEARNING', 'SIGNALS');
const RATINGS_FILE = join(SIGNALS_DIR, 'ratings.jsonl');
const TRENDING_SCRIPT = join(BASE_DIR, 'tools', 'TrendingAnalysis.ts');
const MIN_CONFIDENCE = 0.5;

// ── Parse args ──
const payload = process.argv[2];
if (!payload) {
  console.error('[SentimentWorker] No payload provided, exiting');
  process.exit(0);
}

let sessionId: string, prompt: string, transcriptPath: string;
try {
  const data = JSON.parse(Buffer.from(payload, 'base64').toString('utf-8'));
  sessionId = data.session_id;
  prompt = data.prompt;
  transcriptPath = data.transcript_path;
} catch (err) {
  console.error(`[SentimentWorker] Failed to parse payload: ${err}`);
  process.exit(0);
}

// ── Helpers (duplicated from RatingCapture to keep worker self-contained) ──

function getRecentContext(transcriptPath: string, maxTurns: number = 3): string {
  try {
    if (!transcriptPath || !existsSync(transcriptPath)) return '';
    const content = readFileSync(transcriptPath, 'utf-8');
    const lines = content.trim().split('\n');
    const turns: { role: string; text: string }[] = [];
    for (const line of lines) {
      if (!line.trim()) continue;
      try {
        const entry = JSON.parse(line);
        if (entry.type === 'user' && entry.message?.content) {
          let text = '';
          if (typeof entry.message.content === 'string') text = entry.message.content;
          else if (Array.isArray(entry.message.content)) {
            text = entry.message.content.filter((c: any) => c.type === 'text').map((c: any) => c.text).join(' ');
          }
          if (text.trim()) turns.push({ role: 'User', text: text.slice(0, 200) });
        }
        if (entry.type === 'assistant' && entry.message?.content) {
          const text = typeof entry.message.content === 'string'
            ? entry.message.content
            : Array.isArray(entry.message.content)
              ? entry.message.content.filter((c: any) => c.type === 'text').map((c: any) => c.text).join(' ')
              : '';
          if (text) {
            const summaryMatch = text.match(/SUMMARY:\s*([^\n]+)/i);
            turns.push({ role: 'Assistant', text: summaryMatch ? summaryMatch[1] : text.slice(0, 150) });
          }
        }
      } catch {}
    }
    const recent = turns.slice(-maxTurns);
    return recent.length > 0 ? recent.map(t => `${t.role}: ${t.text}`).join('\n') : '';
  } catch { return ''; }
}

function writeRating(entry: object): void {
  if (!existsSync(SIGNALS_DIR)) mkdirSync(SIGNALS_DIR, { recursive: true });
  appendFileSync(RATINGS_FILE, JSON.stringify(entry) + '\n', 'utf-8');
}

function triggerTrending(): void {
  if (existsSync(TRENDING_SCRIPT)) {
    Bun.spawn(['bun', TRENDING_SCRIPT, '--force'], { stdout: 'ignore', stderr: 'ignore' });
  }
}

function captureLowRatingLearning(
  rating: number,
  summaryOrComment: string,
  detailedContext: string,
): void {
  if (rating >= 6) return;
  const { year, month, day, hours, minutes, seconds } = getPSTComponents();
  const yearMonth = `${year}-${month}`;
  const category = getLearningCategory(detailedContext, summaryOrComment);
  const learningsDir = join(BASE_DIR, 'MEMORY', 'LEARNING', category, yearMonth);
  if (!existsSync(learningsDir)) mkdirSync(learningsDir, { recursive: true });
  const filename = `${year}-${month}-${day}-${hours}${minutes}${seconds}_LEARNING_sentiment-rating-${rating}.md`;
  const filepath = join(learningsDir, filename);
  writeFileSync(filepath, `---\ncapture_type: LEARNING\ntimestamp: ${year}-${month}-${day} ${hours}:${minutes}:${seconds} PST\nrating: ${rating}\nsource: implicit\nauto_captured: true\ntags: [sentiment-detected, implicit-rating, improvement-opportunity]\n---\n\n# Implicit Low Rating Captured: ${rating}/10\n\n**Rating:** ${rating}/10\n**Detection Method:** Sentiment Analysis\n\n---\n\n## Context\n\n${detailedContext || 'No context available'}\n\n---\n\n## Improvement Notes\n\nThis response was rated ${rating}/10 by ${getPrincipalName()}. Use this as an improvement opportunity.\n\n---\n`, 'utf-8');
}

// ── Sentiment prompt ──

const PRINCIPAL_NAME = getPrincipal().name;
const ASSISTANT_NAME = getIdentity().name;

const SENTIMENT_SYSTEM_PROMPT = `Analyze ${PRINCIPAL_NAME}'s message for emotional sentiment toward ${ASSISTANT_NAME} (the AI assistant).

OUTPUT FORMAT (JSON only):
{
  "rating": <1-10 or null>,
  "sentiment": "positive" | "negative" | "neutral",
  "confidence": <0.0-1.0>,
  "summary": "<brief explanation, 10 words max>",
  "detailed_context": "<comprehensive analysis for learning, 100-256 words>"
}

RATING SCALE: 1-2=strong frustration, 3-4=mild frustration, 5=neutral, 6-7=satisfaction, 8-9=strong approval, 10=extraordinary enthusiasm.
Return null for rating when: neutral technical questions, simple commands, no emotional indicators, emotion unrelated to ${ASSISTANT_NAME}'s work.`;

// ── Main ──

async function main() {
  try {
    const context = getRecentContext(transcriptPath);
    const userPrompt = context ? `CONTEXT:\n${context}\n\nCURRENT MESSAGE:\n${prompt}` : prompt;

    const result = await inference({
      systemPrompt: SENTIMENT_SYSTEM_PROMPT,
      userPrompt,
      expectJson: true,
      timeout: 12000,
      level: 'fast',
    });

    if (!result.success || !result.parsed) {
      console.error(`[SentimentWorker] Inference failed: ${result.error}`);
      process.exit(0);
    }

    const sentiment = result.parsed as {
      rating: number | null;
      sentiment: string;
      confidence: number;
      summary: string;
      detailed_context: string;
    };

    if (sentiment.rating === null) sentiment.rating = 5;
    if (sentiment.confidence < MIN_CONFIDENCE) {
      console.error(`[SentimentWorker] Confidence ${sentiment.confidence} below threshold, skipping`);
      process.exit(0);
    }

    const entry = {
      timestamp: getISOTimestamp(),
      rating: sentiment.rating,
      session_id: sessionId,
      source: 'implicit' as const,
      sentiment_summary: sentiment.summary,
      confidence: sentiment.confidence,
    };

    writeRating(entry);
    triggerTrending();

    if (sentiment.rating < 6) {
      captureLowRatingLearning(sentiment.rating, sentiment.summary, sentiment.detailed_context || '');

      if (sentiment.rating <= 3) {
        await captureFailure({
          transcriptPath,
          rating: sentiment.rating,
          sentimentSummary: sentiment.summary,
          detailedContext: sentiment.detailed_context || '',
          sessionId,
        }).catch((err) => console.error(`[SentimentWorker] Failure capture error: ${err}`));
      }
    }

    process.exit(0);
  } catch (err) {
    console.error(`[SentimentWorker] Error: ${err}`);
    process.exit(0);
  }
}

main();
