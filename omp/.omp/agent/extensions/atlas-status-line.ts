import { homedir } from 'node:os';
import { join, parse, sep } from 'node:path';
import { getLastAssistantUsage, type SessionEntry } from '@oh-my-pi/pi-agent-core/compaction';
import { type ExtensionAPI, VERSION } from '@oh-my-pi/pi-coding-agent';
import { Ellipsis, truncateToWidth, visibleWidth } from '@oh-my-pi/pi-tui';

const SYMBOLS = {
  atlas: '▲',
  thinking: '⚡︎', // U+FE0E forces text presentation.
  omp: 'π',
  contextCell: '⛁',
} as const;

export const NERD_FONT_ICONS = {
  branch: '\uF418', // nf-oct-git_branch
  cache: '\uF49B', // nf-oct-cache
} as const;

type Rgb = readonly [number, number, number];
type CacheUsage = { input: number; cacheRead: number; cacheWrite: number };

type StatusPalette = {
  brand: Rgb;
  text: Rgb;
  muted: Rgb;
  separator: Rgb;
  path: Rgb;
  branch: Rgb;
  omp: Rgb;
  contextEmpty: Rgb;
  cacheIcon: Rgb;
  gradient: readonly [Rgb, Rgb, Rgb, Rgb];
  usage: readonly [Rgb, Rgb, Rgb, Rgb];
  cache: readonly [Rgb, Rgb, Rgb, Rgb, Rgb];
};

export type AtlasFooterState = {
  atlasVersion: string;
  provider: string;
  model: string;
  thinking: string;
  ompVersion: string;
  cwd: string;
  branch: string | null;
  contextPercent: number;
  cachePercent: number | null;
  isLight: boolean;
  statuses?: ReadonlyMap<string, string>;
};

const DARK_PALETTE: StatusPalette = {
  brand: [164, 140, 242],
  text: [235, 250, 250],
  muted: [112, 129, 208],
  separator: [50, 52, 73],
  path: [4, 209, 249],
  branch: [242, 101, 181],
  omp: [232, 97, 32],
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
  omp: [234, 157, 52],
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

export function sanitizeLabel(text: string): string {
  return Bun.stripANSI(text)
    .replace(/\p{Cc}/gu, ' ')
    .replace(/ +/g, ' ')
    .trim();
}

export function sanitizeStatusText(text: string): string {
  return text
    .replace(/\p{Cc}/gu, character => {
      if (character === '\x1b') return character;
      return character === '\r' || character === '\n' || character === '\t' ? ' ' : '';
    })
    .replace(/ +/g, ' ')
    .trim();
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

async function readAtlasVersion(): Promise<string> {
  try {
    return (
      sanitizeLabel(await Bun.file(join(homedir(), '.claude', 'LIFEOS', 'VERSION')).text()) || '—'
    );
  } catch {
    return '—';
  }
}

export function formatCurrentDirectory(cwd: string): string {
  const safeCwd = sanitizeLabel(cwd);
  const { root } = parse(safeCwd);
  const parts = safeCwd.slice(root.length).split(sep).filter(Boolean);
  if (parts.length === 0) return root || sep;

  const tail = parts.slice(-2).join(sep);
  return root || parts.length > 2 ? `…${sep}${tail}` : tail;
}

export function calculateCacheHitPercent(usage: CacheUsage): number | null {
  const { input, cacheRead, cacheWrite } = usage;
  if (![input, cacheRead, cacheWrite].every(value => Number.isFinite(value) && value >= 0))
    return null;
  if (cacheRead + cacheWrite <= 0) return null;

  const promptTokens = input + cacheRead + cacheWrite;
  if (promptTokens <= 0) return null;
  return Math.max(0, Math.min(100, Math.round((cacheRead / promptTokens) * 100)));
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
  separator = ' · ',
  ellipsis: Ellipsis | '' | null = Ellipsis.Unicode
): string[] {
  if (width <= 0 || items.length === 0) return [];

  const lines: string[] = [];
  let line = '';
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
  let output = '';
  for (let position = 1; position <= width; position += 1) {
    if (position <= filled) {
      output += color(bucketColor(position, width, palette), SYMBOLS.contextCell);
    } else {
      output += `\x1b[2m${color(palette.contextEmpty, SYMBOLS.contextCell)}\x1b[22m`;
    }
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

export function latestCacheHitPercent(entries: SessionEntry[]): number | null {
  const usage = getLastAssistantUsage(entries);
  return usage ? calculateCacheHitPercent(usage) : null;
}

export function renderAtlasFooter(state: AtlasFooterState, terminalWidth: number): string[] {
  const palette = state.isLight ? LIGHT_PALETTE : DARK_PALETTE;
  const width = Math.max(0, Math.floor(terminalWidth));
  const contextPercent = Math.max(0, Math.min(100, Math.floor(state.contextPercent)));
  const provider = sanitizeLabel(state.provider || 'no-provider');
  const model = sanitizeLabel(state.model || 'no-model');
  const thinking = sanitizeLabel(state.thinking || 'off');
  const branch = sanitizeLabel(state.branch || 'no-git');
  const separator = color(palette.separator, '·');
  const providerModel = `${color(palette.muted, `${provider}/`)}${color(palette.brand, model)}`;
  const context = `${renderContextBar(contextPercent, contextBarWidth(width), palette)} ${color(
    usageColor(contextPercent, palette),
    `${contextPercent}%`
  )}`;
  const items = [
    `${color(palette.brand, `${SYMBOLS.atlas} Atlas`)} ${color(palette.text, `v${sanitizeLabel(state.atlasVersion)}`)}`,
    `${providerModel} ${color(palette.path, SYMBOLS.thinking)} ${color(palette.brand, thinking)}`,
    `${color(palette.omp, SYMBOLS.omp)} ${color(palette.text, sanitizeLabel(state.ompVersion))}`,
    color(palette.path, formatCurrentDirectory(state.cwd)),
    `${color(palette.branch, NERD_FONT_ICONS.branch)} ${color(palette.branch, branch)}`,
    context,
  ];

  if (state.cachePercent !== null) {
    const cachePercent = Math.max(0, Math.min(100, Math.round(state.cachePercent)));
    items.push(
      `${color(palette.cacheIcon, NERD_FONT_ICONS.cache)} ${color(cacheColor(cachePercent, palette), `${cachePercent}%`)}`
    );
  }

  const lines = layoutStatusItems(items, width, ` ${separator} `);
  const statuses = Array.from(state.statuses?.entries() ?? [])
    .sort(([left], [right]) => left.localeCompare(right))
    .map(([, text]) => sanitizeStatusText(text));
  if (statuses.length > 0 && width > 0) {
    lines.push(truncateToWidth(statuses.join(' '), width));
  }
  return lines;
}

export default async function atlasStatusLine(omp: ExtensionAPI): Promise<void> {
  const atlasVersion = await readAtlasVersion();
  let requestRender: (() => void) | undefined;
  const refresh = () => requestRender?.();

  omp.on('turn_start', refresh);
  omp.on('turn_end', refresh);
  omp.on('message_end', refresh);
  omp.on('session_compact', refresh);
  omp.on('session_switch', refresh);

  omp.on('session_start', (_event, ctx) => {
    if (!ctx.hasUI) return;
    ctx.ui.setFooter((tui, theme, footerData) => {
      let active = true;
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
        render(width: number): string[] {
          const usage = ctx.getContextUsage();
          return renderAtlasFooter(
            {
              atlasVersion,
              provider: ctx.model?.provider || 'no-provider',
              model: ctx.model?.id || 'no-model',
              thinking: omp.getThinkingLevel() || 'off',
              ompVersion: VERSION,
              cwd: ctx.sessionManager.getCwd(),
              branch: footerData.getGitBranch(),
              contextPercent: usage?.percent ?? 0,
              cachePercent: latestCacheHitPercent(ctx.sessionManager.getBranch()),
              isLight: theme.isLight,
              statuses: footerData.getExtensionStatuses(),
            },
            width
          );
        },
      };
    });
  });

  omp.on('session_shutdown', (_event, ctx) => {
    requestRender = undefined;
    if (ctx.hasUI) ctx.ui.setFooter(undefined);
  });
}
