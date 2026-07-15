import { readFileSync } from "node:fs";
import { join, parse, sep } from "node:path";
import { VERSION, type ExtensionAPI, type ExtensionContext } from "@earendil-works/pi-coding-agent";
import { truncateToWidth, visibleWidth } from "@earendil-works/pi-tui";

const SYMBOLS = {
	atlas: "▲",
	thinking: "⚡︎", // U+FE0E keeps text presentation instead of emoji
	pi: "π",
	contextCell: "⛁",
} as const;

export const NERD_FONT_ICONS = {
	branch: "\uF418", // nf-oct-git_branch
	cache: "\uF49B", // nf-oct-cache
} as const;

type Theme = ExtensionContext["ui"]["theme"];
type Rgb = readonly [number, number, number];

type CacheUsage = {
	input: number;
	cacheRead: number;
	cacheWrite: number;
};

type StatusPalette = {
	brand: Rgb;
	text: Rgb;
	muted: Rgb;
	separator: Rgb;
	path: Rgb;
	branch: Rgb;
	pi: Rgb;
	contextEmpty: Rgb;
	cacheIcon: Rgb;
	gradient: readonly [Rgb, Rgb, Rgb, Rgb];
	usage: readonly [Rgb, Rgb, Rgb, Rgb];
	cache: readonly [Rgb, Rgb, Rgb, Rgb, Rgb];
};

const DARK_PALETTE: StatusPalette = {
	brand: [164, 140, 242],
	text: [235, 250, 250],
	muted: [112, 129, 208],
	separator: [50, 52, 73],
	path: [4, 209, 249],
	branch: [242, 101, 181],
	pi: [232, 97, 32],
	contextEmpty: [50, 52, 73],
	cacheIcon: [164, 140, 242],
	gradient: [
		[55, 244, 153],
		[241, 252, 121],
		[247, 198, 127],
		[241, 108, 117],
	],
	usage: [
		[55, 244, 153],
		[251, 191, 36],
		[251, 146, 60],
		[241, 108, 117],
	],
	cache: [
		[55, 244, 153],
		[4, 209, 249],
		[241, 252, 121],
		[247, 198, 127],
		[241, 108, 117],
	],
};

const LIGHT_PALETTE: StatusPalette = {
	brand: [144, 122, 169],
	text: [87, 82, 121],
	muted: [121, 117, 147],
	separator: [223, 218, 217],
	path: [86, 148, 159],
	branch: [215, 130, 126],
	pi: [234, 157, 52],
	contextEmpty: [223, 218, 217],
	cacheIcon: [144, 122, 169],
	gradient: [
		[40, 105, 131],
		[234, 157, 52],
		[215, 130, 126],
		[180, 99, 122],
	],
	usage: [
		[40, 105, 131],
		[234, 157, 52],
		[215, 130, 126],
		[180, 99, 122],
	],
	cache: [
		[40, 105, 131],
		[86, 148, 159],
		[234, 157, 52],
		[215, 130, 126],
		[180, 99, 122],
	],
};

function sanitizeLabel(text: string): string {
	return text.replace(/[\r\n\t]/g, " ").replace(/\p{Cc}/gu, "").replace(/ +/g, " ").trim();
}

function sanitizeStatusText(text: string): string {
	return text.replace(/[\r\n\t]/g, " ").replace(/ +/g, " ").trim();
}

function color(rgb: Rgb, text: string): string {
	return `\x1b[38;2;${rgb[0]};${rgb[1]};${rgb[2]}m${text}\x1b[39m`;
}

function mix(from: Rgb, to: Rgb, amount: number): Rgb {
	const clamped = Math.max(0, Math.min(1, amount));
	return [
		Math.round(from[0] + (to[0] - from[0]) * clamped),
		Math.round(from[1] + (to[1] - from[1]) * clamped),
		Math.round(from[2] + (to[2] - from[2]) * clamped),
	];
}

function paletteFor(theme: Theme): StatusPalette {
	const themeName = theme.name?.toLowerCase() || "";
	if (/light|dawn|latte/.test(themeName)) return LIGHT_PALETTE;

	try {
		const activeTheme = readFileSync(join(process.env.HOME || "", ".config", "theme-switcher", "current"), "utf8")
			.trim()
			.toLowerCase();
		if (activeTheme === "rose-pine-dawn" || activeTheme === "catppuccin-latte") return LIGHT_PALETTE;
	} catch {
		// Fall through to the canonical dark palette.
	}

	return DARK_PALETTE;
}

function readAtlasVersion(): string {
	const atlasDir = process.env.ATLAS_DIR || join(process.env.HOME || "", ".claude");
	try {
		return sanitizeLabel(readFileSync(join(atlasDir, "LIFEOS", "VERSION"), "utf8")) || "—";
	} catch {
		return "—";
	}
}

export function formatCurrentDirectory(cwd: string): string {
	const safeCwd = sanitizeLabel(cwd);
	const { root } = parse(safeCwd);
	const parts = safeCwd.slice(root.length).split(sep).filter(Boolean);

	if (parts.length === 0) return root || sep;

	const tail = parts.slice(-2).join(sep);
	if (root || parts.length > 2) return `…${sep}${tail}`;
	return tail;
}

export function calculateCacheHitPercent(usage: CacheUsage): number | null {
	const promptTokens = usage.input + usage.cacheRead + usage.cacheWrite;
	if (!Number.isFinite(promptTokens) || promptTokens <= 0) return null;

	const percent = (usage.cacheRead / promptTokens) * 100;
	return Math.max(0, Math.min(100, Math.round(percent)));
}

export function contextBarWidth(terminalWidth: number): number {
	if (terminalWidth < 35) return 6;
	if (terminalWidth < 55) return 8;
	if (terminalWidth < 80) return 16;
	return 24;
}

export function contextFilledCells(percent: number, width: number): number {
	const clamped = Math.max(0, Math.min(100, Math.floor(percent)));
	return Math.max(0, Math.min(width, Math.floor((clamped * width) / 100)));
}

export function layoutStatusItems(
	items: string[],
	width: number,
	separator = " · ",
	ellipsis = "…",
): string[] {
	if (width <= 0 || items.length === 0) return [];

	const lines: string[] = [];
	let line = "";

	for (const item of items) {
		const fitted = truncateToWidth(item, width, ellipsis);
		const candidate = line ? `${line}${separator}${fitted}` : fitted;
		if (!line || visibleWidth(candidate) <= width) {
			line = candidate;
		} else {
			lines.push(line);
			line = fitted;
		}
	}

	if (line) lines.push(line);
	return lines;
}

function bucketColor(position: number, width: number, palette: StatusPalette): Rgb {
	const percent = (position * 100) / width;
	const [low, middle, warning, high] = palette.gradient;
	if (percent <= 33) return mix(low, middle, percent / 33);
	if (percent <= 66) return mix(middle, warning, (percent - 33) / 33);
	return mix(warning, high, (percent - 66) / 34);
}

function renderContextBar(percent: number, width: number, palette: StatusPalette): string {
	const filled = contextFilledCells(percent, width);
	let output = "";
	for (let position = 1; position <= width; position += 1) {
		const bucket = position <= filled ? bucketColor(position, width, palette) : palette.contextEmpty;
		output += color(bucket, SYMBOLS.contextCell);
	}
	return output;
}

function usageColor(percent: number, palette: StatusPalette): Rgb {
	if (percent >= 80) return palette.usage[3];
	if (percent >= 60) return palette.usage[2];
	if (percent >= 40) return palette.usage[1];
	return palette.usage[0];
}

function cacheColor(percent: number, palette: StatusPalette): Rgb {
	if (percent >= 80) return palette.cache[0];
	if (percent >= 60) return palette.cache[1];
	if (percent >= 40) return palette.cache[2];
	if (percent >= 20) return palette.cache[3];
	return palette.cache[4];
}

function latestCacheHitPercent(ctx: ExtensionContext): number | null {
	const entries = ctx.sessionManager.getBranch();
	for (let index = entries.length - 1; index >= 0; index -= 1) {
		const entry = entries[index];
		if (entry.type !== "message" || entry.message.role !== "assistant") continue;
		if (entry.message.stopReason === "aborted" || entry.message.stopReason === "error") continue;

		const percent = calculateCacheHitPercent(entry.message.usage);
		if (percent !== null) return percent;
	}
	return null;
}

function buildStatusItems(
	pi: ExtensionAPI,
	ctx: ExtensionContext,
	palette: StatusPalette,
	branch: string | null,
	atlasVersion: string,
	terminalWidth: number,
): string[] {
	const model = ctx.model;
	const provider = sanitizeLabel(model?.provider || "no-provider");
	const modelId = sanitizeLabel(model?.id || "no-model");
	const thinking = sanitizeLabel(pi.getThinkingLevel());
	const cwd = formatCurrentDirectory(ctx.sessionManager.getCwd());
	const gitBranch = sanitizeLabel(branch || "no-git");
	const usage = ctx.getContextUsage();
	const contextPercent = Math.max(0, Math.min(100, Math.floor(usage?.percent ?? 0)));
	const cachePercent = latestCacheHitPercent(ctx);
	const separator = color(palette.separator, "·");

	const providerModel = `${color(palette.muted, `${provider}/`)}${color(palette.brand, modelId)}`;
	const modelText = [providerModel, color(palette.path, SYMBOLS.thinking), color(palette.brand, thinking)].join(" ");
	const contextText = `${renderContextBar(contextPercent, contextBarWidth(terminalWidth), palette)} ${color(
		usageColor(contextPercent, palette),
		`${contextPercent}%`,
	)}`;

	const items = [
		`${color(palette.brand, `${SYMBOLS.atlas} Atlas`)} ${color(palette.text, `v${atlasVersion}`)}`,
		modelText,
		`${color(palette.pi, SYMBOLS.pi)} ${color(palette.text, VERSION)}`,
		color(palette.path, cwd),
		`${color(palette.branch, NERD_FONT_ICONS.branch)} ${color(palette.branch, gitBranch)}`,
		contextText,
	];

	if (cachePercent !== null) {
		items.push(
			`${color(palette.cacheIcon, NERD_FONT_ICONS.cache)} ${color(
				cacheColor(cachePercent, palette),
				`${cachePercent}%`,
			)}`,
		);
	}

	return layoutStatusItems(items, terminalWidth, ` ${separator} `, color(palette.muted, "…"));
}

export default function atlasStatusLine(pi: ExtensionAPI) {
	const atlasVersion = readAtlasVersion();
	let requestRender: (() => void) | undefined;
	const refresh = () => requestRender?.();

	pi.on("turn_start", refresh);
	pi.on("message_end", refresh);
	pi.on("agent_settled", refresh);
	pi.on("session_compact", refresh);
	pi.on("model_select", refresh);
	pi.on("thinking_level_select", refresh);

	pi.on("session_start", (_event, ctx) => {
		if (ctx.mode !== "tui") return;

		ctx.ui.setFooter((tui, theme, footerData) => {
			let active = true;
			let palette = paletteFor(theme);
			const renderNow = () => {
				if (active) tui.requestRender();
			};
			requestRender = renderNow;
			const unsubscribeBranch = footerData.onBranchChange(renderNow);

			return {
				dispose() {
					active = false;
					unsubscribeBranch();
					if (requestRender === renderNow) requestRender = undefined;
				},
				invalidate() {
					palette = paletteFor(theme);
				},
				render(width: number): string[] {
					const lines = buildStatusItems(
						pi,
						ctx,
						palette,
						footerData.getGitBranch(),
						atlasVersion,
						width,
					);

					const statuses = Array.from(footerData.getExtensionStatuses().entries())
						.sort(([left], [right]) => left.localeCompare(right))
						.map(([, text]) => sanitizeStatusText(text));
					if (statuses.length > 0) {
						lines.push(truncateToWidth(statuses.join(" "), width, color(palette.muted, "…")));
					}
					return lines;
				},
			};
		});
	});

	pi.on("session_shutdown", () => {
		requestRender = undefined;
	});
}
