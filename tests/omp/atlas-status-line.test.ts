import { describe, expect, it } from 'bun:test';
import {
  VERSION as OMP_VERSION,
  type ExtensionAPI,
  type Theme,
  type ThemeColor
} from '@oh-my-pi/pi-coding-agent';
import { visibleWidth, type EditorTopBorder } from '@oh-my-pi/pi-tui';
import atlasStatusLine, {
  applyStatusLineBackground,
  calculateCacheHitPercent,
  contextBarWidth,
  contextFilledCells,
  formatCurrentDirectory,
  formatGitBranch,
  layoutStatusItems,
  NERD_FONT_ICONS,
  sessionCacheHitPercent,
  renderAtlasFooter,
  renderExtensionStatusRows,
  sanitizeLabel,
  sanitizeStatusText,
  statusPaletteFromTheme
} from '../../omp/.omp/agent/extensions/atlas-status-line';

function stripAnsi(text: string): string {
  return Bun.stripANSI(text);
}

const FOOTER_STATE = {
  atlasVersion: '7.1.1',
  provider: 'openai',
  model: 'gpt-5.6',
  thinking: 'high',
  ompVersion: '17.0.0',
  cwd: '/Users/ed/Developer/Atlas/Config',
  branch: 'main',
  contextPercent: 42,
  cachePercent: 99,
  isLight: false
};

const THEME_COLORS: Partial<Record<ThemeColor, string>> = {
  statusLineModel: '#010203',
  text: '#040506',
  muted: '#070809',
  statusLineSep: '#0a0b0c',
  statusLinePath: '#0d0e0f',
  border: '#2a2b2c',
  statusLineGitClean: '#101112',
  statusLineOutput: '#131415',
  accent: '#161718',
  success: '#192021',
  warning: '#222324',
  error: '#252627'
};

function mockTheme(isLight = false): Pick<Theme, 'isLight' | 'getColorHex' | 'getBgAnsi'> {
  return {
    isLight,
    getColorHex(color: ThemeColor) {
      return THEME_COLORS[color] ?? '#303132';
    },
    getBgAnsi() {
      return '\x1b[48;2;33;35;55m';
    }
  };
}

describe('Atlas OMP status line', () => {
  it('shows only the current directory and its parent', () => {
    expect(formatCurrentDirectory('/Users/ed/Developer/Atlas/Config')).toBe('…/Atlas/Config');
    expect(formatCurrentDirectory('/Atlas/Config')).toBe('…/Atlas/Config');
    expect(formatCurrentDirectory('/')).toBe('/');
  });

  it('removes label controls while preserving trusted SGR status color', () => {
    expect(sanitizeLabel('open\n\t\x1b[31mai\x1b[0m\u0007\u202E\u2066')).toBe('open ai');
    expect(sanitizeStatusText('\x1b[31mred\x1b[39m\n\u0007ready')).toBe(
      '\x1b[31mred\x1b[39m ready'
    );
  });

  it('strips non-SGR terminal escapes from extension statuses', () => {
    const unsafe = [
      '\x1b[2Jclear',
      '\x1b[?25lhidden',
      '\x1b[2Acursor',
      '\x1b]8;;https://example.com\x1b\\link\x1b]8;;\x1b\\',
      '\x1b]52;c;payload\u0007clip',
      '\x1bcreset',
      'broken\x1b[31'
    ].join(' ');

    expect(sanitizeStatusText(unsafe)).toBe('clear hidden cursor link clip reset broken');
  });

  it('isolates unterminated ANSI styles between extension statuses and rows', () => {
    const statuses = new Map([
      ['a', '\x1b[31;1mred'],
      ['b', '\x1b[44mblue'],
      ['c', 'plain']
    ]);
    const background = '\x1b[48;2;33;35;55m';
    const opaque = renderExtensionStatusRows(statuses, 40, background)[0];
    const transparent = renderExtensionStatusRows(statuses, 40, '\x1b[49m')[0];

    expect(opaque).toContain(
      `\x1b[31;1mred\x1b[0m${background} \x1b[44mblue\x1b[0m${background} plain\x1b[0m${background}`
    );
    expect(opaque).toEndWith('\x1b[49m');
    expect(transparent.match(/\x1b\[[0-9;]*m/g)?.at(-1)).toBe('\x1b[0m');
    expect(stripAnsi(transparent).trim()).toBe('red blue plain');
  });

  it('uses the canonical responsive context-meter widths', () => {
    expect([34, 35, 54, 55, 79, 80, 160].map(contextBarWidth)).toEqual([6, 8, 8, 16, 16, 24, 24]);
  });

  it('floors filled context cells within meter bounds', () => {
    expect(contextFilledCells(0, 24)).toBe(0);
    expect(contextFilledCells(50, 24)).toBe(12);
    expect(contextFilledCells(99, 24)).toBe(23);
    expect(contextFilledCells(100, 24)).toBe(24);
    expect(contextFilledCells(150, 24)).toBe(24);
  });

  it('renders a 99% cache hit and hides input-only usage', () => {
    expect(calculateCacheHitPercent({ input: 1, cacheRead: 99, cacheWrite: 0 })).toBe(99);
    expect(calculateCacheHitPercent({ input: 100, cacheRead: 0, cacheWrite: 0 })).toBeNull();
  });

  it('reads cache hit rate from the session usage snapshot', () => {
    expect(sessionCacheHitPercent({ getUsageStatistics: () => ({ input: 1, cacheRead: 99, cacheWrite: 0 }) })).toBe(99);
    expect(sessionCacheHitPercent({ getUsageStatistics: () => ({ input: 100, cacheRead: 0, cacheWrite: 0 }) })).toBeNull();
    expect(sessionCacheHitPercent({ getUsageStatistics: () => undefined })).toBeNull();
    expect(sessionCacheHitPercent({})).toBeNull();
    expect(sessionCacheHitPercent(undefined)).toBeNull();
  });

  it('middle-truncates long branch names like the bash status line', () => {
    expect(formatGitBranch('main')).toBe('main');
    expect(formatGitBranch('feat/atlas-agent-config-updates')).toBe('feat/atlas-…nfig-updates');
    expect(visibleWidth(formatGitBranch('feat/atlas-agent-config-updates'))).toBe(24);
    expect(formatGitBranch('a'.repeat(24))).toBe('a'.repeat(24));
  });


  it('uses the exact canonical glyph codepoints', () => {
    expect(NERD_FONT_ICONS.branch.codePointAt(0)).toBe(0xf418);
    expect(NERD_FONT_ICONS.cache.codePointAt(0)).toBe(0xf49b);
    expect(Array.from('⚡︎', (glyph) => glyph.codePointAt(0))).toEqual([0x26a1, 0xfe0e]);
  });

  it('wraps and truncates ANSI-styled items by visible width', () => {
    const red = '\x1b[31mabcdef\x1b[39m';
    const lines = layoutStatusItems([red, 'xy'], 5, ' · ');
    expect(lines.map(visibleWidth)).toEqual([5, 2]);
    expect(lines[0]).toContain('\x1b[31m');
    expect(stripAnsi(lines.join('\n'))).toBe('abcd…\nxy');
    expect(stripAnsi(lines.join('\n'))).not.toContain('[31m');
  });

  it('renders the complete mocked footer with a colored extension status', () => {
    const lines = renderAtlasFooter(
      {
        ...FOOTER_STATE,
        statuses: new Map([['fixture', '\x1b[35mcolored\x1b[39m']])
      },
      240
    );
    const footer = stripAnsi(lines[0]);

    expect(footer).toContain('▲ Atlas v7.1.1');
    expect(footer).toContain('openai/gpt-5.6 ⚡︎ high');
    expect(footer).toContain('π 17.0.0');
    expect(footer).toContain('…/Atlas/Config');
    expect(footer).toContain(' main');
    expect(footer).toContain('42%');
    expect(footer).toContain(' 99%');
    expect(Array.from(footer).filter((glyph) => glyph === '⛁')).toHaveLength(24);
    expect(footer).toContain(` main  ${'⛁'.repeat(24)} 42%   99%`);
    expect(visibleWidth(lines[0]) < 240).toBeTrue();
    expect(lines[1]).toContain('\x1b[35mcolored\x1b[39m');
    expect(stripAnsi(lines[1]).trim()).toBe('colored');
    expect(lines[0]).toContain('\x1b[38;2;164;140;242m▲ Atlas');
    expect(lines[0]).toContain('\x1b[38;2;232;97;32mπ');
    expect(lines[0]).toContain('\x1b[38;2;4;209;249m…/Atlas/Config');
    expect(lines[0]).toContain('\x1b[38;2;242;101;181m');
    expect(lines[0]).toContain('\x1b[2m\x1b[38;2;50;52;73m⛁');
    expect(lines[0]).toContain('\x1b[38;2;235;250;250mv7.1.1');
    expect(lines[0]).toContain('\x1b[38;2;112;129;208mopenai/');
    expect(lines[0]).toContain('\x1b[38;2;50;52;73m');
  });

  it('preserves cache and a truncated session title at narrow editor widths', () => {
    const titledState = { ...FOOTER_STATE, sessionTitle: 'native-chip-color-rendering' };
    const wide = renderAtlasFooter(titledState, 240)[0];
    const narrow = renderAtlasFooter(titledState, 32)[0];
    const wideText = stripAnsi(wide);
    const narrowText = stripAnsi(narrow);

    expect(wideText.trimEnd()).toEndWith('native-chip-color-rendering');
    expect(wide).toContain('\x1b[38;2;164;140;242mnative-chip-color-rendering');
    expect(visibleWidth(wide)).toBe(240);
    expect(wideText).toMatch(/ 99% ─+ native-chip-color-rendering $/);
    expect(narrowText).toContain(NERD_FONT_ICONS.cache);
    expect(narrowText).toContain('native');
  });

  it('uses the canonical Rosé Pine Dawn light palette', () => {
    const footer = renderAtlasFooter({ ...FOOTER_STATE, isLight: true }, 240)[0];

    expect(footer).toContain('\x1b[38;2;144;122;169m▲ Atlas');
    expect(footer).toContain('\x1b[38;2;234;157;52mπ');
    expect(footer).toContain('\x1b[38;2;86;148;159m…/Atlas/Config');
    expect(footer).toContain('\x1b[38;2;215;130;126m');
    expect(footer).toContain('\x1b[38;2;87;82;121mv7.1.1');
    expect(footer).toContain('\x1b[38;2;121;117;147mopenai/');
    expect(footer).toContain('\x1b[38;2;223;218;217m');
  });

  it('uses active OMP theme colors without changing the canonical context bar', () => {
    const theme = mockTheme();
    const palette = statusPaletteFromTheme(theme, false);
    const footer = renderAtlasFooter({ ...FOOTER_STATE, contextPercent: 100, theme }, 240)[0];

    expect(palette.brand).toEqual([1, 2, 3]);
    expect(palette.path).toEqual([13, 14, 15]);
    expect(footer).toContain('\x1b[38;2;1;2;3m▲ Atlas');
    expect(footer).toContain('\x1b[38;2;13;14;15m…/Atlas/Config');
    expect(footer).toContain('\x1b[38;2;16;17;18m');
    expect(footer).toContain('\x1b[38;2;19;20;21mπ');
    expect(footer).toContain('\x1b[38;2;37;38;39m100%');
    expect(footer).toContain('\x1b[38;2;241;108;117m⛁');
  });

  it('fills the compact primary background and full-width extension statuses', () => {
    const opaque = renderAtlasFooter(
      {
        ...FOOTER_STATE,
        backgroundAnsi: '\x1b[48;2;33;35;55m',
        statuses: new Map([['reset', '\x1b[35mcolored\x1b[0m ready']])
      },
      120
    );
    const transparent = applyStatusLineBackground(['status'], 12, '\x1b[49m');

    expect(visibleWidth(opaque[0]) < 120).toBeTrue();
    expect(visibleWidth(opaque[1])).toBe(120);
    expect(opaque[0]).toStartWith('\x1b[48;2;33;35;55m');
    expect(opaque[1]).toContain('\x1b[0m\x1b[48;2;33;35;55m ready');
    expect(opaque.every((line) => line.endsWith('\x1b[49m'))).toBeTrue();
    expect(transparent).toEqual(['status      ']);
    expect(transparent[0]).not.toContain('\x1b[48;');
  });

  it('stays on one compact row and responds at mobile breakpoints', () => {
    for (const width of [24, 34, 35, 54, 55, 79, 80, 120, 240]) {
      const lines = renderAtlasFooter(FOOTER_STATE, width);
      const plain = stripAnsi(lines[0]);

      expect(lines).toHaveLength(1);
      expect(visibleWidth(lines[0]) <= width).toBeTrue();
      expect(plain.startsWith(' ')).toBeTrue();
      expect(plain.endsWith(' ')).toBeTrue();
      expect(Array.from(plain).filter((glyph) => glyph === '⛁')).toHaveLength(
        contextBarWidth(width)
      );
    }

    const wideFooter = renderAtlasFooter(FOOTER_STATE, 240)[0];
    expect(stripAnsi(renderAtlasFooter(FOOTER_STATE, 24)[0])).toContain(NERD_FONT_ICONS.cache);
    expect(stripAnsi(renderAtlasFooter(FOOTER_STATE, 55)[0])).not.toContain('…/Atlas/Config');
    expect(stripAnsi(wideFooter)).toContain('…/Atlas/Config');
    expect(visibleWidth(wideFooter) < 240).toBeTrue();
  });

  it('preserves the context and cache group while truncating long model labels', () => {
    for (const [width, meterWidth] of [
      [54, 8],
      [80, 24]
    ] as const) {
      const suffix = `${'⛁'.repeat(meterWidth)} 42%   99% `;
      for (let length = 1; length <= 80; length += 1) {
        const rendered = renderAtlasFooter(
          { ...FOOTER_STATE, model: `model-${'x'.repeat(length)}` },
          width
        )[0];
        expect(visibleWidth(rendered) <= width).toBeTrue();
        expect(stripAnsi(rendered).endsWith(suffix)).toBeTrue();
      }
    }
  });

  it('never exceeds zero-, one-, or two-column terminal widths', () => {
    expect(renderAtlasFooter(FOOTER_STATE, 0)).toEqual([]);
    expect(visibleWidth(renderAtlasFooter(FOOTER_STATE, 1)[0])).toBe(1);
    expect(visibleWidth(renderAtlasFooter(FOOTER_STATE, 2)[0])).toBe(2);
  });

  it('uses exact context and cache threshold colors in both palettes', () => {
    const palettes = [
      {
        isLight: false,
        usage: [
          [80, '241;108;117'],
          [60, '251;146;60'],
          [40, '251;191;36'],
          [39, '55;244;153']
        ],
        cache: [
          [80, '55;244;153'],
          [60, '4;209;249'],
          [40, '241;252;121'],
          [20, '247;198;127'],
          [19, '241;108;117']
        ]
      },
      {
        isLight: true,
        usage: [
          [80, '180;99;122'],
          [60, '215;130;126'],
          [40, '234;157;52'],
          [39, '40;105;131']
        ],
        cache: [
          [80, '40;105;131'],
          [60, '86;148;159'],
          [40, '234;157;52'],
          [20, '215;130;126'],
          [19, '180;99;122']
        ]
      }
    ] as const;

    for (const palette of palettes) {
      for (const [percent, rgb] of palette.usage) {
        const footer = renderAtlasFooter(
          {
            ...FOOTER_STATE,
            contextPercent: percent,
            cachePercent: null,
            isLight: palette.isLight
          },
          240
        )[0];
        expect(footer).toContain(`\x1b[38;2;${rgb}m${percent}%`);
      }
      for (const [percent, rgb] of palette.cache) {
        const footer = renderAtlasFooter(
          {
            ...FOOTER_STATE,
            contextPercent: 0,
            cachePercent: percent,
            isLight: palette.isLight
          },
          240
        )[0];
        expect(footer).toContain(`\x1b[38;2;${rgb}m${percent}%`);
      }
    }
  });

  it('uses exact gradient endpoints and dim empty-cell colors', () => {
    const darkFull = renderAtlasFooter(
      { ...FOOTER_STATE, contextPercent: 100, cachePercent: null },
      240
    )[0];
    const lightFull = renderAtlasFooter(
      {
        ...FOOTER_STATE,
        contextPercent: 100,
        cachePercent: null,
        isLight: true
      },
      240
    )[0];
    const lightEmpty = renderAtlasFooter(
      { ...FOOTER_STATE, contextPercent: 0, cachePercent: null, isLight: true },
      240
    )[0];

    expect(darkFull).toContain('\x1b[38;2;241;108;117m⛁');
    expect(lightFull).toContain('\x1b[38;2;180;99;122m⛁');
    expect(lightEmpty).toContain('\x1b[2m\x1b[38;2;223;218;217m⛁');
  });

  it('suppresses native status-line segments in authored OMP config', async () => {
    const config = Bun.YAML.parse(
      await Bun.file(new URL('../../omp/.omp/agent/config.yml', import.meta.url)).text()
    ) as {
      statusLine?: {
        preset?: string;
        leftSegments?: unknown[];
        rightSegments?: unknown[];
        showHookStatus?: boolean;
      };
    };

    expect(config.statusLine?.preset).toBe('custom');
    expect(config.statusLine?.leftSegments).toEqual([]);
    expect(config.statusLine?.rightSegments).toEqual([]);
    expect(config.statusLine?.showHookStatus).toBeTrue();
  });

  it('ships and selects a snapshot of the OMP default dark theme', async () => {
    const config = Bun.YAML.parse(
      await Bun.file(new URL('../../omp/.omp/agent/config.yml', import.meta.url)).text()
    ) as {
      theme?: { dark?: string };
      statusLine?: { separator?: string; transparent?: boolean };
    };
    const atlasDefault = (await Bun.file(
      new URL('../../omp/.omp/agent/themes/atlas-default.json', import.meta.url)
    ).json()) as Record<string, unknown>;
    const packageEntry = new URL(import.meta.resolve('@oh-my-pi/pi-coding-agent'));
    const ompDefault = (await Bun.file(
      new URL('./modes/theme/dark.json', packageEntry)
    ).json()) as Record<string, unknown>;

    expect(config.theme?.dark).toBe('atlas-default');
    expect(config.statusLine?.separator).toBe('powerline-thin');
    expect(config.statusLine?.transparent).toBeFalse();
    expect(atlasDefault).toEqual({ ...ompDefault, name: 'atlas-default' });
  });

  it('registers, refreshes, and cleans up through the supported OMP 17 widget API', async () => {
    type TestContext = {
      hasUI: boolean;
      model: { provider: string; id: string };
      getContextUsage(): { percent: number };
      sessionManager: {
        getCwd(): string;
        getUsageStatistics(): { input: number; cacheRead: number; cacheWrite: number } | undefined;
        getSessionName(): string | undefined;
      };
      ui: {
        theme: Pick<Theme, 'isLight' | 'getColorHex' | 'getBgAnsi'>;
        setStatus(key: string, text: string | undefined): void;
        setFooter(): void;
        setWidget(key: string, content: unknown, options?: { placement?: string }): void;
      };
    };
    type TestHandler = (event: unknown, ctx: TestContext) => void;
    type TopBorderProvider = (width: number) => EditorTopBorder | undefined;
    type ComponentNode = {
      children?: ComponentNode[];
      getTopBorder?(width: number): EditorTopBorder;
      setTopBorderProvider?(provider: TopBorderProvider | undefined): void;
      borderColor?: (text: string) => string;
    };
    type WidgetComponent = {
      render(width: number): string[];
      dispose?(): void;
    };
    type WidgetFactory = (
      tui: ComponentNode & { requestRender(): void },
      theme: Pick<Theme, 'isLight' | 'getColorHex' | 'getBgAnsi'>
    ) => WidgetComponent;
    const handlers = new Map<string, TestHandler>();
    let thinking = 'high';
    const omp = {
      getThinkingLevel: () => thinking,
      on(event: string, handler: TestHandler) {
        handlers.set(event, handler);
      }
    };
    let widget: { key: string; content: unknown; placement: string | undefined } | undefined;
    let mountedComponent: WidgetComponent | undefined;
    let footerUsed = false;
    let nativeStatusUsed = false;
    let contextPercent = 42;
    let usageStats: { input: number; cacheRead: number; cacheWrite: number } | undefined;
    let currentCwd = '/Users/ed/Developer/Atlas/Config';
    let sessionTitle = 'native-chip-color-rendering';
    let activeTheme = mockTheme();
    let editorTopBorderProvider: TopBorderProvider | undefined;
    const nativeStatus: ComponentNode = {
      getTopBorder: () => ({ content: 'native', width: 6 })
    };
    const nativeBorderColor = (text: string) => `native:${text}`;
    const editor: ComponentNode = {
      borderColor: nativeBorderColor,
      setTopBorderProvider(provider) {
        editorTopBorderProvider = provider;
      }
    };
    const replacementBorderColor = (text: string) => `replacement:${text}`;
    let replacementTopBorderProvider: TopBorderProvider | undefined;
    const replacementEditor: ComponentNode = {
      borderColor: replacementBorderColor,
      setTopBorderProvider(provider) {
        replacementTopBorderProvider = provider;
      }
    };
    const originalSetStatus = () => {
      nativeStatusUsed = true;
    };
    const testContext: TestContext = {
      hasUI: true,
      model: { provider: 'openai-codex', id: 'gpt-5.6-sol' },
      getContextUsage: () => ({ percent: contextPercent }),
      sessionManager: {
        getCwd: () => currentCwd,
        getSessionName: () => sessionTitle,
        getUsageStatistics: () => usageStats
      },
      ui: {
        get theme() {
          return activeTheme;
        },
        setFooter() {
          footerUsed = true;
        },
        setStatus: originalSetStatus,
        setWidget(key: string, content: unknown, options?: { placement?: string }) {
          if (content === undefined) mountedComponent?.dispose?.();
          widget = { key, content, placement: options?.placement };
        }
      }
    };
    // This test intentionally supplies only the ExtensionAPI methods used during registration.
    const extensionApi = omp as unknown as ExtensionAPI;

    await atlasStatusLine(extensionApi);
    handlers.get('session_start')?.({}, testContext);
    testContext.ui.setStatus('probe', '\x1b[35mSTATUS_COLOR_PROBE\x1b[39m');

    expect(footerUsed).toBeFalse();
    expect(nativeStatusUsed).toBeFalse();
    expect(widget?.key).toBe('atlas-status-line');
    expect(typeof widget?.content).toBe('function');
    expect(widget?.placement).toBe('belowEditor');
    if (typeof widget?.content !== 'function') throw new TypeError('Atlas widget not registered');

    let renderRequests = 0;
    const theme = mockTheme();
    const tui: ComponentNode & { requestRender(): void } = {
      children: [nativeStatus, { children: [editor] }],
      requestRender() {
        renderRequests += 1;
      }
    };
    mountedComponent = (widget.content as WidgetFactory)(tui, theme);
    const renderedLines = mountedComponent.render(240);
    const border = editorTopBorderProvider?.(236);
    const rendered = stripAnsi(border?.content ?? '');
    expect(border).toBeDefined();
    expect(visibleWidth(border?.content ?? '')).toBe(236);
    expect(border?.width).toBe(visibleWidth(border?.content ?? ''));
    // The editor's own border color is left intact so its trailing rule fills
    // out to the right corner (no suppression) — this is what makes the line span.
    expect(editor.borderColor?.('────────────────')).toBe('native:────────────────');
    expect(editor.borderColor?.('─')).toBe('native:─');
    expect(rendered).toContain('▲ Atlas v');
    expect(rendered).toContain('openai-codex/gpt-5.6-sol ⚡︎ high');
    expect(rendered).toContain(`π ${OMP_VERSION}`);
    expect(Array.from(rendered).filter((glyph) => glyph === '⛁')).toHaveLength(24);
    expect(rendered).toContain(sessionTitle);
    expect(renderedLines).toHaveLength(1);
    expect(renderedLines[0]).toContain('\x1b[35mSTATUS_COLOR_PROBE\x1b[39m');
    expect(visibleWidth(renderedLines[0])).toBe(240);
    expect(stripAnsi(renderedLines[0])).not.toContain('▲ Atlas v');
    const refreshedBorderColor = (text: string) => `refreshed:${text}`;
    editor.borderColor = refreshedBorderColor;
    mountedComponent.render(240);
    expect(editor.borderColor).toBe(refreshedBorderColor);
    editorTopBorderProvider?.(236);
    expect(editor.borderColor?.('────────────────')).toBe('refreshed:────────────────');
    expect(editor.borderColor?.('─')).toBe('refreshed:─');
    tui.children = [nativeStatus, { children: [replacementEditor] }];
    mountedComponent.render(240);
    expect(editorTopBorderProvider?.(236)).toEqual({ content: 'native', width: 6 });
    expect(replacementTopBorderProvider).toBeDefined();
    expect(editor.borderColor).toBe(refreshedBorderColor);
    expect(replacementEditor.borderColor).toBe(replacementBorderColor);
    replacementTopBorderProvider?.(236);
    expect(replacementEditor.borderColor?.('────────────────')).toBe('replacement:────────────────');
    expect(replacementEditor.borderColor?.('─')).toBe('replacement:─');
    // One render request so far: only the editor swap re-installs the provider.
    // Reassigning borderColor no longer forces a re-install (it is never wrapped).
    expect(renderRequests).toBe(1);
    testContext.model.id = 'gpt-next';
    thinking = 'max';
    contextPercent = 73;
    // Cache % is read from the session manager's cumulative usage snapshot.
    usageStats = { input: 1, cacheRead: 99, cacheWrite: 0 };
    sessionTitle = 'updated-session-title';
    activeTheme = mockTheme(true);
    handlers.get('turn_start')?.({}, testContext);
    expect(renderRequests).toBe(2);
    handlers.get('message_end')?.({}, testContext);
    handlers.get('session_compact')?.({}, testContext);
    expect(renderRequests).toBe(4);
    const refreshed = stripAnsi(replacementTopBorderProvider?.(236)?.content ?? '');
    expect(refreshed).toContain('openai-codex/gpt-next ⚡︎ max');
    expect(refreshed).toContain('73%');
    expect(refreshed).toContain(' 99%');
    expect(refreshed).toContain(sessionTitle);
    currentCwd = '/Users/ed/Developer/Elsewhere/Moved';
    const moved = stripAnsi(replacementTopBorderProvider?.(236)?.content ?? '');
    expect(moved).toContain('…/Elsewhere/Moved');
    expect(moved).toContain('no-git');

    const narrowBorder = replacementTopBorderProvider?.(32);
    expect(visibleWidth(narrowBorder?.content ?? '')).toBe(32);
    expect(stripAnsi(narrowBorder?.content ?? '')).toBe(
      ` ${'⛁'.repeat(6)} 73%   99% ─ updated-… `
    );
    const nonUiContext = { ...testContext, hasUI: false };
    handlers.get('session_switch')?.({}, nonUiContext);
    expect(testContext.ui.setStatus).toBe(originalSetStatus);
    expect(widget?.content).toBeUndefined();
    expect(replacementTopBorderProvider?.(236)).toEqual({ content: 'native', width: 6 });
    expect(replacementEditor.borderColor).toBe(replacementBorderColor);
    handlers.get('turn_end')?.({}, nonUiContext);
    expect(renderRequests).toBe(4);

    handlers.get('session_start')?.({}, testContext);
    expect(testContext.ui.setStatus).not.toBe(originalSetStatus);
    handlers.get('session_shutdown')?.({}, testContext);
    expect(testContext.ui.setStatus).toBe(originalSetStatus);
    expect(widget?.content).toBeUndefined();
  });
});
