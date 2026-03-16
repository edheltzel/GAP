/**
 * oh-pi Safe Guard Extension
 *
 * Combines destructive command confirmation + protected paths in one extension.
 *
 * Protected paths trigger a confirmation dialog with 3 options:
 * - Allow: allows this one time
 * - Block: blocks the operation
 * - Always allow for this session: allows this + all future writes to this path for the current session
 *
 * The session allowlist is cleared on session resume.
 */
import type { ExtensionAPI } from "@mariozechner/pi-coding-agent";

export const DANGEROUS_PATTERNS = [
  /\brm\s+(-[a-zA-Z]*f[a-zA-Z]*\s+|.*-rf\b|.*--force\b)/,
  /\bsudo\s+rm\b/,
  /\b(DROP|TRUNCATE|DELETE\s+FROM)\b/i,
  /\bchmod\s+777\b/,
  /\bmkfs\b/,
  /\bdd\s+if=/,
  />\s*\/dev\/sd[a-z]/,
];

export const PROTECTED_PATHS = [".env", ".git/", "node_modules/", ".pi/", "id_rsa", ".ssh/"];

// Session-level allowlist for protected paths
// Cleared on session resume
let sessionAllowedPaths: string[] = [];

function isPathAllowed(path: string): boolean {
  return sessionAllowedPaths.some((allowed) => {
    // Exact match
    if (path === allowed) return true;
    // Path is a subdirectory/subfile of an allowed path
    if (path.startsWith(allowed + "/")) return true;
    // Wildcard match: if allowed path ends with /, it matches anything under it
    if (allowed.endsWith("/") && path.startsWith(allowed)) return true;
    return false;
  });
}

function addToAllowlist(path: string): void {
  // Normalize: add trailing slash for directories to make matching consistent
  const normalized = path.endsWith("/") ? path : path + "/";
  if (!sessionAllowedPaths.some((p) => p === normalized || p.startsWith(normalized))) {
    sessionAllowedPaths.push(normalized);
  }
}

export default function (pi: ExtensionAPI) {
  // Clear session allowlist on session start/resume
  pi.on("session_start", async () => {
    sessionAllowedPaths = [];
  });

  pi.on("tool_call", async (event, ctx) => {
    // Check bash commands for dangerous patterns
    if (event.toolName === "bash") {
      const cmd = (event.input as { command?: string }).command ?? "";
      const match = DANGEROUS_PATTERNS.find((p) => p.test(cmd));
      if (match && ctx.hasUI) {
        const ok = await ctx.ui.confirm("⚠️ Dangerous Command", `Execute: ${cmd}?`);
        if (!ok) return { block: true, reason: "Blocked by user" };
      }
    }

    // Check write/edit for protected paths
    if (event.toolName === "write" || event.toolName === "edit") {
      const path = (event.input as { path?: string }).path ?? "";

      // Check if path matches any protected pattern
      const hit = PROTECTED_PATHS.find((p) => path.includes(p));
      if (!hit) return;

      // Check if already allowed for this session
      if (isPathAllowed(hit)) return;

      // Prompt user with 3 options
      const choice = await ctx.ui.select(`🛡️ Protected Path`, [
        "Allow",
        "Block",
        "Allow for session",
      ]);

      if (choice === "Block" || choice === undefined) {
        return { block: true, reason: `Protected path: ${hit}` };
      }

      if (choice === "Allow for session") {
        addToAllowlist(hit);
        // Allow the operation to proceed
      }
      // "Allow" also allows the operation to proceed
    }
  });
}
