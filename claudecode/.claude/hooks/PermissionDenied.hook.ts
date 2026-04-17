#!/usr/bin/env bun
/**
 * PermissionDenied.hook.ts — Auto-Mode Denial Observer (PermissionDenied)
 *
 * PURPOSE:
 * Logs every tool-call denial issued by Claude Code's auto-mode permission
 * classifier to a JSONL learning dataset. Records are used to tune the
 * permissions.deny ruleset over time — no retry logic yet.
 *
 * Schema source: Claude Code v2.1.89 release notes (PermissionDenied event)
 *
 * TRIGGER: PermissionDenied
 * OUTPUT: Appends one JSONL record to MEMORY/LEARNING/permission-denials.jsonl
 * PERFORMANCE: ~3ms (single sync append after stdin read)
 * NON-BLOCKING: Always exits 0 — never blocks the denial
 * RETRY: Intentionally omitted — observe-only until denial log is reviewed
 */

import { appendFileSync, mkdirSync } from 'fs';
import { dirname } from 'path';
import { paiPath } from './lib/paths';

/** Confirmed schema from Claude Code v2.1.89 PermissionDenied hook event. */
interface HookInput {
  /** Session identifier provided by Claude Code. */
  session_id: string;
  /** The tool whose invocation was denied. */
  tool_name: string;
  /** The input arguments that were passed to the denied tool call. */
  tool_input: Record<string, unknown>;
  /**
   * Human-readable explanation from the classifier for the denial.
   * Defensively optional — may not be present in all versions.
   */
  reason?: string;
}

/** Shape of one record written to the JSONL log. */
interface DenialRecord {
  timestamp: string;
  session_id: string;
  tool: string;
  input_preview: string;
  reason: string | null;
}

async function readStdin(timeout = 1000): Promise<string> {
  return new Promise((resolve) => {
    let data = '';
    const timer = setTimeout(() => resolve(data), timeout);
    process.stdin.on('data', chunk => { data += chunk.toString(); });
    process.stdin.on('end', () => { clearTimeout(timer); resolve(data); });
    process.stdin.on('error', () => { clearTimeout(timer); resolve(''); });
  });
}

async function main() {
  try {
    const raw = await readStdin();
    if (!raw.trim()) process.exit(0);

    let data: HookInput;
    try {
      data = JSON.parse(raw) as HookInput;
    } catch {
      process.stderr.write('[PermissionDenied] Failed to parse stdin JSON\n');
      process.exit(0);
      return;
    }

    const logPath = paiPath('MEMORY', 'LEARNING', 'permission-denials.jsonl');

    // Ensure the directory exists before appending
    mkdirSync(dirname(logPath), { recursive: true });

    const record: DenialRecord = {
      timestamp: new Date().toISOString(),
      session_id: data.session_id ?? 'unknown',
      tool: data.tool_name ?? 'unknown',
      // Truncate to 200 chars to avoid logging huge payloads (e.g. file content)
      input_preview: JSON.stringify(data.tool_input ?? {}).slice(0, 200),
      reason: data.reason ?? null,
    };

    appendFileSync(logPath, JSON.stringify(record) + '\n', { flag: 'a' });

    // No stdout output — hook is observe-only, no system-reminder needed
    // No retry: true — classifier denials are authoritative until log is reviewed
    process.exit(0);
  } catch {
    // Never block on hook error — denial already issued, just log and exit
    process.stderr.write('[PermissionDenied] Hook error — exiting cleanly\n');
    process.exit(0);
  }
}

main();
