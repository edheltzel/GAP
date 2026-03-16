/**
 * Usage Tracker Extension
 * 
 * Logs skill and extension usage to inform promotion decisions.
 * Data saved to ~/.pi/agent/usage-stats.json
 */

import * as fs from "node:fs";
import * as path from "node:path";
import type { ExtensionAPI } from "@mariozechner/pi-coding-agent";

const USAGE_FILE = path.join(process.env.HOME!, ".pi/agent/usage-stats.json");

interface UsageStats {
  skills: Record<string, number>;
  extensions: Record<string, number>;
  lastUpdated: string;
  sessions: number;
}

function loadStats(): UsageStats {
  try {
    return JSON.parse(fs.readFileSync(USAGE_FILE, "utf-8"));
  } catch {
    return { 
      skills: {}, 
      extensions: {}, 
      lastUpdated: new Date().toISOString(),
      sessions: 0 
    };
  }
}

function saveStats(stats: UsageStats) {
  fs.writeFileSync(USAGE_FILE, JSON.stringify(stats, null, 2));
}

export default function (pi: ExtensionAPI) {
  // Track tool calls that include skill hints
  pi.on("tool_call", async (event, ctx) => {
    const stats = loadStats();
    
    // Track by tool name patterns
    const toolName = event.toolName;
    
    // Map tool names to skills
    const skillMapping: Record<string, string> = {
      "web_search": "web-search",
      "web_fetch": "web-fetch",
      "browser_navigate": "browser-tools",
      "browser_click": "browser-tools",
      "context7_query": "context7"
    };
    
    if (skillMapping[toolName]) {
      const skill = skillMapping[toolName];
      stats.skills[skill] = (stats.skills[skill] || 0) + 1;
      stats.lastUpdated = new Date().toISOString();
      saveStats(stats);
    }
  });
  
  // Track session start
  pi.on("session_start", async () => {
    const stats = loadStats();
    stats.sessions = (stats.sessions || 0) + 1;
    stats.lastUpdated = new Date().toISOString();
    saveStats(stats);
  });
}