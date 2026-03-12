import { parseFrontmatter, stringifyFrontmatter } from "./parse-yaml.ts";

export interface CCSkill {
  name: string;
  description: string;
  [key: string]: unknown;
}

export interface OCSkill {
  name: string;
  description: string;
  [key: string]: unknown;
}

const CURL_8888_PATTERN =
  /```bash\n\s*curl[^\n]*localhost:8888[^\n]*(\n[^\n]*\\\n[^\n]*)*[\s\S]*?```\n?/g;

const VOICE_NOTIFICATION_HEADER =
  /## 🚨 MANDATORY: Voice Notification[\s\S]*?(?=\n# )/g;

const MANDATORY_VOICE_SECTION =
  /## 🎯 MANDATORY VOICE NOTIFICATION[\s\S]*?(?=\n# )/g;

const VOICE_INSTRUCTION_BLOCK =
  /\*\*You MUST send this notification BEFORE doing anything else\*\*[\s\S]*?```bash\n\s*curl[^\n]*localhost:8888[\s\S]*?```\n\s*```\n[^\n]*```\n\n\*\*This is not optional[\s\S]*?\n\n/g;

const CUSTOMIZATION_PATH_PATTERN = /~\/\.claude\/skills\/PAI\/USER\/SKILLCUSTOMIZATIONS\//g;

const CLAUDE_PATH_PATTERN = /~\/\.claude\//g;

export function convertSkill(content: string): string | null {
  const parsed = parseFrontmatter<CCSkill>(content);
  if (!parsed) {
    console.error("Failed to parse skill frontmatter");
    return null;
  }

  const { frontmatter, body } = parsed;

  let convertedBody = body;

  convertedBody = convertedBody.replace(CURL_8888_PATTERN, "");
  convertedBody = convertedBody.replace(VOICE_NOTIFICATION_HEADER, "");
  convertedBody = convertedBody.replace(MANDATORY_VOICE_SECTION, "");
  convertedBody = convertedBody.replace(VOICE_INSTRUCTION_BLOCK, "");

  convertedBody = convertedBody.replace(CUSTOMIZATION_PATH_PATTERN, "~/.config/opencode/skills/PAI/USER/SKILLCUSTOMIZATIONS/");
  convertedBody = convertedBody.replace(CLAUDE_PATH_PATTERN, "~/.config/opencode/");

  convertedBody = cleanupEmptySections(convertedBody);

  const ocFrontmatter: OCSkill = {
    name: frontmatter.name,
    description: frontmatter.description,
  };

  for (const [key, value] of Object.entries(frontmatter)) {
    if (key !== "name" && key !== "description") {
      if (!key.startsWith("voice") && !key.includes("Voice")) {
        ocFrontmatter[key] = value;
      }
    }
  }

  return stringifyFrontmatter(ocFrontmatter, convertedBody.trim() + "\n");
}

function cleanupEmptySections(body: string): string {
  return body
    .replace(/\n{3,}/g, "\n\n")
    .replace(/^##\s*\n/gm, "")
    .replace(/##\s+$/gm, "");
}

export function isValidSkill(content: string): boolean {
  const parsed = parseFrontmatter<CCSkill>(content);
  if (!parsed) return false;
  return !!(parsed.frontmatter.name && parsed.frontmatter.description);
}
