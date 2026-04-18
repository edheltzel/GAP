#!/usr/bin/env bun

/**
 * RebuildPAI.ts - Assembles SKILL.md from Components/
 *
 * Usage: bun ~/.claude/skills/PAI/Tools/RebuildPAI.ts
 *
 * Reads all .md files from Components/, sorts by numeric prefix,
 * concatenates them, and writes to SKILL.md with build timestamp
 */

import { readdirSync, readFileSync, writeFileSync } from "fs";
import { join } from "path";

const HOME = process.env.HOME!;
// PAI_BUILD_ROOT: override to point at a worktree's skill root (e.g. worktree/.claude/skills/PAI).
// Note: PAI_DIR in settings.json means ~/.claude (the .claude root), not the skill root —
// we use a distinct name to avoid a naming clash with that established convention.
const CORE_DIR = process.env.PAI_BUILD_ROOT || join(HOME, ".claude/skills/PAI");
const COMPONENTS_DIR = join(CORE_DIR, "Components");
const ALGORITHM_DIR = join(COMPONENTS_DIR, "Algorithm");
const OUTPUT_FILE = join(CORE_DIR, "SKILL.md");
// SETTINGS_PATH: always resolves to Claude Code's settings.json under $HOME/.claude.
// Do NOT honor PAI_CONFIG_DIR here — that var is set in Ed's shell to ~/.config/PAI
// (where PAI's .env lives, used by tools like YouTubeApi.ts) and has a different meaning
// than Claude Code's settings directory. For worktree builds that need an alternate
// settings source, set PAI_BUILD_SETTINGS explicitly.
const SETTINGS_PATH = process.env.PAI_BUILD_SETTINGS || join(HOME, ".claude", "settings.json");

/**
 * Load identity variables from settings.json for template resolution
 */
function loadVariables(): Record<string, string> {
  try {
    const settings = JSON.parse(readFileSync(SETTINGS_PATH, "utf-8"));
    return {
      "{DAIDENTITY.NAME}": settings.daidentity?.name || "PAI",
      "{DAIDENTITY.FULLNAME}": settings.daidentity?.fullName || "Personal AI",
      "{DAIDENTITY.DISPLAYNAME}": settings.daidentity?.displayName || "PAI",
      "{PRINCIPAL.NAME}": settings.principal?.name || "User",
      "{PRINCIPAL.TIMEZONE}": settings.principal?.timezone || "UTC",
      "{DAIDENTITY.ALGORITHMVOICEID}": settings.daidentity?.algorithmVoiceID || "",
    };
  } catch {
    console.warn("⚠️ Could not read settings.json, using defaults");
    return {
      "{DAIDENTITY.NAME}": "PAI",
      "{DAIDENTITY.FULLNAME}": "Personal AI",
      "{DAIDENTITY.DISPLAYNAME}": "PAI",
      "{PRINCIPAL.NAME}": "User",
      "{PRINCIPAL.TIMEZONE}": "UTC",
      "{DAIDENTITY.ALGORITHMVOICEID}": "",
    };
  }
}

/**
 * Resolve template variables in content
 */
function resolveVariables(content: string, variables: Record<string, string>): string {
  let result = content;
  for (const [key, value] of Object.entries(variables)) {
    result = result.replaceAll(key, value);
  }
  return result;
}

// Generate timestamp in format: DAY MONTH YEAR HOUR MINUTE SECOND
function getTimestamp(): string {
  const now = new Date();
  const day = now.getDate();
  const months = ['January', 'February', 'March', 'April', 'May', 'June',
                  'July', 'August', 'September', 'October', 'November', 'December'];
  const month = months[now.getMonth()];
  const year = now.getFullYear();
  const hour = now.getHours().toString().padStart(2, '0');
  const minute = now.getMinutes().toString().padStart(2, '0');
  const second = now.getSeconds().toString().padStart(2, '0');

  return `${day} ${month} ${year} ${hour}:${minute}:${second}`;
}

// Load versioned algorithm
function loadAlgorithm(): string {
  const latestFile = join(ALGORITHM_DIR, "LATEST");
  const version = readFileSync(latestFile, "utf-8").trim();
  const algorithmFile = join(ALGORITHM_DIR, `${version}.md`);
  return readFileSync(algorithmFile, "utf-8");
}

// Get all .md files, sorted by numeric prefix
const components = readdirSync(COMPONENTS_DIR)
  .filter(f => f.endsWith(".md"))
  .sort((a, b) => {
    const numA = parseInt(a.split("-")[0]) || 0;
    const numB = parseInt(b.split("-")[0]) || 0;
    return numA - numB;
  });

if (components.length === 0) {
  console.error("❌ No component files found in Components/");
  process.exit(1);
}

// Assemble content
let output = "";
const timestamp = getTimestamp();
const algorithmContent = loadAlgorithm();

for (const file of components) {
  let content = readFileSync(join(COMPONENTS_DIR, file), "utf-8");

  // Inject timestamp into frontmatter component
  if (file === "00-frontmatter.md") {
    content = content.replace(
      "  Build:  bun ~/.claude/skills/PAI/Tools/RebuildPAI.ts",
      `  Build:  bun ~/.claude/skills/PAI/Tools/RebuildPAI.ts\n  Built:  ${timestamp}`
    );
  }

  // Inject versioned algorithm
  if (content.includes("{{ALGORITHM_VERSION}}")) {
    content = content.replace("{{ALGORITHM_VERSION}}", algorithmContent);
  }

  output += content;

  // No extra newlines - components manage their own spacing
}

// Resolve template variables from settings.json
const variables = loadVariables();
output = resolveVariables(output, variables);

// Write output
writeFileSync(OUTPUT_FILE, output);

const resolvedCount = Object.entries(variables)
  .filter(([key]) => output.includes(key) === false)
  .length;

console.log(`✅ Built SKILL.md from ${components.length} components:`);
components.forEach((c, i) => {
  console.log(`   ${(i + 1).toString().padStart(2)}. ${c}`);
});
console.log(`\n🔄 Resolved ${Object.keys(variables).length} template variables:`);
for (const [key, value] of Object.entries(variables)) {
  console.log(`   ${key} → ${value}`);
}
console.log(`\n📄 Output: ${OUTPUT_FILE}`);
