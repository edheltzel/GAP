#!/usr/bin/env bun
/**
 * InstructionsLoaded.hook.ts — Context Load Confirmation (InstructionsLoaded)
 *
 * PURPOSE:
 * Logs when CLAUDE.md or .claude/rules/*.md files are loaded into context.
 * Outputs a system-reminder confirming which instruction files were loaded.
 *
 * NOTE: This event is from Claude Code changelog (v2.1.69) but not yet in
 * official docs. The input schema is inferred — the handler is defensive
 * against unexpected shapes.
 *
 * TRIGGER: InstructionsLoaded
 * PERFORMANCE: ~2ms (no I/O beyond stdin)
 * NON-BLOCKING: Always exits 0
 */

interface HookInput {
  session_id?: string;
  agent_id?: string;
  agent_type?: string;
  /** Path(s) or identifier(s) of loaded instruction files — schema unknown */
  [key: string]: unknown;
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
    if (!raw) process.exit(0);

    let data: HookInput;
    try {
      data = JSON.parse(raw);
    } catch {
      process.stderr.write('[InstructionsLoaded] Failed to parse stdin JSON\n');
      process.exit(0);
    }

    // Log the raw event shape to stderr for debugging (schema is undocumented)
    process.stderr.write(`[InstructionsLoaded] Event received: ${JSON.stringify(data)}\n`);

    // Extract file path if present — try common field names
    const filePath = data.file_path || data.path || data.file || data.source || null;

    if (filePath) {
      console.log(`<system-reminder>
InstructionsLoaded: ${filePath} loaded into context.
</system-reminder>`);
    } else {
      // No recognizable file path — still confirm the event fired
      console.log(`<system-reminder>
InstructionsLoaded: instruction files loaded into context.
</system-reminder>`);
    }

    process.exit(0);
  } catch {
    // Never block on hook error
    process.exit(0);
  }
}

main();
