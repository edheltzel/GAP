#!/usr/bin/env bun

import { parseArgs } from "node:util";
import { readdir, readFile, writeFile, mkdir, cp } from "node:fs/promises";
import { join, basename } from "node:path";
import { existsSync } from "node:fs";
import { convertSkill, isValidSkill } from "./lib/skill-converter.ts";
import { convertAgent, isValidAgent, getAgentName } from "./lib/agent-converter.ts";

const CC_SKILLS_DIR = join(import.meta.dir, "../.claude/skills");
const CC_AGENTS_DIR = join(import.meta.dir, "../.claude/agents");
const OC_SKILLS_DIR = expandHome("~/.config/opencode/skills");
const OC_AGENTS_DIR = expandHome("~/.config/opencode/agent");

const SKILLS_TO_SKIP = new Set([
  "PAI",
  "PAIUpgrade",
  "Observability",
  "converting-slash-commands",
  "converting-claude-subagents",
]);

function expandHome(path: string): string {
  if (path.startsWith("~/")) {
    return join(process.env.HOME || "", path.slice(2));
  }
  return path;
}

async function getSkillDirs(): Promise<string[]> {
  const entries = await readdir(CC_SKILLS_DIR, { withFileTypes: true });
  return entries
    .filter((e) => e.isDirectory() && !SKILLS_TO_SKIP.has(e.name))
    .map((e) => e.name)
    .sort();
}

async function getAgentFiles(): Promise<string[]> {
  const entries = await readdir(CC_AGENTS_DIR, { withFileTypes: true });
  return entries
    .filter((e) => e.isFile() && e.name.endsWith(".md"))
    .map((e) => e.name)
    .sort();
}

const TEXT_EXTENSIONS = [".ts", ".js", ".json", ".md", ".hbs", ".yaml", ".yml", ".txt", ".mdc"];

function processFileContent(content: string): string {
  let result = content;
  result = result.replace(/~\/\.claude\//g, "~/.config/opencode/");
  result = result.replace(/\$\{HOME\}\/\.claude\//g, "${HOME}/.config/opencode/");
  result = result.replace(/\$\{process\.env\.HOME\}\/\.claude\//g, "${process.env.HOME}/.config/opencode/");
  result = result.replace(/\.claude\/skills\/PAI\//g, ".config/opencode/skills/PAI/");
  result = result.replace(/\.claude\/skills\//g, ".config/opencode/skills/");
  result = result.replace(/\.claude\/custom-agents/g, ".config/opencode/custom-agents");
  result = result.replace(/\.claude\/settings\.json/g, ".config/opencode/settings.json");
  
  result = result.replace(/curl[^\n]*localhost:8888[^\n]*\\\n/g, "");
  result = result.replace(/curl[^\n]*localhost:8888[^\n]*\n/g, "");
  result = result.replace(/```bash\n\s*curl[^\n]*localhost:8888[\s\S]*?```\n?/g, "");
  
  return result;
}

async function processSubdirectory(src: string, dest: string, dryRun: boolean): Promise<void> {
  if (dryRun) return;

  await mkdir(dest, { recursive: true });
  const entries = await readdir(src, { withFileTypes: true });

  for (const entry of entries) {
    const srcPath = join(src, entry.name);
    const destPath = join(dest, entry.name);

    if (entry.isDirectory()) {
      await processSubdirectory(srcPath, destPath, dryRun);
    } else if (entry.isFile()) {
      const ext = entry.name.substring(entry.name.lastIndexOf("."));
      if (TEXT_EXTENSIONS.includes(ext)) {
        const content = await readFile(srcPath, "utf-8");
        const processed = processFileContent(content);
        await writeFile(destPath, processed);
      } else {
        await Bun.write(destPath, Bun.file(srcPath));
      }
    }
  }
}

async function convertSingleSkill(
  skillName: string,
  dryRun: boolean,
): Promise<{ success: boolean; message: string }> {
  const skillDir = join(CC_SKILLS_DIR, skillName);
  const skillFile = join(skillDir, "SKILL.md");

  if (!existsSync(skillFile)) {
    return { success: false, message: `SKILL.md not found for ${skillName}` };
  }

  const content = await readFile(skillFile, "utf-8");

  if (!isValidSkill(content)) {
    return { success: false, message: `Invalid skill format: ${skillName}` };
  }

  const converted = convertSkill(content);
  if (!converted) {
    return { success: false, message: `Conversion failed: ${skillName}` };
  }

  const destDir = join(OC_SKILLS_DIR, skillName);
  const destFile = join(destDir, "SKILL.md");

  if (dryRun) {
    console.log(`\n[DRY-RUN] Would convert: ${skillName}`);
    console.log(`  Source: ${skillFile}`);
    console.log(`  Dest:   ${destFile}`);
    console.log(`  Preview (first 500 chars of body):\n`);
    const lines = converted.split("\n");
    const bodyStart = lines.findIndex((l) => !l.startsWith("---") && l !== "");
    console.log(lines.slice(bodyStart, bodyStart + 20).join("\n"));
    return { success: true, message: `[DRY-RUN] ${skillName}` };
  }

  await mkdir(destDir, { recursive: true });
  await writeFile(destFile, converted);

  const subdirs = ["Tools", "templates", "schemas", "examples", "Workflows"];
  for (const sub of subdirs) {
    const subPath = join(skillDir, sub);
    if (existsSync(subPath)) {
      const destSub = join(destDir, sub);
      await processSubdirectory(subPath, destSub, dryRun);
      console.log(`  Processed: ${sub}/`);
    }
  }

  const entries = await readdir(skillDir, { withFileTypes: true });
  for (const entry of entries) {
    if (entry.isDirectory() && !subdirs.includes(entry.name)) {
      const subSkillPath = join(skillDir, entry.name, "SKILL.md");
      if (existsSync(subSkillPath)) {
        const destSub = join(destDir, entry.name);
        await mkdir(destSub, { recursive: true });
        const subContent = await readFile(subSkillPath, "utf-8");
        const convertedSub = convertSkill(subContent);
        if (convertedSub) {
          await writeFile(join(destSub, "SKILL.md"), convertedSub);
          console.log(`  Converted sub-skill: ${entry.name}/`);
        }
      }
    }
  }

  for (const entry of entries) {
    if (entry.isFile() && entry.name.endsWith(".md") && entry.name !== "SKILL.md") {
      const srcPath = join(skillDir, entry.name);
      const destPath = join(destDir, entry.name);
      const content = await readFile(srcPath, "utf-8");
      const processed = processFileContent(content);
      await writeFile(destPath, processed);
      console.log(`  Copied context file: ${entry.name}`);
    }
  }

  return { success: true, message: `Converted: ${skillName}` };
}

async function convertSingleAgent(
  agentFile: string,
  dryRun: boolean,
): Promise<{ success: boolean; message: string }> {
  const agentPath = join(CC_AGENTS_DIR, agentFile);

  const content = await readFile(agentPath, "utf-8");

  if (!isValidAgent(content)) {
    return { success: false, message: `Invalid agent format: ${agentFile}` };
  }

  const agentName = getAgentName(content);
  if (!agentName) {
    return { success: false, message: `Could not extract agent name: ${agentFile}` };
  }

  const converted = convertAgent(content);
  if (!converted) {
    return { success: false, message: `Conversion failed: ${agentFile}` };
  }

  const destFile = join(OC_AGENTS_DIR, `${agentName}.md`);

  if (dryRun) {
    console.log(`\n[DRY-RUN] Would convert: ${agentName}`);
    console.log(`  Source: ${agentPath}`);
    console.log(`  Dest:   ${destFile}`);
    console.log(`  Preview (frontmatter + first 20 lines):\n`);
    const lines = converted.split("\n");
    console.log(lines.slice(0, 40).join("\n"));
    return { success: true, message: `[DRY-RUN] ${agentName}` };
  }

  await mkdir(OC_AGENTS_DIR, { recursive: true });
  await writeFile(destFile, converted);

  return { success: true, message: `Converted: ${agentName}` };
}

async function convertAllSkills(dryRun: boolean): Promise<void> {
  const skills = await getSkillDirs();
  console.log(`\nFound ${skills.length} skills to convert\n`);

  let success = 0;
  let failed = 0;

  for (const skill of skills) {
    const result = await convertSingleSkill(skill, dryRun);
    if (result.success) {
      success++;
      console.log(`✓ ${result.message}`);
    } else {
      failed++;
      console.log(`✗ ${result.message}`);
    }
  }

  console.log(`\nSkills: ${success} succeeded, ${failed} failed`);
}

async function convertAllAgents(dryRun: boolean): Promise<void> {
  const agents = await getAgentFiles();
  console.log(`\nFound ${agents.length} agents to convert\n`);

  let success = 0;
  let failed = 0;

  for (const agent of agents) {
    const result = await convertSingleAgent(agent, dryRun);
    if (result.success) {
      success++;
      console.log(`✓ ${result.message}`);
    } else {
      failed++;
      console.log(`✗ ${result.message}`);
    }
  }

  console.log(`\nAgents: ${success} succeeded, ${failed} failed`);
}

async function main() {
  const { values } = parseArgs({
    args: Bun.argv.slice(2),
    options: {
      skills: { type: "boolean", short: "s", default: false },
      agents: { type: "boolean", short: "a", default: false },
      all: { type: "boolean", default: false },
      skill: { type: "string" },
      agent: { type: "string" },
      "dry-run": { type: "boolean", default: false },
      help: { type: "boolean", short: "h", default: false },
    },
    strict: true,
  });

  if (values.help) {
    console.log(`
PAI to OpenCode Converter

Usage:
  bun run convert-pai-to-oc.ts [options]

Options:
  --skills, -s      Convert all skills
  --agents, -a      Convert all agents
  --all             Convert both skills and agents
  --skill <name>    Convert a single skill
  --agent <name>    Convert a single agent (filename)
  --dry-run         Preview changes without writing
  --help, -h        Show this help

Examples:
  bun run convert-pai-to-oc.ts --skill Research --dry-run
  bun run convert-pai-to-oc.ts --agent Engineer.md --dry-run
  bun run convert-pai-to-oc.ts --all
  bun run convert-pai-to-oc.ts --skills --agents
`);
    process.exit(0);
  }

  const doSkills = values.skills || values.all;
  const doAgents = values.agents || values.all;
  const dryRun = values["dry-run"];

  if (dryRun) {
    console.log("\n🔍 DRY-RUN MODE - No files will be written\n");
  }

  if (values.skill) {
    const result = await convertSingleSkill(values.skill, dryRun);
    console.log(result.message);
    process.exit(result.success ? 0 : 1);
  }

  if (values.agent) {
    const result = await convertSingleAgent(values.agent, dryRun);
    console.log(result.message);
    process.exit(result.success ? 0 : 1);
  }

  if (doSkills) {
    console.log("=== Converting Skills ===");
    await convertAllSkills(dryRun);
  }

  if (doAgents) {
    console.log("\n=== Converting Agents ===");
    await convertAllAgents(dryRun);
  }

  if (!doSkills && !doAgents && !values.skill && !values.agent) {
    console.log("No action specified. Use --skills, --agents, --all, or --skill/--agent <name>");
    console.log("Run with --help for usage information.");
    process.exit(1);
  }
}

main().catch(console.error);
