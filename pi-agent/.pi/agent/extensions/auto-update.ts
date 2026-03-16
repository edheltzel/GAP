/**
 * PI Auto Update — check for new pi-coding-agent version on session start
 */
import type { ExtensionAPI } from "@mariozechner/pi-coding-agent";
import { execSync } from "node:child_process";
import { readFileSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import { homedir } from "node:os";

const PACKAGE_NAME = "@mariozechner/pi-coding-agent";
const CHECK_INTERVAL = 24 * 60 * 60 * 1000; // 24h
const STAMP_FILE = join(homedir(), ".pi", "agent", ".update-check");

function readStamp(): number {
  try { return Number(readFileSync(STAMP_FILE, "utf8").trim()) || 0; } catch { return 0; }
}

function writeStamp() {
  try { writeFileSync(STAMP_FILE, String(Date.now())); } catch {}
}

function getLatestVersion(): string | null {
  try {
    return execSync(`npm view ${PACKAGE_NAME} version`, { encoding: "utf8", timeout: 8000 }).trim();
  } catch { return null; }
}

function getCurrentVersion(): string | null {
  try {
    const out = JSON.parse(execSync(`npm list -g ${PACKAGE_NAME} --json --depth=0`, { encoding: "utf8", timeout: 8000 }));
    return out.dependencies?.[PACKAGE_NAME]?.version ?? null;
  } catch { return null; }
}

export function isNewer(latest: string, current: string): boolean {
  const a = latest.split(".").map(Number);
  const b = current.split(".").map(Number);
  for (let i = 0; i < 3; i++) {
    if ((a[i] ?? 0) > (b[i] ?? 0)) return true;
    if ((a[i] ?? 0) < (b[i] ?? 0)) return false;
  }
  return false;
}

export default function (pi: ExtensionAPI) {
  pi.on("session_start", async (_event, ctx) => {
    setTimeout(async () => {
      try {
        if (Date.now() - readStamp() < CHECK_INTERVAL) return;
        writeStamp();

        const current = getCurrentVersion();
        const latest = getLatestVersion();
        if (!current || !latest || !isNewer(latest, current)) return;

        const msg = `pi ${latest} available (current: ${current}). Run: npm install -g ${PACKAGE_NAME}`;
        if (ctx.hasUI) {
          ctx.ui.notify(msg, "info");
        }
      } catch {}
    }, 2000);
  });
}
