import { execFileSync } from 'node:child_process';
import { readFileSync, watch, type FSWatcher } from 'node:fs';
import { basename, dirname, join, parse, sep } from 'node:path';
import {
  type ExtensionAPI,
  type ExtensionContext,
  type Theme,
  type ThemeColor,
  VERSION
} from '@oh-my-pi/pi-coding-agent';
import {
  Ellipsis,
  truncateToWidth,
  visibleWidth,
  type Component,
  type EditorTopBorder,
  type TUI
} from '@oh-my-pi/pi-tui';
import { getAgentDir, MAIN_CONFIG_FILENAMES } from '@oh-my-pi/pi-utils';

type BunRuntime = {
  stripANSI(text: string): string;
  YAML: { parse(text: string): unknown };
};
const bunRuntime = (globalThis as typeof globalThis & { Bun: BunRuntime }).Bun;

const SYMBOLS = {
  omp: 'π',
  contextCell: '⛁'
} as const;

const WIDGET_KEY = 'atlas-status-line';

export const NERD_FONT_ICONS = {
  branch: '\uF418', // nf-oct-git_branch
  cache: '\uF49B', // nf-oct-cache
  thinking: '\uF0E7' // nf-fa-bolt
} as const;

type Rgb = readonly [number, number, number];
type CacheUsage = { input: number; cacheRead: number; cacheWrite: number };
type EditorTopBorderProvider = (availableWidth: number) => EditorTopBorder | undefined;
type EditorBorderHost = {
  borderColor: (text: string) => string;
  setTopBorderProvider(provider: EditorTopBorderProvider | undefined): void;
};
type NativeStatusLine = {
  getTopBorder(width: number): EditorTopBorder;
};
type ComponentWithChildren = Component & { children?: readonly Component[] };

function findComponent<T extends Component>(
  root: Component,
  predicate: (component: Component) => component is T
): T | undefined {
  if (predicate(root)) return root;
  const children = (root as ComponentWithChildren).children;
  if (!children) return undefined;
  for (const child of children) {
    const match = findComponent(child, predicate);
    if (match) return match;
  }
  return undefined;
}

function isEditorBorderHost(component: Component): component is Component & EditorBorderHost {
  const candidate = component as Partial<EditorBorderHost>;
  return (
    typeof candidate.borderColor === 'function' &&
    typeof candidate.setTopBorderProvider === 'function'
  );
}

function isNativeStatusLine(component: Component): component is Component & NativeStatusLine {
  return typeof (component as Partial<NativeStatusLine>).getTopBorder === 'function';
}

export type StatusPalette = {
  brand: Rgb;
  text: Rgb;
  muted: Rgb;
  separator: Rgb;
  border: Rgb;
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
  provider: string;
  model: string;
  thinking: string;
  ompVersion: string;
  cwd: string;
  branch: string | null;
  contextPercent: number;
  cachePercent: number | null;
  sessionTitle?: string | null;
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
  border: [23, 143, 185],
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
  border: [87, 82, 121],
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
    border: themed('border', canonical.border),
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
  return bunRuntime.stripANSI(text)
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

// Middle-truncate long branch names, matching claude/.claude/Atlas/atlas-statusline.sh:
// over 24 chars → first 11 + … + last 12, preserving date/ticket suffixes.
export function formatGitBranch(branch: string): string {
  return branch.length > 24 ? `${branch.slice(0, 11)}…${branch.slice(-12)}` : branch;
}

export function calculateCacheHitPercent(usage: CacheUsage): number | null {
  const { input, cacheRead, cacheWrite } = usage;
  if (![input, cacheRead, cacheWrite].every((value) => Number.isFinite(value) && value >= 0))
    return null;
  if (cacheRead + cacheWrite <= 0) return null;

  const promptTokens = input + cacheRead + cacheWrite;
  return Math.max(0, Math.min(100, Math.round((cacheRead / promptTokens) * 100)));
}

// Session cache-hit rate from the session manager's cumulative usage snapshot.
// The per-message session entries returned by getBranch() are a different shape
// than the compaction module's Usage records, so read the aggregate snapshot
// (input + cacheRead + cacheWrite) instead; the same fields calculateCacheHitPercent needs.
export function sessionCacheHitPercent(
  sessionManager: { getUsageStatistics?: () => CacheUsage | undefined } | undefined
): number | null {
  const stats = sessionManager?.getUsageStatistics?.();
  return stats ? calculateCacheHitPercent(stats) : null;
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


type ResponsiveItem = {
  key: 'model' | 'omp' | 'path' | 'branch' | 'context' | 'cache';
  content: string;
};

function renderStatusGroup(items: string[], separator: string): string {
  return items.length === 0 ? '' : ` ${items.join(` ${separator} `)} `;
}

function responsiveStatusLine(
  sourceLeft: ResponsiveItem[],
  sourceRight: string[],
  terminalWidth: number,
  separator: string,
  gapColor: Rgb
): string {
  const width = Math.max(0, Math.floor(terminalWidth));
  if (width < 2) return ' '.repeat(width);

  let leftItems = sourceLeft;
  let left = renderStatusGroup(
    leftItems.map((item) => item.content),
    separator
  );
  let leftWidth = visibleWidth(left);
  const right = renderStatusGroup(sourceRight, separator);
  const rightWidth = visibleWidth(right);

  for (const key of ['path', 'branch', 'model', 'omp'] as const) {
    const gapWidth = leftWidth > 0 && rightWidth > 0 ? 1 : 0;
    if (leftWidth + rightWidth + gapWidth <= width) break;

    leftItems = leftItems.filter((item) => item.key !== key);
    left = renderStatusGroup(
      leftItems.map((item) => item.content),
      separator
    );
    leftWidth = visibleWidth(left);
  }

  let fittedRight = right;
  const gapWidth = leftWidth > 0 && rightWidth > 0 ? 1 : 0;
  if (sourceRight.length > 0 && leftWidth + rightWidth + gapWidth > width) {
    const availableRight = Math.max(0, width - leftWidth - 1);
    const availableContent = Math.max(0, availableRight - 2);
    const rightContent = truncateToWidth(sourceRight.join(` ${separator} `), availableContent);
    fittedRight = rightContent ? renderStatusGroup([rightContent], separator) : '';
  }

  if (!fittedRight) return truncateToWidth(left, width);
  if (!left) return truncateToWidth(fittedRight, width);

  const dividerWidth = Math.max(1, width - leftWidth - visibleWidth(fittedRight));
  const divider = color(gapColor, '─'.repeat(dividerWidth));
  return truncateToWidth(`${left}${divider}${fittedRight}`, width, '');
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
  const branch = formatGitBranch(sanitizeLabel(state.branch || 'no-git'));
  const separator = color(palette.separator, '');
  const providerModel = `${color(palette.muted, `${provider}/`)}${color(palette.brand, model)}`;
  const context = `${renderContextBar(contextPercent, contextBarWidth(width), palette)} ${color(
    usageColor(contextPercent, palette),
    `${contextPercent}%`
  )}`;
  const leftItems: ResponsiveItem[] = [
    {
      key: 'omp',
      content: `${color(palette.omp, SYMBOLS.omp)} ${color(palette.text, sanitizeLabel(state.ompVersion))}`
    },
    {
      key: 'model',
      content: `${providerModel} ${color(palette.path, NERD_FONT_ICONS.thinking)} ${color(palette.brand, thinking)}`
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
  leftItems.push({ key: 'context', content: context });

  if (state.cachePercent !== null) {
    const cachePercent = Math.max(0, Math.min(100, Math.round(state.cachePercent)));
    leftItems.push({
      key: 'cache',
      content: `${color(palette.cacheIcon, NERD_FONT_ICONS.cache)} ${color(cacheColor(cachePercent, palette), `${cachePercent}%`)}`
    });
  }

  const sessionTitle = sanitizeLabel(state.sessionTitle ?? '');
  const rightItems = sessionTitle
    ? [color(palette.brand, truncateToWidth(sessionTitle, 40))]
    : [];

  const primary = responsiveStatusLine(leftItems, rightItems, width, separator, palette.border);
  const lines = primary
    ? applyStatusLineBackground([primary], visibleWidth(primary), state.backgroundAnsi)
    : [];
  return lines.concat(renderExtensionStatusRows(state.statuses, width, state.backgroundAnsi));
}

export function renderExtensionStatusRows(
  statuses: ReadonlyMap<string, string> | undefined,
  terminalWidth: number,
  backgroundAnsi: string | undefined
): string[] {
  const width = Math.max(0, Math.floor(terminalWidth));
  if (width === 0 || !statuses || statuses.size === 0) return [];
  const text = Array.from(statuses.entries())
    .sort(([left], [right]) => left.localeCompare(right))
    .map(([, status]) => `${sanitizeStatusText(status)}\x1b[0m`)
    .join(' ');
  return applyStatusLineBackground([insetStatusText(text, width)], width, backgroundAnsi);
}

function readStatusLineTransparency(): boolean {
  for (const filename of MAIN_CONFIG_FILENAMES) {
    try {
      const config = bunRuntime.YAML.parse(readFileSync(join(getAgentDir(), filename), 'utf8')) as {
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

export default function atlasStatusLine(omp: ExtensionAPI): void {
  let requestRender: (() => void) | undefined;
  let restoreSetStatus: (() => void) | undefined;
  let removeWidget: (() => void) | undefined;
  const refresh = () => requestRender?.();
  const clearInstalledWidget = () => {
    restoreSetStatus?.();
    restoreSetStatus = undefined;
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
    restoreSetStatus = () => {
      if (ctx.ui.setStatus === captureStatus) ctx.ui.setStatus = originalSetStatus;
    };

    ctx.ui.setWidget(
      WIDGET_KEY,
      (tui: TUI, _theme) => {
        let active = true;
        let currentCwd = ctx.sessionManager.getCwd();
        let branch = readGitBranch(currentCwd);
        let transparent = readStatusLineTransparency();
        let installedEditor: EditorBorderHost | undefined;
        const renderNow = () => {
          if (active) tui.requestRender();
        };
        const watchCurrentBranch = () =>
          watchGitBranch(currentCwd, () => {
            branch = readGitBranch(currentCwd);
            renderNow();
          });
        let branchWatcher = watchCurrentBranch();
        const syncCwd = () => {
          const nextCwd = ctx.sessionManager.getCwd();
          if (nextCwd === currentCwd) return;

          branchWatcher?.close();
          currentCwd = nextCwd;
          branch = readGitBranch(currentCwd);
          branchWatcher = watchCurrentBranch();
        };
        const nativeStatusLine = findComponent(tui, isNativeStatusLine);
        const nativeTopBorderProvider = nativeStatusLine
          ? (width: number) => nativeStatusLine.getTopBorder(width)
          : undefined;
        const backgroundAnsi = () => {
          const activeTheme = ctx.ui.theme;
          return transparent ? DEFAULT_BACKGROUND_ANSI : activeTheme.getBgAnsi('statusLineBg');
        };
        const currentState = (): AtlasFooterState => {
          syncCwd();
          const usage = ctx.getContextUsage();
          const activeTheme = ctx.ui.theme;
          return {
            provider: ctx.model?.provider || 'no-provider',
            model: ctx.model?.id || 'no-model',
            thinking: omp.getThinkingLevel() || 'off',
            ompVersion: VERSION,
            cwd: currentCwd,
            branch,
            contextPercent: usage?.percent ?? 0,
            cachePercent: sessionCacheHitPercent(ctx.sessionManager),
            sessionTitle: ctx.sessionManager.getSessionName(),
            isLight: activeTheme.isLight,
            theme: activeTheme,
            backgroundAnsi: backgroundAnsi()
          };
        };
        const restoreInstalledEditor = () => {
          if (!installedEditor) return;

          installedEditor.setTopBorderProvider(nativeTopBorderProvider);
          installedEditor = undefined;
        };
        const syncTopBorderProvider = () => {
          const editor = findComponent(tui, isEditorBorderHost);
          if (!editor) return false;
          if (editor === installedEditor) return false;
          restoreInstalledEditor();

          // Return the Atlas footer as the editor's top-border content and let the
          // editor fill the remaining width with its native border rule out to the
          // corner. (An earlier revision suppressed that trailing fill, which left
          // the right corner detached from the terminal edge.)
          const atlasTopBorderProvider = (width: number): EditorTopBorder | undefined => {
            const content = renderAtlasFooter(currentState(), width)[0];
            return content === undefined ? undefined : { content, width: visibleWidth(content) };
          };

          editor.setTopBorderProvider(atlasTopBorderProvider);
          installedEditor = editor;
          return true;
        };
        syncTopBorderProvider();

        const configWatcher = watchStatusLineConfig(() => {
          transparent = readStatusLineTransparency();
          renderNow();
        });
        requestRender = renderNow;

        return {
          dispose() {
            active = false;
            restoreInstalledEditor();
            branchWatcher?.close();
            configWatcher?.close();
            if (requestRender === renderNow) requestRender = undefined;
          },
          render(width: number): string[] {
            if (syncTopBorderProvider()) renderNow();
            return renderExtensionStatusRows(statuses, width, backgroundAnsi());
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
