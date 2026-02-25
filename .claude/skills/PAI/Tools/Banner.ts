#!/usr/bin/env bun

/**
 * Atlas Banner — Eldritch Theme + Planet Animation
 * Uses asciigen.art planet frames when available, falls back to raycast sphere.
 */

import { readFileSync } from "fs";
import { join } from "path";
import { spawnSync } from "child_process";

const HOME = process.env.HOME!;
const CLAUDE_DIR = join(HOME, ".claude");

// ═══════════════════════════════════════════════════════════════════════════
// Terminal Width Detection
// ═══════════════════════════════════════════════════════════════════════════

function getTerminalWidth(): number {
  let width: number | null = null;

  // WezTerm: query via UNIX socket (works even in subprocesses where /dev/tty is unavailable)
  const weztermPane = process.env.WEZTERM_PANE;
  if (weztermPane) {
    try {
      const result = spawnSync("wezterm", ["cli", "list", "--format", "json"], {
        encoding: "utf-8",
        timeout: 2000,
      });
      if (result.stdout) {
        const panes = JSON.parse(result.stdout) as Array<{ pane_id: number; size: { cols: number } }>;
        const pane = panes.find(p => p.pane_id === parseInt(weztermPane));
        if (pane?.size?.cols > 0) width = pane.size.cols;
      }
    } catch {}
  }

  // Kitty fallback
  if (!width || width <= 0) {
    const kittyWindowId = process.env.KITTY_WINDOW_ID;
    if (kittyWindowId) {
      try {
        const result = spawnSync("kitten", ["@", "ls"], { encoding: "utf-8" });
        if (result.stdout) {
          const data = JSON.parse(result.stdout);
          for (const osWindow of data) {
            for (const tab of osWindow.tabs) {
              for (const win of tab.windows) {
                if (win.id === parseInt(kittyWindowId)) { width = win.columns; break; }
              }
            }
          }
        }
      } catch {}
    }
  }

  if (!width || width <= 0) {
    try {
      const result = spawnSync("sh", ["-c", "stty size </dev/tty 2>/dev/null"], { encoding: "utf-8" });
      if (result.stdout) {
        const cols = parseInt(result.stdout.trim().split(/\s+/)[1]);
        if (cols > 0) width = cols;
      }
    } catch {}
  }

  if (!width || width <= 0) {
    try {
      const result = spawnSync("tput", ["cols"], { encoding: "utf-8" });
      if (result.stdout) {
        const cols = parseInt(result.stdout.trim());
        if (cols > 0) width = cols;
      }
    } catch {}
  }

  if (!width || width <= 0) {
    width = parseInt(process.env.COLUMNS || "100") || 100;
  }

  return width;
}

// ═══════════════════════════════════════════════════════════════════════════
// ANSI Helpers
// ═══════════════════════════════════════════════════════════════════════════

const RESET  = "\x1b[0m";
const BOLD   = "\x1b[1m";
const ITALIC = "\x1b[3m";

const rgb = (r: number, g: number, b: number) => `\x1b[38;2;${r};${g};${b}m`;

// Eldritch color palette
const E = {
  cyan:   rgb(4,   209, 249),   // #04d1f9 — primary accent
  green:  rgb(55,  244, 153),   // #37f499 — secondary accent
  purple: rgb(164, 140, 242),   // #a48cf2 — tertiary accent
  pink:   rgb(242, 101, 181),   // #f265b5 — pustule pink (hooks)
  yellow: rgb(241, 252, 121),   // #f1fc79 — gold of yuggoth (signals)
  red:    rgb(241, 108, 117),   // #f16c75
  orange: rgb(247, 198, 127),   // #f7c67f
  fg:     rgb(197, 205, 217),   // #c5cdd9 foreground
  dim:    rgb(120, 130, 155),   // muted text
  muted:  rgb(65,  72,  95),    // very muted (borders, separators)
  mid:    rgb(80,  185, 248),   // purple→cyan blend
};

// Nerd Font icons — matching atlas-statusline.sh
const NF = {
  skills:   "\u{F00D7}", // nf-md-briefcase_check
  workflow: "\u{F04E6}", // nf-md-sync
  hook:     "\u{F06E2}", // nf-md-hook
  wifi:     "\u{F05A9}", // nf-md-wifi (signals)
};

const SPARK = ["\u2581", "\u2582", "\u2583", "\u2584", "\u2585", "\u2586", "\u2587", "\u2588"];

const BOX = {
  tl: "\u256d", tr: "\u256e", bl: "\u2570", br: "\u256f",
  h: "\u2500", v: "\u2502",
};
const RETICLE = {
  tl: "\u250F", tr: "\u2513", bl: "\u2517", br: "\u251B", h: "\u2501",
};

// ═══════════════════════════════════════════════════════════════════════════
// Stats Collection
// ═══════════════════════════════════════════════════════════════════════════

interface SystemStats {
  name: string;
  catchphrase: string;
  skills: number;
  workflows: number;
  hooks: number;
  learnings: number;
  userFiles: number;
  paiVersion: string;
  algorithmVersion: string;
}

function getStats(): SystemStats {
  let name = "Atlas";
  let paiVersion = "3.0";
  let algorithmVersion = "1.2";
  let catchphrase = "Ready to go";

  try {
    const settings = JSON.parse(readFileSync(join(CLAUDE_DIR, "settings.json"), "utf-8"));
    name = settings.daidentity?.displayName || settings.daidentity?.name || name;
    paiVersion = settings.pai?.version || paiVersion;
    catchphrase = settings.daidentity?.startupCatchphrase || catchphrase;
  } catch {}

  catchphrase = catchphrase.replace(/\{name\}/gi, name);

  try {
    const latestPath = join(CLAUDE_DIR, "skills/PAI/Components/Algorithm/LATEST");
    algorithmVersion = readFileSync(latestPath, "utf-8").trim().replace(/^v/i, "");
  } catch {}

  let skills = 0, workflows = 0, hooks = 0, learnings = 0, userFiles = 0;

  try {
    const settings = JSON.parse(readFileSync(join(CLAUDE_DIR, "settings.json"), "utf-8"));
    if (settings.counts) {
      skills    = settings.counts.skills    || 0;
      workflows = settings.counts.workflows || 0;
      hooks     = settings.counts.hooks     || 0;
      learnings = settings.counts.signals   || 0;
      userFiles = settings.counts.files     || 0;
    }
  } catch {
    skills = 65; workflows = 339; hooks = 18; learnings = 3000; userFiles = 172;
  }

  return { name, catchphrase, skills, workflows, hooks, learnings, userFiles, paiVersion, algorithmVersion };
}

// ═══════════════════════════════════════════════════════════════════════════
// Utility Functions
// ═══════════════════════════════════════════════════════════════════════════

function visibleLength(str: string): number {
  return str.replace(/\x1b\[[0-9;]*m/g, "").length;
}

function padEnd(str: string, width: number): string {
  return str + " ".repeat(Math.max(0, width - visibleLength(str)));
}

function center(str: string, width: number): string {
  const visible = visibleLength(str);
  const left = Math.floor((width - visible) / 2);
  return " ".repeat(Math.max(0, left)) + str + " ".repeat(Math.max(0, width - visible - left));
}

function sparkline(length: number, colors: string[]): string {
  return Array.from({ length }, (_, i) => {
    const level = Math.floor(Math.random() * 8);
    return `${colors[i % colors.length]}${SPARK[level]}${RESET}`;
  }).join("");
}

function fmtNum(n: number): string {
  if (n >= 10000) return `${Math.floor(n / 1000)}k`;
  if (n >= 1000)  return `${Math.floor(n / 100) / 10}k`;
  return String(n);
}

// Label padded to 10 visible chars for aligned columns
const lbl = (text: string) => `${E.dim}${text.padEnd(10)}${RESET}`;

// ═══════════════════════════════════════════════════════════════════════════
// Sphere Renderer (fallback)
// ═══════════════════════════════════════════════════════════════════════════

const SPHERE_ROWS = 9;
const SPHERE_COLS = 20;

// Light direction (upper-left-front, normalized)
const LX = -0.50, LY = -0.35, LZ = 0.79;

function renderSphere(frame: number): string[] {
  const cx = (SPHERE_COLS - 1) / 2;
  const cy = (SPHERE_ROWS - 1) / 2;
  const xr = SPHERE_COLS / 2;
  const yr = SPHERE_ROWS / 2;
  const rot = frame * 0.12;

  const output: string[] = [];

  for (let y = 0; y < SPHERE_ROWS; y++) {
    let line = "";
    for (let x = 0; x < SPHERE_COLS; x++) {
      const nx = (x - cx) / xr;
      const ny = (y - cy) / yr;
      const r2 = nx * nx + ny * ny;

      if (r2 > 1) {
        line += " ";
        continue;
      }

      const nz = Math.sqrt(1 - r2);
      const diffuse = Math.max(0, nx * LX + ny * LY + nz * LZ);
      const lon = Math.atan2(nx, nz) + rot;
      const lat = Math.asin(Math.max(-1, Math.min(1, ny)));
      const texVal = Math.sin(lon * 3) * Math.cos(lat * 2.5);
      const isLand = texVal > 0.18;
      const brightness = 0.06 + diffuse * 0.82 + (isLand ? 0.12 : 0);

      let char: string;
      let color: string;

      if (brightness < 0.13) {
        char = "·"; color = E.muted;
      } else if (brightness < 0.28) {
        char = ":"; color = E.purple;
      } else if (brightness < 0.44) {
        char = "+"; color = isLand ? rgb(90, 190, 130) : rgb(120, 140, 248);
      } else if (brightness < 0.60) {
        char = "*"; color = isLand ? E.green : E.mid;
      } else if (brightness < 0.78) {
        char = "#"; color = isLand ? E.green : E.cyan;
      } else {
        char = "@"; color = E.cyan;
      }

      line += `${color}${char}${RESET}`;
    }
    output.push(line);
  }

  return output;
}

// ═══════════════════════════════════════════════════════════════════════════
// Banner Designs
// ═══════════════════════════════════════════════════════════════════════════

// Full banner with sphere: two-column layout
function createSphereBanner(stats: SystemStats, width: number, sphere: string[]): string {
  const SEP = `${E.muted}${BOX.v}${RESET}`;

  const infoLines = [
    `${ITALIC}${E.dim}"${RESET}${ITALIC}${E.cyan}${stats.catchphrase}${RESET}${ITALIC}${E.dim}..."${RESET}`,
    `${E.muted}${BOX.h.repeat(26)}${RESET}`,
    `${E.purple}⚙${RESET}  ${lbl("Algo")}${E.fg}v${stats.algorithmVersion}${RESET}`,
    `${E.cyan}${NF.skills}${RESET}  ${lbl("Skills")}${E.fg}${stats.skills}${RESET}`,
    `${E.green}${NF.workflow}${RESET}  ${lbl("Work")}${E.fg}${stats.workflows}${RESET}`,
    `${E.pink}${NF.hook}${RESET}  ${lbl("Hooks")}${E.fg}${stats.hooks}${RESET}`,
    `${E.yellow}${NF.wifi}${RESET}  ${lbl("Signals")}${E.fg}${stats.learnings}${RESET}`,
    `${E.green}≡${RESET}  ${lbl("Files")}${E.fg}${stats.userFiles}${RESET}`,
    `${E.muted}${BOX.h.repeat(26)}${RESET}`,
  ];

  const gap = "   ";
  const gapAfter = "  ";
  const totalContentWidth = SPHERE_COLS + gap.length + 1 + gapAfter.length + 32;
  const leftPad = Math.floor((width - totalContentWidth) / 2);
  const pad = " ".repeat(Math.max(2, leftPad));
  const empty = " ".repeat(SPHERE_COLS);

  const logoTopPad = Math.floor((infoLines.length - sphere.length) / 2);

  const frameWidth = 70;
  const framePad = " ".repeat(Math.max(0, Math.floor((width - frameWidth) / 2)));

  const lines: string[] = [""];
  lines.push(`${framePad}${E.muted}${RETICLE.tl}${RETICLE.h.repeat(frameWidth - 2)}${RETICLE.tr}${RESET}`);
  lines.push("");

  const header = `${E.purple}▲${RESET}  ${BOLD}${E.cyan}${stats.name}${RESET}  ${E.dim}v${stats.paiVersion}${RESET}`;
  lines.push(center(header, width));
  lines.push("");

  for (let i = 0; i < infoLines.length; i++) {
    const li = i - logoTopPad;
    const sphereRow = (li >= 0 && li < sphere.length) ? sphere[li] : empty;
    lines.push(`${pad}${padEnd(sphereRow, SPHERE_COLS)}${gap}${SEP}${gapAfter}${infoLines[i]}`);
  }

  lines.push("");
  lines.push(`${pad}${sparkline(SPHERE_COLS, [E.purple, E.mid, E.cyan, E.green])}`);
  lines.push("");
  lines.push(`${framePad}${E.muted}${RETICLE.bl}${RETICLE.h.repeat(frameWidth - 2)}${RETICLE.br}${RESET}`);
  lines.push("");

  return lines.join("\n");
}

// Medium banner (70-84 cols) — info block centered, no animation
function createMediumBanner(stats: SystemStats, width: number): string {
  const SEP_LEN = 26;

  const infoLines = [
    `${ITALIC}${E.dim}"${RESET}${ITALIC}${E.cyan}${stats.catchphrase}${RESET}${ITALIC}${E.dim}..."${RESET}`,
    `${E.muted}${BOX.h.repeat(SEP_LEN)}${RESET}`,
    `${E.purple}⚙${RESET}  ${lbl("Algo")}${E.fg}v${stats.algorithmVersion}${RESET}`,
    `${E.cyan}${NF.skills}${RESET}  ${lbl("Skills")}${E.fg}${stats.skills}${RESET}`,
    `${E.green}${NF.workflow}${RESET}  ${lbl("Work")}${E.fg}${stats.workflows}${RESET}`,
    `${E.pink}${NF.hook}${RESET}  ${lbl("Hooks")}${E.fg}${stats.hooks}${RESET}`,
    `${E.yellow}${NF.wifi}${RESET}  ${lbl("Signals")}${E.fg}${stats.learnings}${RESET}`,
    `${E.green}≡${RESET}  ${lbl("Files")}${E.fg}${stats.userFiles}${RESET}`,
    `${E.muted}${BOX.h.repeat(SEP_LEN)}${RESET}`,
  ];

  const pad = " ".repeat(Math.max(0, Math.floor((width - SEP_LEN) / 2)));
  const header = `${E.purple}▲${RESET}  ${BOLD}${E.cyan}${stats.name}${RESET}  ${E.dim}v${stats.paiVersion}${RESET}`;

  const lines: string[] = [""];
  lines.push(center(header, width));
  lines.push("");
  for (const line of infoLines) lines.push(`${pad}${line}`);
  lines.push("");
  return lines.join("\n");
}

// Compact banner (55-69 cols)
function createCompactBanner(stats: SystemStats, width: number): string {
  const SEP_LEN = 22;
  const shortPhrase = stats.catchphrase.length > 22
    ? stats.catchphrase.slice(0, 19) + "..."
    : stats.catchphrase;

  const infoLines = [
    `${ITALIC}${E.dim}"${RESET}${ITALIC}${E.cyan}${shortPhrase}${RESET}${ITALIC}${E.dim}"${RESET}`,
    `${E.muted}${BOX.h.repeat(SEP_LEN)}${RESET}`,
    `${E.purple}⚙${RESET} ${E.dim}Algo${RESET} ${E.fg}v${stats.algorithmVersion}${RESET}`,
    `${E.cyan}${NF.skills}${RESET} ${E.dim}Skills${RESET}   ${E.fg}${stats.skills}${RESET}  ${E.green}${NF.workflow}${RESET} ${E.dim}Work${RESET} ${E.fg}${stats.workflows}${RESET}`,
    `${E.pink}${NF.hook}${RESET} ${E.dim}Hooks${RESET}    ${E.fg}${stats.hooks}${RESET}  ${E.yellow}${NF.wifi}${RESET} ${E.dim}Signals${RESET} ${E.fg}${stats.learnings}${RESET}`,
    `${E.muted}${BOX.h.repeat(SEP_LEN)}${RESET}`,
  ];

  const pad = " ".repeat(Math.max(1, Math.floor((width - SEP_LEN) / 2)));
  const header = `${E.purple}▲${RESET}  ${BOLD}${E.cyan}${stats.name}${RESET}  ${E.dim}v${stats.paiVersion}${RESET}`;

  const lines: string[] = [""];
  lines.push(center(header, width));
  lines.push("");
  for (const line of infoLines) lines.push(`${pad}${line}`);
  lines.push("");
  return lines.join("\n");
}

// Minimal banner (45-54 cols)
function createMinimalBanner(stats: SystemStats, width: number): string {
  const header = `${E.purple}▲${RESET}  ${BOLD}${E.cyan}${stats.name}${RESET}  ${E.dim}v${stats.paiVersion}${RESET}`;

  const lines: string[] = [""];
  lines.push(center(header, width));
  lines.push(center(`${E.purple}⚙${RESET}${E.dim}Algo v${stats.algorithmVersion}${RESET}`, width));
  lines.push(center(`${E.muted}${BOX.h.repeat(16)}${RESET}`, width));
  lines.push(center(
    `${E.cyan}${NF.skills}${RESET}${E.fg}${stats.skills}${RESET} ${E.green}${NF.workflow}${RESET}${E.fg}${stats.workflows}${RESET} ${E.pink}${NF.hook}${RESET}${E.fg}${stats.hooks}${RESET}`,
    width,
  ));
  lines.push("");
  return lines.join("\n");
}

// Ultra-compact banner (<45 cols)
function createUltraCompactBanner(stats: SystemStats, width: number): string {
  const header = `${E.purple}▲${RESET}  ${BOLD}${E.cyan}${stats.name}${RESET}  ${E.dim}v${stats.paiVersion}${RESET}`;

  const lines: string[] = [""];
  lines.push(center(header, width));
  lines.push(center(`${E.purple}⚙${RESET}${E.dim}v${stats.algorithmVersion}${RESET}`, width));
  lines.push(center(`${E.muted}${BOX.h.repeat(Math.min(16, width - 4))}${RESET}`, width));
  lines.push(center(
    `${E.cyan}${NF.skills}${RESET}${E.fg}${stats.skills}${RESET} ${E.green}${NF.workflow}${RESET}${E.fg}${stats.workflows}${RESET} ${E.pink}${NF.hook}${RESET}${E.fg}${stats.hooks}${RESET}`,
    width,
  ));
  lines.push("");
  return lines.join("\n");
}

// Route non-full banners
function createStaticBanner(stats: SystemStats, width: number, design?: string): string {
  const w = width;
  if (design === "medium"  || (!design && w >= 70 && w < 85))  return createMediumBanner(stats, w);
  if (design === "compact" || (!design && w >= 55 && w < 70))  return createCompactBanner(stats, w);
  if (design === "minimal" || (!design && w >= 45 && w < 55))  return createMinimalBanner(stats, w);
  if (design === "ultra"   || (!design && w < 45))             return createUltraCompactBanner(stats, w);
  return createMediumBanner(stats, w);
}

// ═══════════════════════════════════════════════════════════════════════════
// Animation
// ═══════════════════════════════════════════════════════════════════════════

const ANIM_FPS = 15;

async function runAnimation(
  stats: SystemStats,
  width: number,
): Promise<void> {
  process.on("SIGINT", () => {
    process.stdout.write("\x1b[?25h\n");
    process.exit(0);
  });

  process.stdout.write("\x1b[?25l"); // hide cursor

  const frameCount = 48;
  let prevLines = 0;

  try {
    for (let f = 0; f < frameCount; f++) {
      const banner = createSphereBanner(stats, width, renderSphere(f));

      if (f > 0) {
        process.stdout.write(`\x1b[${prevLines}A\x1b[J`);
      }

      process.stdout.write(banner);
      prevLines = (banner.match(/\n/g) ?? []).length;

      if (f < frameCount - 1) {
        await new Promise(resolve => setTimeout(resolve, Math.floor(1000 / ANIM_FPS)));
      }
    }
  } finally {
    process.stdout.write("\x1b[?25h"); // restore cursor
  }
}

// ═══════════════════════════════════════════════════════════════════════════
// CLI
// ═══════════════════════════════════════════════════════════════════════════

const BREAKPOINTS = { FULL: 85, MEDIUM: 70, COMPACT: 55, MINIMAL: 45 };
const ALL_DESIGNS = ["full", "medium", "compact", "minimal", "ultra"] as const;

const args       = process.argv.slice(2);
const testMode   = args.includes("--test");
const staticMode = args.includes("--static");
const designArg  = args.find(a => a.startsWith("--design="))?.split("=")[1];

const stats = getStats();
const width = getTerminalWidth();

if (testMode) {
  for (const design of ALL_DESIGNS) {
    console.log(`\n${"═".repeat(60)}\n  DESIGN: ${design.toUpperCase()}\n${"═".repeat(60)}`);
    if (design === "full") {
      process.stdout.write(createSphereBanner(stats, width, renderSphere(20)));
    } else {
      process.stdout.write(createStaticBanner(stats, width, design));
    }
  }
} else if (staticMode || designArg) {
  if (designArg === "full" || (!designArg && width >= BREAKPOINTS.FULL)) {
    process.stdout.write(createSphereBanner(stats, width, renderSphere(20)));
  } else {
    process.stdout.write(createStaticBanner(stats, width, designArg));
  }
} else {
  if (width >= BREAKPOINTS.FULL) {
    await runAnimation(stats, width);
  } else {
    process.stdout.write(createStaticBanner(stats, width));
  }
}
