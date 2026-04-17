#!/usr/bin/env bun
/**
 * InstructionsLoaded.hook.ts — Context Load Confirmation (InstructionsLoaded)
 *
 * PURPOSE:
 * Logs when CLAUDE.md or .claude/rules/*.md files are loaded into context.
 * Outputs a system-reminder confirming which instruction files were loaded.
 *
 * Schema source: code.claude.com/docs/en/hooks (reviewed 2026-04-17)
 *
 * TRIGGER: InstructionsLoaded
 * PERFORMANCE: ~2ms (no I/O beyond stdin)
 * NON-BLOCKING: Always exits 0
 */

/** Confirmed schema from Claude Code official hooks documentation (2026-04-17). */
interface HookInput {
  /** Absolute path to the instruction file that was loaded. */
  file_path: string;
  /** Which memory slot the file was loaded into. */
  memory_type: "User" | "Project" | "Local" | "Managed";
  /** Human-readable explanation of why this file was loaded. */
  load_reason: string;
  /** Path glob patterns from the file's frontmatter, if any. */
  globs?: string[];
  /** The file path that triggered this load, if applicable. */
  trigger_file_path?: string;
  /** The parent file that included this one, if applicable. */
  parent_file_path?: string;
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
      data = JSON.parse(raw) as HookInput;
    } catch {
      process.stderr.write('[InstructionsLoaded] Failed to parse stdin JSON\n');
      process.exit(0);
      return;
    }

    // Log the raw event shape to stderr for debugging
    process.stderr.write(`[InstructionsLoaded] Event received: ${JSON.stringify(data)}\n`);

    // file_path is always present per the confirmed schema
    const filePath = data.file_path || null;

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
