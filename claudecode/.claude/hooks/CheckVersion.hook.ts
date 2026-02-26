#!/usr/bin/env bun
/**
 * CheckVersion.hook.ts - Check for Claude Code Updates (SessionStart async)
 *
 * PURPOSE:
 * Compares the installed Claude Code version against the latest available on npm.
 * If an update is available, displays a notification to stderr. This keeps the
 * system current without interrupting workflow.
 *
 * TRIGGER: SessionStart (async: true — does NOT block session startup)
 *
 * INPUT:
 * - None (reads version info from CLI and npm)
 *
 * OUTPUT:
 * - stdout: None
 * - stderr: Update notification if newer version available
 * - exit(0): Always (non-blocking)
 *
 * SIDE EFFECTS:
 * - Network request to npm registry (at most once per 24h, cached)
 * - Spawns one child process for current version check
 * - Writes MEMORY/STATE/version-check-cache.json with 24h TTL
 *
 * PERFORMANCE:
 * - Async: Yes (runs in background, never blocks startup)
 * - npm call: at most once per 24h (cached in MEMORY/STATE/version-check-cache.json)
 * - Skipped for subagents: Yes
 */

import { existsSync, readFileSync, writeFileSync, mkdirSync } from 'fs';
import { join } from 'path';

const CLAUDE_DIR = join(process.env.HOME!, '.claude');
const CACHE_PATH = join(CLAUDE_DIR, 'MEMORY', 'STATE', 'version-check-cache.json');
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
    const proc = Bun.spawn(['claude', '--version'], {
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

  // Cache miss or stale — fetch from npm
  try {
    const proc = Bun.spawn(['npm', 'view', '@anthropic-ai/claude-code', 'version'], {
      stdout: 'pipe',
      stderr: 'pipe'
    });
    const output = await new Response(proc.stdout).text();
    const version = output.trim() || 'unknown';
    if (version !== 'unknown') writeCache(version);
    return version;
  } catch {
    return 'unknown';
  }
}

async function main() {
  try {
    // Skip for subagents
    const claudeProjectDir = process.env.CLAUDE_PROJECT_DIR || '';
    const isSubagent = claudeProjectDir.includes('/.claude/Agents/') ||
                      process.env.CLAUDE_AGENT_TYPE !== undefined;

    if (isSubagent) {
      process.exit(0);
    }

    const [currentVersion, latestVersion] = await Promise.all([
      getCurrentVersion(),
      getLatestVersion()
    ]);

    if (currentVersion !== 'unknown' && latestVersion !== 'unknown' && currentVersion !== latestVersion) {
      console.error(`💡 Update available: CC ${currentVersion} → ${latestVersion}`);
    }

    process.exit(0);
  } catch {
    process.exit(0);
  }
}

main();
