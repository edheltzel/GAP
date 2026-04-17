#!/usr/bin/env bun
/**
 * PreCompact.hook.ts — Algorithm State Preservation (PreCompact)
 *
 * PURPOSE:
 * Injects the tail of the active LOG.md into the compaction summary so that
 * Algorithm state (work in progress, decisions, sub-tasks) survives context
 * compaction. Without this hook the compaction summarizer loses all LOG.md
 * content, breaking the Algorithm's ability to resume mid-session work.
 *
 * Schema source: Claude Code v2.1.105 release notes + claude.ai/docs/hooks
 * (reviewed 2026-04-17). PreCompact payload includes session_id,
 * transcript_path, and trigger ("manual" | "auto"). Fields typed defensively
 * because this lifecycle event is new and schema may evolve.
 *
 * TRIGGER: PreCompact
 * PERFORMANCE: ~5ms (file reads only)
 * NON-BLOCKING: Always exits 0 — state preservation only, never blocks compaction
 *
 * OUTPUT FORMAT:
 * { "hookSpecificOutput": { "hookEventName": "PreCompact", "additionalContext": "..." } }
 */

import { existsSync, readFileSync } from 'fs';
import { paiPath } from './lib/paths';

/** Defensively typed — PreCompact is a new lifecycle event (CC v2.1.105). */
interface HookInput {
  session_id?: string;
  transcript_path?: string;
  /** "manual" = user-triggered compaction, "auto" = automatic threshold compaction */
  trigger?: 'manual' | 'auto' | string;
  [key: string]: unknown;
}

/** Schema written by AutoWorkCreation.hook.ts, read by multiple hooks. */
interface CurrentWork {
  session_id?: string;
  /** Relative subdirectory name under MEMORY/WORK/ (e.g. "20260417-120000_my-task") */
  work_dir?: string;
  /** Older field name — kept for defensive compatibility */
  slug?: string;
  [key: string]: unknown;
}

const LOG_TAIL_CHARS = 3000;

async function readStdin(timeout = 1000): Promise<string> {
  return new Promise((resolve) => {
    let data = '';
    const timer = setTimeout(() => resolve(data), timeout);
    process.stdin.on('data', (chunk) => { data += chunk.toString(); });
    process.stdin.on('end', () => { clearTimeout(timer); resolve(data); });
    process.stdin.on('error', () => { clearTimeout(timer); resolve(''); });
  });
}

async function main(): Promise<void> {
  try {
    const raw = await readStdin();

    let input: HookInput = {};
    if (raw.trim()) {
      try {
        input = JSON.parse(raw) as HookInput;
      } catch {
        process.stderr.write('[PreCompact] Failed to parse stdin JSON — proceeding with no context\n');
      }
    }

    process.stderr.write(`[PreCompact] Fired — trigger=${input.trigger ?? 'unknown'} session=${input.session_id ?? 'unknown'}\n`);

    // --- Step 1: Locate the active work directory ---
    const currentWorkPath = paiPath('MEMORY', 'STATE', 'current-work.json');

    if (!existsSync(currentWorkPath)) {
      process.stderr.write('[PreCompact] No current-work.json — nothing to inject\n');
      process.exit(0);
      return;
    }

    let currentWork: CurrentWork;
    try {
      currentWork = JSON.parse(readFileSync(currentWorkPath, 'utf-8')) as CurrentWork;
    } catch {
      process.stderr.write('[PreCompact] Failed to parse current-work.json — skipping\n');
      process.exit(0);
      return;
    }

    // Resolve the work directory identifier — prefer work_dir, fall back to slug
    const workDirName = currentWork.work_dir || currentWork.slug;

    if (!workDirName) {
      process.stderr.write('[PreCompact] current-work.json has no work_dir or slug — skipping\n');
      process.exit(0);
      return;
    }

    // --- Step 2: Read LOG.md tail ---
    const logPath = paiPath('MEMORY', 'WORK', workDirName, 'LOG.md');

    if (!existsSync(logPath)) {
      process.stderr.write(`[PreCompact] LOG.md not found at ${logPath} — skipping\n`);
      process.exit(0);
      return;
    }

    let logContent: string;
    try {
      logContent = readFileSync(logPath, 'utf-8');
    } catch {
      process.stderr.write(`[PreCompact] Failed to read LOG.md at ${logPath} — skipping\n`);
      process.exit(0);
      return;
    }

    // Extract the last LOG_TAIL_CHARS characters, breaking on a newline boundary
    // to avoid cutting mid-line
    let tail: string;
    if (logContent.length <= LOG_TAIL_CHARS) {
      tail = logContent;
    } else {
      const rawTail = logContent.slice(-LOG_TAIL_CHARS);
      // Advance to the first newline so we don't start mid-line
      const firstNewline = rawTail.indexOf('\n');
      tail = firstNewline !== -1 ? rawTail.slice(firstNewline + 1) : rawTail;
    }

    if (!tail.trim()) {
      process.stderr.write('[PreCompact] LOG.md tail is empty — skipping\n');
      process.exit(0);
      return;
    }

    // --- Step 3: Emit additionalContext ---
    const additionalContext = `## Algorithm State Before Compaction\n\n${tail}`;

    const output = {
      hookSpecificOutput: {
        hookEventName: 'PreCompact',
        additionalContext,
      },
    };

    console.log(JSON.stringify(output));
    process.stderr.write(`[PreCompact] Injected ${tail.length} chars from ${workDirName}/LOG.md\n`);

    process.exit(0);
  } catch {
    // Never block compaction on hook error
    process.stderr.write('[PreCompact] Unexpected error — exiting cleanly to allow compaction\n');
    process.exit(0);
  }
}

main();
