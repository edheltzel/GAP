import { describe, expect, it } from 'bun:test';
import type { SessionEntry } from '@oh-my-pi/pi-agent-core/compaction';
import { VERSION as OMP_VERSION, type ExtensionAPI } from '@oh-my-pi/pi-coding-agent';
import { visibleWidth } from '@oh-my-pi/pi-tui';
import atlasStatusLine, {
  calculateCacheHitPercent,
  contextBarWidth,
  contextFilledCells,
  formatCurrentDirectory,
  latestCacheHitPercent,
  layoutStatusItems,
  NERD_FONT_ICONS,
  renderAtlasFooter,
  sanitizeLabel,
  sanitizeStatusText,
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
  isLight: false,
};

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
      'broken\x1b[31',
    ].join(' ');

    expect(sanitizeStatusText(unsafe)).toBe('clear hidden cursor link clip reset broken');
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

  it('uses only the latest successful request for cache display', () => {
    const cached = {
      type: 'message',
      message: {
        role: 'assistant',
        stopReason: 'stop',
        usage: { input: 1, cacheRead: 99, cacheWrite: 0 },
      },
    };
    const inputOnly = {
      type: 'message',
      message: {
        role: 'assistant',
        stopReason: 'stop',
        usage: { input: 100, cacheRead: 0, cacheWrite: 0 },
      },
    };
    const failed = {
      type: 'message',
      message: {
        role: 'assistant',
        stopReason: 'error',
        usage: { input: 100, cacheRead: 0, cacheWrite: 0 },
      },
    };

    expect(latestCacheHitPercent([cached, inputOnly] as SessionEntry[])).toBeNull();
    expect(latestCacheHitPercent([cached, failed] as SessionEntry[])).toBe(99);
  });

  it('uses the exact canonical glyph codepoints', () => {
    expect(NERD_FONT_ICONS.branch.codePointAt(0)).toBe(0xf418);
    expect(NERD_FONT_ICONS.cache.codePointAt(0)).toBe(0xf49b);
    expect(Array.from('⚡︎', glyph => glyph.codePointAt(0))).toEqual([0x26a1, 0xfe0e]);
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
        statuses: new Map([['fixture', '\x1b[35mcolored\x1b[39m']]),
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
    expect(Array.from(footer).filter(glyph => glyph === '⛁')).toHaveLength(24);
    expect(lines[1]).toBe('\x1b[35mcolored\x1b[39m');
    expect(stripAnsi(lines[1])).toBe('colored');
    expect(lines[0]).toContain('\x1b[38;2;164;140;242m▲ Atlas');
    expect(lines[0]).toContain('\x1b[38;2;232;97;32mπ');
    expect(lines[0]).toContain('\x1b[38;2;4;209;249m…/Atlas/Config');
    expect(lines[0]).toContain('\x1b[38;2;242;101;181m');
    expect(lines[0]).toContain('\x1b[2m\x1b[38;2;50;52;73m⛁');
    expect(lines[0]).toContain('\x1b[38;2;235;250;250mv7.1.1');
    expect(lines[0]).toContain('\x1b[38;2;112;129;208mopenai/');
    expect(lines[0]).toContain('\x1b[38;2;50;52;73m·');
  });

  it('uses the canonical Rosé Pine Dawn light palette', () => {
    const footer = renderAtlasFooter(
      { ...FOOTER_STATE, isLight: true },
      240
    )[0];

    expect(footer).toContain('\x1b[38;2;144;122;169m▲ Atlas');
    expect(footer).toContain('\x1b[38;2;234;157;52mπ');
    expect(footer).toContain('\x1b[38;2;86;148;159m…/Atlas/Config');
    expect(footer).toContain('\x1b[38;2;215;130;126m');
    expect(footer).toContain('\x1b[38;2;87;82;121mv7.1.1');
    expect(footer).toContain('\x1b[38;2;121;117;147mopenai/');
    expect(footer).toContain('\x1b[38;2;223;218;217m·');
  });

  it('uses exact context and cache threshold colors in both palettes', () => {
    const palettes = [
      {
        isLight: false,
        usage: [
          [80, '241;108;117'],
          [60, '251;146;60'],
          [40, '251;191;36'],
          [39, '55;244;153'],
        ],
        cache: [
          [80, '55;244;153'],
          [60, '4;209;249'],
          [40, '241;252;121'],
          [20, '247;198;127'],
          [19, '241;108;117'],
        ],
      },
      {
        isLight: true,
        usage: [
          [80, '180;99;122'],
          [60, '215;130;126'],
          [40, '234;157;52'],
          [39, '40;105;131'],
        ],
        cache: [
          [80, '40;105;131'],
          [60, '86;148;159'],
          [40, '234;157;52'],
          [20, '215;130;126'],
          [19, '180;99;122'],
        ],
      },
    ] as const;

    for (const palette of palettes) {
      for (const [percent, rgb] of palette.usage) {
        const footer = renderAtlasFooter(
          { ...FOOTER_STATE, contextPercent: percent, cachePercent: null, isLight: palette.isLight },
          240
        )[0];
        expect(footer).toContain(`\x1b[38;2;${rgb}m${percent}%`);
      }
      for (const [percent, rgb] of palette.cache) {
        const footer = renderAtlasFooter(
          { ...FOOTER_STATE, contextPercent: 0, cachePercent: percent, isLight: palette.isLight },
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
      { ...FOOTER_STATE, contextPercent: 100, cachePercent: null, isLight: true },
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

  it('registers, refreshes, and cleans up through the supported OMP 17 widget API', async () => {
    type TestContext = {
      hasUI: boolean;
      model: { provider: string; id: string };
      getContextUsage(): { percent: number };
      sessionManager: { getCwd(): string; getBranch(): SessionEntry[] };
      ui: {
        setStatus(key: string, text: string | undefined): void;
        setFooter(): void;
        setWidget(key: string, content: unknown, options?: { placement?: string }): void;
      };
    };
    type TestHandler = (event: unknown, ctx: TestContext) => void;
    type WidgetComponent = { render(width: number): string[]; dispose?(): void };
    type WidgetFactory = (
      tui: { requestRender(): void },
      theme: { isLight: boolean }
    ) => WidgetComponent;
    const handlers = new Map<string, TestHandler>();
    let thinking = 'high';
    const omp = {
      getThinkingLevel: () => thinking,
      on(event: string, handler: TestHandler) {
        handlers.set(event, handler);
      },
    };
    let widget: { key: string; content: unknown; placement: string | undefined } | undefined;
    let mountedComponent: WidgetComponent | undefined;
    let footerUsed = false;
    let nativeStatusUsed = false;
    let contextPercent = 42;
    let branchEntries: SessionEntry[] = [];
    const originalSetStatus = () => {
      nativeStatusUsed = true;
    };
    const testContext: TestContext = {
      hasUI: true,
      model: { provider: 'openai-codex', id: 'gpt-5.6-sol' },
      getContextUsage: () => ({ percent: contextPercent }),
      sessionManager: {
        getCwd: () => '/Users/ed/Developer/Atlas/Config',
        getBranch: () => branchEntries,
      },
      ui: {
        setFooter() {
          footerUsed = true;
        },
        setStatus: originalSetStatus,
        setWidget(key: string, content: unknown, options?: { placement?: string }) {
          if (content === undefined) mountedComponent?.dispose?.();
          widget = { key, content, placement: options?.placement };
        },
      },
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
    expect(widget?.placement).toBe('aboveEditor');
    if (typeof widget?.content !== 'function') throw new TypeError('Atlas widget not registered');

    let renderRequests = 0;
    const theme = { isLight: false };
    mountedComponent = (widget.content as WidgetFactory)(
      {
        requestRender() {
          renderRequests += 1;
        },
      },
      theme
    );
    const renderedLines = mountedComponent.render(240);
    const rendered = stripAnsi(renderedLines[0] ?? '');
    expect(rendered).toContain('▲ Atlas v');
    expect(rendered).toContain('openai-codex/gpt-5.6-sol ⚡︎ high');
    expect(rendered).toContain(`π ${OMP_VERSION}`);
    expect(Array.from(rendered).filter(glyph => glyph === '⛁')).toHaveLength(24);
    expect(renderedLines[1]).toBe('\x1b[35mSTATUS_COLOR_PROBE\x1b[39m');

    testContext.model.id = 'gpt-next';
    thinking = 'max';
    contextPercent = 73;
    branchEntries = [
      {
        type: 'message',
        message: {
          role: 'assistant',
          stopReason: 'stop',
          usage: { input: 1, cacheRead: 99, cacheWrite: 0 },
        },
      } as SessionEntry,
    ];
    theme.isLight = true;
    handlers.get('turn_start')?.({}, testContext);
    expect(renderRequests).toBe(1);
    handlers.get('message_end')?.({}, testContext);
    handlers.get('session_compact')?.({}, testContext);
    expect(renderRequests).toBe(3);
    const refreshed = stripAnsi(mountedComponent.render(240)[0] ?? '');
    expect(refreshed).toContain('openai-codex/gpt-next ⚡︎ max');
    expect(refreshed).toContain('73%');
    expect(refreshed).toContain(' 99%');

    const nonUiContext = { ...testContext, hasUI: false };
    handlers.get('session_switch')?.({}, nonUiContext);
    expect(testContext.ui.setStatus).toBe(originalSetStatus);
    expect(widget?.content).toBeUndefined();
    handlers.get('turn_end')?.({}, nonUiContext);
    expect(renderRequests).toBe(3);

    handlers.get('session_start')?.({}, testContext);
    expect(testContext.ui.setStatus).not.toBe(originalSetStatus);
    handlers.get('session_shutdown')?.({}, testContext);
    expect(testContext.ui.setStatus).toBe(originalSetStatus);
    expect(widget?.content).toBeUndefined();
  });
});
