import type { ExtensionAPI } from "@oh-my-pi/pi-coding-agent";
import { VERSION } from "@oh-my-pi/pi-coding-agent";
import { SEGMENTS } from "@oh-my-pi/pi-coding-agent/modes/components/status-line/segments";
import { theme, type ThemeColor } from "@oh-my-pi/pi-coding-agent/modes/theme/theme";

let installed = false;

function contextColor(pct: number): ThemeColor {
  if (pct >= 80) return "error";
  if (pct >= 55) return "warning";
  return "success";
}

function bucketColor(bucketPct: number): ThemeColor {
  if (bucketPct >= 80) return "error";
  if (bucketPct >= 55) return "warning";
  if (bucketPct >= 35) return "statusLineContext";
  return "success";
}

function formatCompactNumber(value: number): string {
  const rounded = Math.max(0, Math.round(value));
  if (rounded >= 1_000_000) return `${Math.round(rounded / 100_000) / 10}M`;
  if (rounded >= 1_000) return `${Math.round(rounded / 100) / 10}K`;
  return `${rounded}`;
}

function unusedStatusColor(text: string): string {
  return theme.fg("borderMuted", text);
}



function renderContextBar(pct: number): string {
  const width = 16;
  const clamped = Math.max(0, Math.min(100, Math.round(pct)));
  const filled = Math.round((clamped * width) / 100);
  let out = "";

  for (let i = 1; i <= width; i += 1) {
    if (i <= filled) {
      out += theme.fg(bucketColor((i * 100) / width), "⛁");
    } else {
      out += theme.fg("borderMuted", "⛁");
    }
  }

  return out;
}

export default function atlasStatusline(_pi: ExtensionAPI) {
  if (installed) return;
  installed = true;

  SEGMENTS.pi = {
    id: "pi",
    render() {
      const icon = theme.icon.pi || "π";
      return { content: theme.fg("accent", `${icon} ${VERSION}`), visible: true };
    },
  };

  SEGMENTS.context_pct = {
    id: "context_pct",
    render(ctx) {
      const pct = Number.isFinite(ctx.contextPercent) ? ctx.contextPercent : 0;
      const rounded = Math.max(0, Math.min(100, Math.round(pct)));
      const bar = renderContextBar(rounded);
      const pctText = theme.fg(contextColor(rounded), `${rounded}%`);
      const contextTotal = ctx.contextWindow > 0 ? unusedStatusColor(`/${formatCompactNumber(ctx.contextWindow)}`) : "";
      return { content: `${bar} ${pctText}${contextTotal}`, visible: true };
    },
  };

  SEGMENTS.cache_hit = {
    id: "cache_hit",
    render(ctx) {
      const { cacheRead, cacheWrite, input } = ctx.usageStats;

      const total = cacheRead + cacheWrite + input;
      if (total <= 0) return { content: "", visible: false };

      const rate = Math.round((cacheRead / total) * 100);
      const icon = theme.icon.cache ? `${theme.icon.cache} ` : "";
      const cachedTokens = formatCompactNumber(cacheRead);
      const promptTotal = unusedStatusColor(`/${formatCompactNumber(total)}`);
      return { content: theme.fg("statusLineSpend", `${icon}${rate}% ${cachedTokens}${promptTotal}`), visible: true };
    },
  };
}
