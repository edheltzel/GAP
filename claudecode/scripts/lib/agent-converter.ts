import { parseFrontmatter, stringifyFrontmatter } from "./parse-yaml.ts";

export interface CCAgent {
  name: string;
  description: string;
  model?: string;
  color?: string;
  maxTurns?: number;
  voiceId?: string;
  voice?: Record<string, unknown>;
  persona?: {
    name: string;
    title: string;
    background: string;
  };
  permissions?: {
    allow: string[];
  };
}

export interface OCAgent {
  description: string;
  mode: "subagent" | "primary" | "all";
  model?: string;
  color?: string;
  steps?: number;
  tools: Record<string, boolean>;
  permission?: {
    edit?: "ask" | "allow" | "deny";
    bash?: Record<string, "ask" | "allow" | "deny">;
    webfetch?: "ask" | "allow" | "deny";
  };
  hidden?: boolean;
}

const MODEL_MAP: Record<string, string> = {
  opus: "anthropic/claude-opus-4-20250514",
  sonnet: "anthropic/claude-sonnet-4-20250514",
  haiku: "anthropic/claude-haiku-3-5-20241022",
};

const COLOR_MAP: Record<string, string> = {
  blue: "#3B82F6",
  green: "#22C55E",
  red: "#EF4444",
  yellow: "#EAB308",
  purple: "#A855F7",
  orange: "#F97316",
  cyan: "#06B6D4",
  pink: "#EC4899",
  white: "#FFFFFF",
  gray: "#6B7280",
};

const TOOL_MAP: Record<string, string> = {
  Bash: "bash",
  "Read(*)": "read",
  "Write(*)": "write",
  "Edit(*)": "edit",
  "MultiEdit(*)": "edit",
  "Grep(*)": "grep",
  "Glob(*)": "glob",
  "WebFetch(domain:*)": "webfetch",
  "WebFetch(*)": "webfetch",
  "TodoWrite(*)": "todowrite",
  "Task(*)": "task",
  "Skill(*)": "skill",
  "mcp__*": "mcp_*",
  "List(*)": "list",
};

const VOICE_NOTIFICATION_PATTERN =
  /```bash\ncurl -X POST http:\/\/localhost:8888\/notify[\s\S]*?```\n\n?/g;

const VOICE_SECTION_PATTERN =
  /## 🎯 MANDATORY VOICE NOTIFICATION[\s\S]*?---\n\n?/g;

const STARTUP_VOICE_PATTERN =
  /## 🚨 MANDATORY STARTUP SEQUENCE[\s\S]*?```bash\ncurl[\s\S]*?```\n\n?/g;

const CLAUDE_PATH_PATTERN = /~\/\.claude\//g;

export function convertAgent(content: string): string | null {
  const parsed = parseFrontmatter<CCAgent>(content);
  if (!parsed) {
    console.error("Failed to parse agent frontmatter");
    return null;
  }

  const { frontmatter, body } = parsed;

  const oc: OCAgent = {
    description: frontmatter.description,
    mode: "subagent",
    tools: {},
  };

  if (frontmatter.model && MODEL_MAP[frontmatter.model]) {
    oc.model = MODEL_MAP[frontmatter.model];
  } else if (frontmatter.model) {
    oc.model = frontmatter.model;
  }

  if (frontmatter.color) {
    oc.color = COLOR_MAP[frontmatter.color] || frontmatter.color;
  }

  if (frontmatter.maxTurns) {
    oc.steps = frontmatter.maxTurns;
  }

  if (frontmatter.permissions?.allow) {
    for (const perm of frontmatter.permissions.allow) {
      let tool = TOOL_MAP[perm];
      if (!tool) {
        const basePerm = perm.replace(/\(.*\)/, "(*)");
        tool = TOOL_MAP[basePerm];
      }
      if (tool) {
        oc.tools[tool] = true;
      } else {
        console.warn(`  Unknown permission: ${perm}`);
      }
    }
  }

  if (oc.tools.edit) {
    oc.permission = oc.permission || {};
    oc.permission.edit = "ask";
  }
  if (oc.tools.bash) {
    oc.permission = oc.permission || {};
    oc.permission.bash = { "*": "ask" };
  }
  if (oc.tools.webfetch) {
    oc.permission = oc.permission || {};
    oc.permission.webfetch = "ask";
  }

  let convertedBody = body;

  convertedBody = convertedBody.replace(VOICE_NOTIFICATION_PATTERN, "");
  convertedBody = convertedBody.replace(VOICE_SECTION_PATTERN, "");
  convertedBody = convertedBody.replace(STARTUP_VOICE_PATTERN, "");
  convertedBody = convertedBody.replace(CLAUDE_PATH_PATTERN, "~/.config/opencode/");

  convertedBody = cleanupEmptySections(convertedBody);

  return stringifyFrontmatter(oc as Record<string, unknown>, convertedBody.trim() + "\n");
}

function cleanupEmptySections(body: string): string {
  return body
    .replace(/\n{3,}/g, "\n\n")
    .replace(/^##\s*\n/gm, "")
    .replace(/##\s+$/gm, "");
}

export function getAgentName(content: string): string | null {
  const parsed = parseFrontmatter<CCAgent>(content);
  return parsed?.frontmatter.name || null;
}

export function isValidAgent(content: string): boolean {
  const parsed = parseFrontmatter<CCAgent>(content);
  if (!parsed) return false;
  return !!(parsed.frontmatter.name && parsed.frontmatter.description);
}
