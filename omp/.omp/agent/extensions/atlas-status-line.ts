import { execFileSync } from 'node:child_process';
import { readFileSync, watch, type FSWatcher } from 'node:fs';
import { homedir } from 'node:os';
import { basename, dirname, join, parse, sep } from 'node:path';
import { getLastAssistantUsage, type SessionEntry } from '@oh-my-pi/pi-agent-core/compaction';
import {
  type ExtensionAPI,
  type ExtensionContext,
  type Theme,
  type ThemeColor,
  VERSION
} from '@oh-my-pi/pi-coding-agent';
import { Ellipsis, truncateToWidth, visibleWidth } from '@oh-my-pi/pi-tui';
import { getAgentDir, MAIN_CONFIG_FILENAMES } from '@oh-my-pi/pi-utils';

const SYMBOLS = {
  atlas: '▲',
  thinking: '⚡︎', // U+FE0E forces text presentation.
  omp: 'π',
  contextCell: '⛁'
} as const;

const WIDGET_KEY = 'atlas-status-line';

export const NERD_FONT_ICONS = {
  branch: '\uF418', // nf-oct-git_branch
  cache: '\uF49B' // nf-oct-cache
} as const;

type Rgb = readonly [number, number, number];
type CacheUsage = { input: number; cacheRead: number; cacheWrite: number };

export type StatusPalette = {
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
  theme?: Pick<Theme, 'getColorHex'>;
  backgroundAnsi?: string;
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
    [241, 108, 117]
  ],
  usage: [
    [55, 244, 153],
    [251, 191, 36],
    [251, 146, 60],
    [241, 108, 117]
  ],
  cache: [
    [55, 244, 153],
    [4, 209, 249],
    [241, 252, 121],
    [247, 198, 127],
    [241, 108, 117]
  ]
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
    [180, 99, 122]
  ],
  usage: [
    [40, 105, 131],
    [234, 157, 52],
    [215, 130, 126],
    [180, 99, 122]
  ],
  cache: [
    [40, 105, 131],
    [86, 148, 159],
    [234, 157, 52],
    [215, 130, 126],
    [180, 99, 122]
  ]
};

function rgbFromHex(hex: string, fallback: Rgb): Rgb {
  const match = /^#([\da-f]{2})([\da-f]{2})([\da-f]{2})$/i.exec(hex);
  return match
    ? [Number.parseInt(match[1], 16), Number.parseInt(match[2], 16), Number.parseInt(match[3], 16)]
    : fallback;
}

export function statusPaletteFromTheme(
  theme: Pick<Theme, 'getColorHex'>,
  isLight: boolean
): StatusPalette {
  const canonical = isLight ? LIGHT_PALETTE : DARK_PALETTE;
  const themed = (name: ThemeColor, fallback: Rgb) => rgbFromHex(theme.getColorHex(name), fallback);

  return {
    ...canonical,
    brand: themed('statusLineModel', canonical.brand),
    text: themed('text', canonical.text),
    muted: themed('muted', canonical.muted),
    separator: themed('statusLineSep', canonical.separator),
    path: themed('statusLinePath', canonical.path),
    branch: themed('statusLineGitClean', canonical.branch),
    omp: themed('statusLineOutput', canonical.omp),
    cacheIcon: themed('accent', canonical.cacheIcon),
    usage: [
      themed('success', canonical.usage[0]),
      themed('warning', canonical.usage[1]),
      themed('statusLineOutput', canonical.usage[2]),
      themed('error', canonical.usage[3])
    ],
    cache: [
      themed('success', canonical.cache[0]),
      themed('statusLinePath', canonical.cache[1]),
      themed('warning', canonical.cache[2]),
      themed('statusLineOutput', canonical.cache[3]),
      themed('error', canonical.cache[4])
    ]
  };
}

export function sanitizeLabel(text: string): string {
  return Bun.stripANSI(text)
    .replace(/[\p{Cc}\p{Cf}]/gu, (character) =>
      character === '\r' || character === '\n' || character === '\t' ? ' ' : ''
    )
    .replace(/ +/g, ' ')
    .trim();
}

export function sanitizeStatusText(text: string): string {
  let output = '';
  let index = 0;
  while (index < text.length) {
    const character = text[index];
    if (character === '\x1b') {
      const introducer = text[index + 1];
      if (introducer === '[') {
        let end = index + 2;
        while (end < text.length) {
          const code = text.charCodeAt(end);
          if (code >= 0x40 && code <= 0x7e) break;
          end += 1;
        }
        if (end >= text.length) break;

        const parameters = text.slice(index + 2, end);
        if (text[end] === 'm' && /^[0-9:;]*$/.test(parameters)) {
          output += text.slice(index, end + 1);
        }
        index = end + 1;
        continue;
      }

      if (introducer !== undefined && ']P_X^'.includes(introducer)) {
        index += 2;
        while (index < text.length) {
          if (text[index] === '\u0007') {
            index += 1;
            break;
          }
          if (text[index] === '\x1b' && text[index + 1] === '\\') {
            index += 2;
            break;
          }
          index += 1;
        }
        continue;
      }

      index += introducer === undefined ? 1 : 2;
      continue;
    }

    const codePoint = text.codePointAt(index);
    if (codePoint === undefined) break;
    const glyph = String.fromCodePoint(codePoint);
    index += glyph.length;
    if (/[\p{Cc}\p{Cf}]/u.test(glyph)) {
      if (glyph === '\r' || glyph === '\n' || glyph === '\t') output += ' ';
      continue;
    }
    output += glyph;
  }

  return output.replace(/ +/g, ' ').trim();
}

function color(rgb: Rgb, text: string): string {
  return `\x1b[38;2;${rgb[0]};${rgb[1]};${rgb[2]}m${text}\x1b[39m`;
}

function mix(from: Rgb, to: Rgb, amount: number): Rgb {
  const clamped = Math.max(0, Math.min(1, amount));
  return [
    Math.round(from[0] + (to[0] - from[0]) * clamped),
    Math.round(from[1] + (to[1] - from[1]) * clamped),
    Math.round(from[2] + (to[2] - from[2]) * clamped)
  ];
}

async function readAtlasVersion(): Promise<string> {
  try {
    const version = await Bun.file(join(homedir(), '.claude', 'LIFEOS', 'VERSION')).text();
    return sanitizeLabel(version) || '—';
  } catch {
    return '—';
  }
}

function gitOutput(cwd: string, args: string[]): string | null {
  try {
    return (
      execFileSync('git', ['-C', cwd, ...args], {
        encoding: 'utf8',
        stdio: ['ignore', 'pipe', 'ignore']
      }).trim() || null
    );
  } catch {
    return null;
  }
}

export function readGitBranch(cwd: string): string | null {
  return (
    gitOutput(cwd, ['symbolic-ref', '--quiet', '--short', 'HEAD']) ??
    gitOutput(cwd, ['rev-parse', '--short', 'HEAD'])
  );
}

function watchGitBranch(cwd: string, onChange: () => void): FSWatcher | undefined {
  const headPath = gitOutput(cwd, ['rev-parse', '--path-format=absolute', '--git-path', 'HEAD']);
  if (!headPath) return undefined;

  try {
    return watch(dirname(headPath), { persistent: false }, (_event, filename) => {
      if (filename?.toString() === basename(headPath)) onChange();
    });
  } catch {
    return undefined;
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
  if (![input, cacheRead, cacheWrite].every((value) => Number.isFinite(value) && value >= 0))
    return null;
  if (cacheRead + cacheWrite <= 0) return null;

  const promptTokens = input + cacheRead + cacheWrite;
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

const DEFAULT_BACKGROUND_ANSI = '\x1b[49m';

function restoresDefaultBackground(parameters: string): boolean {
  if (parameters === '') return true;
  return parameters.split(';').some((parameter) => {
    const code = parameter.split(':', 1)[0];
    return code === '' || code === '0' || code === '49';
  });
}

export function applyStatusLineBackground(
  lines: string[],
  terminalWidth: number,
  backgroundAnsi: string | undefined
): string[] {
  if (backgroundAnsi === undefined) return lines;

  const width = Math.max(0, Math.floor(terminalWidth));
  return lines.map((line) => {
    const fitted = truncateToWidth(line, width);
    const padded = fitted + ' '.repeat(Math.max(0, width - visibleWidth(fitted)));
    if (backgroundAnsi === '' || backgroundAnsi === DEFAULT_BACKGROUND_ANSI) return padded;

    const restored = padded.replace(/\x1b\[([0-9:;]*)m/g, (sequence, parameters: string) =>
      restoresDefaultBackground(parameters) ? `${sequence}${backgroundAnsi}` : sequence
    );
    return `${backgroundAnsi}${restored}${DEFAULT_BACKGROUND_ANSI}`;
  });
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

type ResponsiveItem = {
  key: 'atlas' | 'model' | 'omp' | 'path' | 'branch';
  content: string;
};

function responsiveStatusLine(
  sourceLeft: ResponsiveItem[],
  sourceRight: string[],
  terminalWidth: number,
  separator: string
): string {
  const width = Math.max(0, Math.floor(terminalWidth));
  if (width === 0) return '';

  const innerWidth = Math.max(0, width - 2);
  const join = (items: string[]) => items.join(` ${separator} `);
  let rightItems = sourceRight;
  let right = join(rightItems);
  if (visibleWidth(right) > innerWidth && rightItems.length > 1) {
    rightItems = rightItems.slice(0, 1);
    right = join(rightItems);
  }
  right = truncateToWidth(right, innerWidth);

  let leftItems = sourceLeft;
  for (const key of ['path', 'omp', 'branch'] as const) {
    const left = join(leftItems.map((item) => item.content));
    const gap = left && right ? 1 : 0;
    if (visibleWidth(left) + gap + visibleWidth(right) <= innerWidth) break;
    leftItems = leftItems.filter((item) => item.key !== key);
  }

  const rightWidth = visibleWidth(right);
  const availableLeft = Math.max(0, innerWidth - rightWidth - (right ? 1 : 0));
  const left = truncateToWidth(join(leftItems.map((item) => item.content)), availableLeft);
  const gapWidth = Math.max(0, innerWidth - visibleWidth(left) - rightWidth);
  return ` ${left}${' '.repeat(gapWidth)}${right} `;
}

function insetStatusText(text: string, terminalWidth: number): string {
  const width = Math.max(0, Math.floor(terminalWidth));
  if (width === 0) return '';
  const innerWidth = Math.max(0, width - 2);
  const fitted = truncateToWidth(text, innerWidth);
  return ` ${fitted}${' '.repeat(Math.max(0, innerWidth - visibleWidth(fitted)))} `;
}

export function renderAtlasFooter(state: AtlasFooterState, terminalWidth: number): string[] {
  const canonicalPalette = state.isLight ? LIGHT_PALETTE : DARK_PALETTE;
  const palette = state.theme
    ? statusPaletteFromTheme(state.theme, state.isLight)
    : canonicalPalette;
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
  const leftItems: ResponsiveItem[] = [
    {
      key: 'atlas',
      content: `${color(palette.brand, `${SYMBOLS.atlas} Atlas`)} ${color(palette.text, `v${sanitizeLabel(state.atlasVersion)}`)}`
    },
    {
      key: 'model',
      content: `${providerModel} ${color(palette.path, SYMBOLS.thinking)} ${color(palette.brand, thinking)}`
    },
    {
      key: 'omp',
      content: `${color(palette.omp, SYMBOLS.omp)} ${color(palette.text, sanitizeLabel(state.ompVersion))}`
    },
    {
      key: 'path',
      content: color(palette.path, formatCurrentDirectory(state.cwd))
    },
    {
      key: 'branch',
      content: `${color(palette.branch, NERD_FONT_ICONS.branch)} ${color(palette.branch, branch)}`
    }
  ];
  const rightItems = [context];

  if (state.cachePercent !== null && width >= 80) {
    const cachePercent = Math.max(0, Math.min(100, Math.round(state.cachePercent)));
    rightItems.push(
      `${color(palette.cacheIcon, NERD_FONT_ICONS.cache)} ${color(cacheColor(cachePercent, palette), `${cachePercent}%`)}`
    );
  }

  const lines = width > 0 ? [responsiveStatusLine(leftItems, rightItems, width, separator)] : [];
  const statuses = Array.from(state.statuses?.entries() ?? [])
    .sort(([left], [right]) => left.localeCompare(right))
    .map(([, text]) => sanitizeStatusText(text));
  if (statuses.length > 0 && width > 0) {
    lines.push(insetStatusText(statuses.join(' '), width));
  }
  return applyStatusLineBackground(lines, width, state.backgroundAnsi);
}

function readStatusLineTransparency(): boolean {
  for (const filename of MAIN_CONFIG_FILENAMES) {
    try {
      const config = Bun.YAML.parse(readFileSync(join(getAgentDir(), filename), 'utf8')) as {
        statusLine?: { transparent?: unknown };
      };
      return config.statusLine?.transparent === true;
    } catch {
      // Try the legacy-compatible filename before using the opaque default.
    }
  }
  return false;
}

function watchStatusLineConfig(onChange: () => void): FSWatcher | undefined {
  try {
    return watch(getAgentDir(), { persistent: false }, (_event, filename) => {
      const changed = filename?.toString();
      if (changed && MAIN_CONFIG_FILENAMES.some((candidate) => candidate === changed)) onChange();
    });
  } catch {
    return undefined;
  }
}

export default async function atlasStatusLine(omp: ExtensionAPI): Promise<void> {
  const atlasVersion = await readAtlasVersion();
  let requestRender: (() => void) | undefined;
  let restoreStatusPublisher: (() => void) | undefined;
  let removeWidget: (() => void) | undefined;
  const refresh = () => requestRender?.();
  const clearInstalledWidget = () => {
    restoreStatusPublisher?.();
    restoreStatusPublisher = undefined;
    requestRender = undefined;
    removeWidget?.();
    removeWidget = undefined;
  };
  const installWidget = (ctx: ExtensionContext) => {
    clearInstalledWidget();
    if (!ctx.hasUI) return;

    const statuses = new Map<string, string>();
    const originalSetStatus = ctx.ui.setStatus;
    const captureStatus = (key: string, text: string | undefined) => {
      if (text === undefined) {
        statuses.delete(key);
      } else {
        statuses.set(key, text);
      }
      requestRender?.();
    };
    ctx.ui.setStatus = captureStatus;
    restoreStatusPublisher = () => {
      if (ctx.ui.setStatus === captureStatus) ctx.ui.setStatus = originalSetStatus;
    };

    const cwd = ctx.sessionManager.getCwd();
    ctx.ui.setWidget(
      WIDGET_KEY,
      (tui, _theme) => {
        let active = true;
        let branch = readGitBranch(cwd);
        let transparent = readStatusLineTransparency();
        const renderNow = () => {
          if (active) tui.requestRender();
        };
        const branchWatcher = watchGitBranch(cwd, () => {
          branch = readGitBranch(cwd);
          renderNow();
        });
        const configWatcher = watchStatusLineConfig(() => {
          transparent = readStatusLineTransparency();
          renderNow();
        });
        requestRender = renderNow;

        return {
          dispose() {
            active = false;
            branchWatcher?.close();
            configWatcher?.close();
            if (requestRender === renderNow) requestRender = undefined;
          },
          render(width: number): string[] {
            const usage = ctx.getContextUsage();
            const activeTheme = ctx.ui.theme;
            return renderAtlasFooter(
              {
                atlasVersion,
                provider: ctx.model?.provider || 'no-provider',
                model: ctx.model?.id || 'no-model',
                thinking: omp.getThinkingLevel() || 'off',
                ompVersion: VERSION,
                cwd,
                branch,
                contextPercent: usage?.percent ?? 0,
                cachePercent: latestCacheHitPercent(ctx.sessionManager.getBranch()),
                isLight: activeTheme.isLight,
                theme: activeTheme,
                backgroundAnsi: transparent
                  ? DEFAULT_BACKGROUND_ANSI
                  : activeTheme.getBgAnsi('statusLineBg'),
                statuses
              },
              width
            );
          }
        };
      },
      { placement: 'belowEditor' }
    );
    removeWidget = () => ctx.ui.setWidget(WIDGET_KEY, undefined);
  };

  omp.on('turn_start', refresh);
  omp.on('turn_end', refresh);
  omp.on('message_end', refresh);
  omp.on('session_compact', refresh);
  omp.on('session_start', (_event, ctx) => installWidget(ctx));
  omp.on('session_switch', (_event, ctx) => installWidget(ctx));
  omp.on('session_shutdown', clearInstalledWidget);
}
