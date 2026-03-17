/**
 * CheckVersion.ts - Check for OpenCode Updates (OpenCode Plugin)
 *
 * PURPOSE:
 * Compares the installed OpenCode version against the latest available on GitHub.
 * If an update is available, displays a notification to stderr. This keeps the
 * system current without interrupting workflow.
 *
 * TRIGGER: session.created (runs once per session startup)
 *
 * PORTED FROM: CheckVersion.hook.ts (Claude Code)
 *
 * BEHAVIOR:
 * - Non-blocking: Always succeeds, never blocks session startup
 * - Cached: GitHub API call at most once per 24h
 * - Subagent-aware: Skips for subagent sessions
 */

import { existsSync, readFileSync, writeFileSync, mkdirSync } from 'fs';
import { join } from 'path';
import { homedir } from 'os';

const CLAUDE_DIR = join(homedir(), '.claude');
const CACHE_PATH = join(CLAUDE_DIR, 'MEMORY', 'STATE', 'opencode-version-cache.json');
const CACHE_TTL_MS = 24 * 60 * 60 * 1000; // 24 hours

interface VersionCache {
  latestVersion: string;
  checkedAt: number;
}

function readCache(): VersionCache | null {
  try {
    if (!existsSync(CACHE_PATH)) return null;
    const data = JSON.parse(readFileSync(CACHE_PATH, 'utf-8'));
    if (typeof data.latestVersion === 'string' && typeof data.checkedAt === 'number') {
      return data as VersionCache;
    }
  } catch { /* corrupt cache — ignore */ }
  return null;
}

function writeCache(latestVersion: string): void {
  try {
    const stateDir = join(CLAUDE_DIR, 'MEMORY', 'STATE');
    if (!existsSync(stateDir)) mkdirSync(stateDir, { recursive: true });
    writeFileSync(CACHE_PATH, JSON.stringify({ latestVersion, checkedAt: Date.now() }, null, 2));
  } catch { /* non-fatal */ }
}

async function getCurrentVersion(): Promise<string> {
  try {
    const proc = Bun.spawn(['opencode', '--version'], {
      stdout: 'pipe',
      stderr: 'pipe'
    });
    const output = await new Response(proc.stdout).text();
    const match = output.match(/(\d+\.\d+\.\d+)/);
    return match ? match[1] : 'unknown';
  } catch {
    return 'unknown';
  }
}

async function getLatestVersion(): Promise<string> {
  // Use cache if fresh (< 24h)
  const cache = readCache();
  if (cache && Date.now() - cache.checkedAt < CACHE_TTL_MS) {
    return cache.latestVersion;
  }

  // Cache miss or stale — fetch from GitHub releases
  try {
    const response = await fetch('https://api.github.com/repos/opencode-ai/opencode/releases/latest', {
      headers: {
        'Accept': 'application/vnd.github.v3+json',
        'User-Agent': 'opencode-check-version'
      }
    });
    
    if (!response.ok) return 'unknown';
    
    const data = await response.json() as { tag_name?: string };
    const version = data.tag_name?.replace(/^v/, '') || 'unknown';
    
    if (version !== 'unknown') writeCache(version);
    return version;
  } catch {
    return 'unknown';
  }
}

export default async function CheckVersion({
  event,
}: {
  event: string;
}): Promise<void> {
  if (event !== 'session.created') return;
  
  try {
    // Skip for subagents
    const isSubagent = process.env.OPENCODE_SUBAGENT !== undefined || 
                       process.env.OPENCODE_AGENT !== undefined;
    
    if (isSubagent) return;
    
    const [currentVersion, latestVersion] = await Promise.all([
      getCurrentVersion(),
      getLatestVersion()
    ]);

    if (currentVersion !== 'unknown' && 
        latestVersion !== 'unknown' && 
        currentVersion !== latestVersion) {
      console.error(`[CheckVersion] Update available: OpenCode ${currentVersion} → ${latestVersion}`);
    }
  } catch {
    // Non-fatal - always succeed
  }
}