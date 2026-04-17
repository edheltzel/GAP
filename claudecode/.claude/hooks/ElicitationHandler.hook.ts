#!/usr/bin/env bun
/**
 * ElicitationHandler.hook.ts — MCP Elicitation Observer (Elicitation)
 *
 * PURPOSE:
 * Logs every MCP elicitation request to a JSONL learning dataset. Records are
 * used to understand what MCP servers are asking for, enabling future
 * pre-fill automation. No auto-response is issued — elicitation falls through
 * to the user.
 *
 * Schema source: MCP spec 2025-06-18 (Elicitation capability — bleeding-edge).
 * Fields defensively typed; use [key: string]: unknown fallback where uncertain.
 *
 * TRIGGER: Elicitation
 * OUTPUT: Appends one JSONL record to MEMORY/LEARNING/mcp-elicitations.jsonl
 * PERFORMANCE: ~3ms (single sync append after stdin read)
 * NON-BLOCKING: Always exits 0 — never blocks the elicitation
 * AUTO-RESPOND: Intentionally omitted — observe-only until log is reviewed
 */

import { appendFileSync, mkdirSync } from 'fs';
import { dirname } from 'path';
import { paiPath } from './lib/paths';

/**
 * Elicitation hook input from Claude Code.
 *
 * MCP spec 2025-06-18 defines server-sent elicitation requests; the exact
 * Claude Code hook envelope is undocumented at time of writing, so both
 * possible field names are handled defensively.
 */
interface HookInput {
  /** Session identifier provided by Claude Code. */
  session_id: string;
  /**
   * Name of the MCP server that triggered elicitation.
   * Claude Code may use either field name — both are handled.
   */
  mcp_server_name?: string;
  server_name?: string;
  /** The MCP tool whose execution triggered the elicitation. */
  tool_name?: string;
  /** The human-readable prompt shown in the elicitation form. */
  message?: string;
  /**
   * JSON Schema describing the elicitation form fields.
   * Typed as unknown because the schema shape varies per tool.
   * May be keyed as either form_schema or schema.
   */
  form_schema?: unknown;
  schema?: unknown;
  /** Catch-all for any additional fields in future spec revisions. */
  [key: string]: unknown;
}

/** Shape of one record written to the JSONL log. */
interface ElicitationRecord {
  timestamp: string;
  session_id: string;
  server: string;
  tool: string;
  message: string;
  schema_fields: string;
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
      process.stderr.write('[ElicitationHandler] Failed to parse stdin JSON\n');
      process.exit(0);
      return;
    }

    const logPath = paiPath('MEMORY', 'LEARNING', 'mcp-elicitations.jsonl');

    // Ensure the directory exists before appending
    mkdirSync(dirname(logPath), { recursive: true });

    // Resolve the form schema — handle both possible field names
    const rawSchema = data.form_schema ?? data.schema;

    // Extract property keys from the JSON Schema if present
    let schemaFields = '[]';
    try {
      const schemaObj = rawSchema as Record<string, unknown> | null | undefined;
      const properties = schemaObj?.properties as Record<string, unknown> | undefined;
      schemaFields = JSON.stringify(Object.keys(properties ?? {}));
    } catch {
      // Schema was present but not parseable — leave as empty array
    }

    const record: ElicitationRecord = {
      timestamp: new Date().toISOString(),
      session_id: (data.session_id as string) ?? 'unknown',
      server: (data.mcp_server_name as string) ?? (data.server_name as string) ?? 'unknown',
      tool: (data.tool_name as string) ?? 'unknown',
      // Truncate to 300 chars to avoid logging huge prompts
      message: String(data.message ?? '').slice(0, 300),
      schema_fields: schemaFields,
    };

    appendFileSync(logPath, JSON.stringify(record) + '\n', { flag: 'a' });

    // No stdout output — hook is observe-only, no system-reminder needed
    // No auto-respond — elicitation falls through to the user
    process.exit(0);
  } catch {
    // Never block on hook error — elicitation already in flight, just exit
    process.stderr.write('[ElicitationHandler] Hook error — exiting cleanly\n');
    process.exit(0);
  }
}

main();
