#!/usr/bin/env bun

import { parseArgs } from "node:util";
import { readdir, readFile, writeFile, mkdir } from "node:fs/promises";
import { join, basename } from "node:path";
import { existsSync } from "node:fs";
import { parse, stringify } from "yaml";

const CC_COMMANDS_DIR = join(process.env.HOME || "", ".claude", "commands");
const OC_COMMANDS_DIR = join(process.env.HOME || "", ".config", "opencode", "command");

interface CCCommand {
  description?: string;
  "allowed-tools"?: string;
  "argument-hint"?: string;
  model?: string;
  "disable-model-invocation"?: boolean;
}

interface OCCommand {
  description: string;
  agent?: string;
  model?: string;
  tools?: Record<string, boolean>;
  subtask?: boolean;
}

const TOOL_TO_AGENT: Record<string, string> = {
  "Read": "plan",
  "Read, Grep, Glob": "plan",
  "Read,Grep,Glob": "plan",
  "Read, Write, Edit, Bash": "build",
  "Read,Write,Edit,Bash": "build",
  "Read, Write, Edit": "build",
  "Read,Write,Edit": "build",
};

const MODEL_MAP: Record<string, string> = {
  "claude-3-5-sonnet-20241022": "anthropic/claude-sonnet-4-20250514",
  "claude-3-5-haiku-20241022": "anthropic/claude-haiku-3-5-20241022",
  "claude-sonnet-4-20250514": "anthropic/claude-sonnet-4-20250514",
  "claude-opus-4-20250514": "anthropic/claude-opus-4-20250514",
  "claude-haiku-3-5-20241022": "anthropic/claude-haiku-3-5-20241022",
};

function parseFrontmatter(content: string): { frontmatter: CCCommand | null; body: string } {
  const match = content.match(/^---\n([\s\S]*?)\n---\n([\s\S]*)$/);
  if (!match) {
    return { frontmatter: null, body: content };
  }

  try {
    const frontmatter = parse(match[1]) as CCCommand;
    const body = match[2];
    return { frontmatter, body };
  } catch {
    return { frontmatter: null, body: content };
  }
}

function convertCommand(content: string): string | null {
  const { frontmatter, body } = parseFrontmatter(content);

  if (!frontmatter) {
    console.warn("  No frontmatter found, copying as-is");
    return content;
  }

  const oc: OCCommand = {
    description: frontmatter.description || "No description",
  };

  if (frontmatter["allowed-tools"]) {
    const toolsStr = frontmatter["allowed-tools"];
    oc.agent = TOOL_TO_AGENT[toolsStr] || "build";
  }

  if (frontmatter.model) {
    oc.model = MODEL_MAP[frontmatter.model] || frontmatter.model;
    if (!oc.model.startsWith("anthropic/") && !oc.model.startsWith("openai/") && !oc.model.startsWith("opencode/")) {
      oc.model = `anthropic/${frontmatter.model}`;
    }
  }

  if (frontmatter["argument-hint"]) {
    oc.description = `${oc.description} ${frontmatter["argument-hint"]}`;
  }

  const frontmatterYaml = stringify(oc, { lineWidth: 0 });
  return `---\n${frontmatterYaml}---\n${body.trim()}\n`;
}

async function getCommandFiles(dir: string): Promise<string[]> {
  if (!existsSync(dir)) return [];
  const entries = await readdir(dir, { withFileTypes: true });
  return entries
    .filter((e) => e.isFile() && e.name.endsWith(".md"))
    .map((e) => e.name)
    .sort();
}

async function convertSingleCommand(
  commandFile: string,
  dryRun: boolean
): Promise<{ success: boolean; message: string }> {
  const srcPath = join(CC_COMMANDS_DIR, commandFile);

  if (!existsSync(srcPath)) {
    return { success: false, message: `Command not found: ${commandFile}` };
  }

  const content = await readFile(srcPath, "utf-8");
  const converted = convertCommand(content);

  if (!converted) {
    return { success: false, message: `Conversion failed: ${commandFile}` };
  }

  const destPath = join(OC_COMMANDS_DIR, commandFile);

  if (dryRun) {
    console.log(`\n[DRY-RUN] Would convert: ${commandFile}`);
    console.log(`  Source: ${srcPath}`);
    console.log(`  Dest:   ${destPath}`);
    console.log(`  Preview (frontmatter):\n`);
    const lines = converted.split("\n");
    const frontmatterEnd = lines.findIndex((l) => l === "---" && lines.indexOf(l) > 0);
    console.log(lines.slice(0, frontmatterEnd + 1).join("\n"));
    return { success: true, message: `[DRY-RUN] ${commandFile}` };
  }

  await mkdir(OC_COMMANDS_DIR, { recursive: true });
  await writeFile(destPath, converted);

  return { success: true, message: `Converted: ${commandFile}` };
}

async function convertAllCommands(dryRun: boolean): Promise<void> {
  const commands = await getCommandFiles(CC_COMMANDS_DIR);
  console.log(`\nFound ${commands.length} commands to convert\n`);

  let success = 0;
  let failed = 0;

  for (const cmd of commands) {
    const result = await convertSingleCommand(cmd, dryRun);
    if (result.success) {
      success++;
      console.log(`✓ ${result.message}`);
    } else {
      failed++;
      console.log(`✗ ${result.message}`);
    }
  }

  console.log(`\nCommands: ${success} succeeded, ${failed} failed`);
}

async function main() {
  const { values } = parseArgs({
    args: Bun.argv.slice(2),
    options: {
      "dry-run": { type: "boolean", default: false },
      command: { type: "string" },
      help: { type: "boolean", short: "h", default: false },
    },
    strict: true,
  });

  if (values.help) {
    console.log(`
CC Commands to OC Commands Converter

Usage:
  bun run convert-commands-to-oc.ts [options]

Options:
  --dry-run         Preview changes without writing
  --command <file>  Convert a single command
  --help, -h        Show this help

Examples:
  bun run convert-commands-to-oc.ts --dry-run
  bun run convert-commands-to-oc.ts --command diff-review.md
`);
    process.exit(0);
  }

  const dryRun = values["dry-run"];

  if (dryRun) {
    console.log("\n🔍 DRY-RUN MODE - No files will be written\n");
  }

  if (values.command) {
    const result = await convertSingleCommand(values.command, dryRun);
    console.log(result.message);
    process.exit(result.success ? 0 : 1);
  }

  await convertAllCommands(dryRun);
}

main().catch(console.error);