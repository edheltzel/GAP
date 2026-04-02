// z-atlas-footer.ts — Eldritch Statusline Extension for pi
// Named z-* so it loads LAST (after custom-footer.ts)

import { existsSync, mkdirSync, readFileSync, statSync, writeFileSync } from "fs";
import { join } from "path";
import { homedir } from "os";

import type { AssistantMessage } from "@mariozechner/pi-ai";
import type { ExtensionAPI } from "@mariozechner/pi-coding-agent";
import { truncateToWidth } from "@mariozechner/pi-tui";

// ─────────────────────────────────────────────────────────────────────────────
// Config
// ─────────────────────────────────────────────────────────────────────────────

const CACHE_DIR = join(homedir(), ".pi", "agent", "memory", "STATE");
const RATE_CACHE_FILE = join(CACHE_DIR, "eldritch-rate-cache.json");
const EXTRA_CACHE_FILE = join(CACHE_DIR, "eldritch-extra-cache.json");
const CACHE_TTL_MS = 300_000;
const MODEL_CACHE_FILE = join(CACHE_DIR, "eldritch-model-cache.txt");

// ─────────────────────────────────────────────────────────────────────────────
// Color helpers (inline ANSI RGB matching Eldritch palette)
// ─────────────────────────────────────────────────────────────────────────────

const C = {
	SLATE_300: "\x1b[38;2;235;250;250m",
	SLATE_500: "\x1b[38;2;112;129;208m",
	SLATE_600: "\x1b[38;2;50;52;73m",
	EMERALD: "\x1b[38;2;55;244;153m",
	ROSE: "\x1b[38;2;241;108;117m",
	LOVECRAFT: "\x1b[38;2;164;140;242m",
	CYAN: "\x1b[38;2;4;209;249m",
	PINK: "\x1b[38;2;242;101;181m",
	ORANGE: "\x1b[38;2;247;198;127m",
	YELLOW: "\x1b[38;2;241;252;121m",
	RESET: "\x1b[0m",
};

const ICON = {
	GAUGE: "\u26a1",
	CACHE: "\uf49b",
	ATLAS: "\u25b2",
	ATLAS_PI: "\u03c0",
};

const CTX_EMPTY = `${C.SLATE_600}\u26c1${C.RESET}`;

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

function ctxBucketColor(i, barWidth) {
	const pct = Math.floor((i / barWidth) * 100);
	if (pct <= 33) { return "\x1b[38;2;55;244;153m"; }
	if (pct <= 66) { return "\x1b[38;2;241;252;121m"; }
	if (pct <= 85) { return "\x1b[38;2;247;198;127m"; }
	return "\x1b[38;2;241;108;117m";
}

function usageColor(pct) {
	if (pct >= 80) { return C.ROSE; }
	if (pct >= 60) { return "\x1b[38;2;251;146;60m"; }
	if (pct >= 40) { return "\x1b[38;2;251;191;36m"; }
	return C.EMERALD;
}

function cacheColor(pct) {
	if (pct >= 80) { return C.EMERALD; }
	if (pct >= 60) { return C.CYAN; }
	if (pct >= 40) { return C.YELLOW; }
	if (pct >= 20) { return C.ORANGE; }
	return C.ROSE;
}

function formatReset(epochSec) {
	if (!epochSec) { return "—"; }
	try {
		const d = new Date(epochSec * 1000);
		const now = new Date();
		const sameDay = d.toDateString() === now.toDateString();
		const timeStr = d.toLocaleTimeString("en-US", { hour: "2-digit", minute: "2-digit", hour12: false });
		if (sameDay) { return timeStr; }
		const dayStr = d.toLocaleDateString("en-US", { weekday: "short" });
		return `${dayStr} ${timeStr}`;
	} catch (e) {
		return "—";
	}
}

function fmtNum(n) {
	if (n < 1000) { return String(n); }
	if (n < 10_000) { return `${(n / 1000).toFixed(1)}k`; }
	if (n < 1_000_000) { return `${Math.round(n / 1000)}k`; }
	return `${(n / 1_000_000).toFixed(1)}M`;
}

function fmtDollars(n) {
	if (n >= 1000) { return `$${(n / 1000).toFixed(0)}K`; }
	return `$${n.toFixed(0)}`;
}

// ─────────────────────────────────────────────────────────────────────────────
// Keychain + Cache helpers
// ─────────────────────────────────────────────────────────────────────────────

async function getOAuthToken() {
	return new Promise((resolve) => {
		const { execSync } = require("child_process");
		try {
			const result = execSync(
				'security find-generic-password -s "Claude Code-credentials" -w 2>/dev/null',
				{ stdio: "pipe" },
			);
			const data = JSON.parse(result.toString());
			resolve(data?.claudeAiOauth?.accessToken ?? null);
		} catch (e) {
			resolve(null);
		}
	});
}

function readCache(path) {
	try {
		if (!existsSync(path)) { return null; }
		if (Date.now() - statSync(path).mtimeMs > CACHE_TTL_MS) { return null; }
		return JSON.parse(readFileSync(path, "utf8"));
	} catch (e) {
		return null;
	}
}

function writeCache(path, data) {
	try {
		mkdirSync(CACHE_DIR, { recursive: true });
		writeFileSync(path, JSON.stringify(data));
	} catch (e) {
		// ignore
	}
}

async function fetchRateLimits() {
	const token = await getOAuthToken();
	if (!token) { return null; }
	try {
		const res = await fetch("https://api.anthropic.com/api/oauth/usage", {
			headers: {
				Authorization: `Bearer ${token}`,
				"Content-Type": "application/json",
				"anthropic-beta": "oauth-2025-04-20",
			},
			signal: AbortSignal.timeout(5000),
		});
		if (!res.ok) { return null; }
		const data = await res.json();
		const rl = data.rate_limits || {};
		const result = {
			rate5h_pct: Math.round(rl.five_hour?.used_percentage ?? 0),
			rate5h_reset: rl.five_hour?.resets_at ?? 0,
			rate7d_pct: Math.round(rl.seven_day?.used_percentage ?? 0),
			rate7d_reset: rl.seven_day?.resets_at ?? 0,
			fetchedAt: Date.now(),
		};
		writeCache(RATE_CACHE_FILE, result);
		return result;
	} catch (e) {
		return null;
	}
}

async function fetchExtraUsage() {
	const token = await getOAuthToken();
	if (!token) { return null; }
	try {
		const res = await fetch("https://api.anthropic.com/api/oauth/usage", {
			headers: {
				Authorization: `Bearer ${token}`,
				"Content-Type": "application/json",
				"anthropic-beta": "oauth-2025-04-20",
			},
			signal: AbortSignal.timeout(5000),
		});
		if (!res.ok) { return null; }
		const data = await res.json();
		const eu = data.extra_usage || {};
		const result = {
			enabled: eu.is_enabled ?? false,
			limit_dollars: (eu.monthly_limit ?? 0) / 100,
			used_dollars: (eu.used_credits ?? 0) / 100,
			fetchedAt: Date.now(),
		};
		writeCache(EXTRA_CACHE_FILE, result);
		return result;
	} catch (e) {
		return null;
	}
}

function getPiVersion() {
	const paths = [
		join(homedir(), ".bun", "install", "global", "node_modules", "@mariozechner", "pi-coding-agent", "package.json"),
		join(homedir(), "Developer", "AI", "pi-mono", "packages", "coding-agent", "package.json"),
	];
	for (const p of paths) {
		try {
			const pkg = JSON.parse(readFileSync(p, "utf8"));
			if (pkg.version) { return pkg.version; }
		} catch (e) {
			// try next
		}
	}
	return "—";
}

function getAtlasVersion() {
	try {
		const s = JSON.parse(readFileSync(
			join(homedir(), "Developer", "atlas-config", "claudecode", ".claude", "settings.json"),
			"utf8"
		));
		return s.pai?.version ?? "?";
	} catch (e) {
		return "?";
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Extension
// ─────────────────────────────────────────────────────────────────────────────

export default function (pi) {
	let cachedRate = readCache(RATE_CACHE_FILE);
	let cachedExtra = readCache(EXTRA_CACHE_FILE);

	// Kick off background fetch
	void fetchRateLimits().then((r) => { if (r) { cachedRate = r; } });
	void fetchExtraUsage().then((e) => { if (e) { cachedExtra = e; } });

	pi.on("session_start", (_event, ctx) => {
		ctx.ui.setFooter((tui, _theme, footerData) => {
			const unsub = footerData.onBranchChange(() => tui.requestRender());
			const timer = setInterval(() => {
				void fetchRateLimits().then((r) => {
					if (r) { cachedRate = r; tui.requestRender(); }
				});
				void fetchExtraUsage().then((e) => {
					if (e) { cachedExtra = e; tui.requestRender(); }
				});
			}, 30_000);

			return {
				dispose() {
					unsub();
					clearInterval(timer);
				},
				invalidate() {},
				render(width) {
					// Token / cost stats
					let input = 0, output = 0, cost = 0, cacheRead = 0, cacheWrite = 0;
					for (const e of ctx.sessionManager.getBranch()) {
						if (e.type === "message" && e.message.role === "assistant") {
							const m = /** @type {AssistantMessage} */ (e.message);
							input += m.usage.input;
							output += m.usage.output;
							cacheRead += m.usage.cacheRead;
							cacheWrite += m.usage.cacheWrite;
							cost += m.usage.cost.total;
						}
					}

					const usage = ctx.getContextUsage();
					const ctxWindow = usage?.contextWindow ?? 0;
					const ctxPct = usage?.percent ?? 0;

					const branch = footerData.getGitBranch();
					const sessionName = ctx.sessionManager.getSessionName();
					const dirName = process.cwd().split("/").pop() ?? process.cwd();

					const model = ctx.model;
					const modelId = model?.id ?? "no-model";
					const modelProvider = (model && model.provider) ? model.provider : "";
					const modelShort = modelId
						.replace(/^claude-/, "")
						.replace(/sonnet/i, "Sonnet")
						.replace(/opus/i, "Opus")
						.replace(/haiku/i, "Haiku");

					try {
						mkdirSync(CACHE_DIR, { recursive: true });
						writeFileSync(MODEL_CACHE_FILE, modelId, "utf8");
					} catch (e) { /* ignore */ }

					const rate = cachedRate;
					const extra = cachedExtra;
					const cacheTotal = cacheRead + cacheWrite;
					const cacheHitPct = cacheTotal > 0 ? Math.round((cacheRead / cacheTotal) * 100) : 0;

					// ── LINE 1 ──────────────────────────────────────────────────
					const p1 = [];
					p1.push(`${C.LOVECRAFT}${ICON.ATLAS} Atlas${C.RESET} ${C.SLATE_300}v${getAtlasVersion()}${C.RESET}`);
					if (modelProvider || modelId) {
						p1.push(`${C.SLATE_600}·${C.RESET} ${C.LOVECRAFT}${modelProvider}${C.RESET} ${C.SLATE_300}${modelShort}${C.RESET}`);
					}
					p1.push(`${C.SLATE_600}·${C.RESET} ${C.ORANGE}${ICON.ATLAS_PI}${C.RESET} ${C.SLATE_300}v${getPiVersion()}${C.RESET} ${C.CYAN}/${dirName}${C.RESET}`);
					if (branch) {
						p1.push(`${C.SLATE_600}·${C.RESET} ${C.PINK}git:${C.RESET}${C.PINK}${branch}${C.RESET}`);
					}
					if (sessionName) {
						p1.push(`${C.SLATE_600}·${C.RESET} ${C.CYAN}${sessionName.toUpperCase()}${C.RESET}`);
					}

					// ── LINE 2 ──────────────────────────────────────────────────
					const p2 = [];

					if (rate) {
						p2.push(
							`${C.ORANGE}${ICON.GAUGE}${C.RESET} ` +
							`${C.SLATE_500}5H:${C.RESET}${usageColor(rate.rate5h_pct)}${rate.rate5h_pct}%${C.RESET} ` +
							`${C.SLATE_500}↻${C.RESET}${C.SLATE_300}${formatReset(rate.rate5h_reset)}${C.RESET}`
						);
						p2.push(
							`${C.SLATE_600}·${C.RESET} ` +
							`${C.SLATE_500}WK:${C.RESET}${usageColor(rate.rate7d_pct)}${rate.rate7d_pct}%${C.RESET} ` +
							`${C.SLATE_500}↻${C.RESET}${C.SLATE_300}${formatReset(rate.rate7d_reset)}${C.RESET}`
						);
					}

					if (input > 0 || output > 0 || cost > 0) {
						const stats = [];
						if (input > 0) { stats.push(`${C.EMERALD}\u2191${C.RESET}${C.SLATE_300}${fmtNum(input)}${C.RESET}`); }
						if (output > 0) { stats.push(`${C.CYAN}\u2193${C.RESET}${C.SLATE_300}${fmtNum(output)}${C.RESET}`); }
						if (cost > 0) { stats.push(`${C.ORANGE}$${cost.toFixed(2)}${C.RESET}`); }
						if (stats.length > 0) {
							p2.push(`${C.SLATE_600}·${C.RESET} ${stats.join(" ")}`);
						}
					}

					if (ctxWindow > 0) {
						const barWidth = 24;
						const pct = Math.min(100, Math.round(ctxPct));
						const filled = Math.round((pct / 100) * barWidth);
						let bar = "";
						for (let i = 0; i < barWidth; i++) {
							if (i < filled) {
								bar += `${ctxBucketColor(i, barWidth)}\u26c1${C.RESET}`;
							} else {
								bar += CTX_EMPTY;
							}
						}
						const barColor = ctxPct > 80 ? C.ROSE : ctxPct > 60 ? C.ORANGE : ctxPct > 40 ? C.YELLOW : C.EMERALD;
						p2.push(
							`${C.SLATE_600}·${C.RESET} ${bar} ${barColor}${pct}%${C.RESET}` +
							`${C.SLATE_300}/${fmtNum(ctxWindow)}${C.RESET}`
						);
					}

					if (cacheTotal > 0) {
						p2.push(
							`${C.SLATE_600}·${C.RESET} ` +
							`${C.LOVECRAFT}${ICON.CACHE}${C.RESET} ${cacheColor(cacheHitPct)}${cacheHitPct}%${C.RESET}`
						);
					}
					if (extra && extra.enabled) {
						p2.push(
							`${C.SLATE_600}·${C.RESET} ` +
							`${C.PINK}$${extra.used_dollars.toFixed(0)}/${fmtDollars(extra.limit_dollars)}${C.RESET}`
						);
					}

					return [
						truncateToWidth(p1.join(" "), width),
						truncateToWidth(p2.join(" "), width),
					];
				},
			};
		});
	});

	pi.on("session_switch", () => {
		void fetchRateLimits().then((r) => { if (r) { cachedRate = r; } });
		void fetchExtraUsage().then((e) => { if (e) { cachedExtra = e; } });
	});
}
