import { describe, expect, it } from 'bun:test';
import type { SessionEntry } from '@oh-my-pi/pi-agent-core/compaction';
import { visibleWidth } from '@oh-my-pi/pi-tui';
import {
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

const stripAnsi = (text: string): string => Bun.stripANSI(text);

describe('Atlas OMP status line', () => {
  it('shows only the current directory and its parent', () => {
    expect(formatCurrentDirectory('/Users/ed/Developer/Atlas/Config')).toBe('…/Atlas/Config');
    expect(formatCurrentDirectory('/Atlas/Config')).toBe('…/Atlas/Config');
    expect(formatCurrentDirectory('/')).toBe('/');
  });

  it('removes controls from labels while preserving trusted SGR status color', () => {
    expect(sanitizeLabel('open\n\t\x1b[31mai\x1b[0m\u0007')).toBe('open ai');
    expect(sanitizeStatusText('\x1b[31mred\x1b[39m\n\u0007ready')).toBe(
      '\x1b[31mred\x1b[39m ready'
    );
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
        atlasVersion: '7.1.1',
        provider: 'openai',
        model: 'gpt-5.6',
        thinking: 'high',
        ompVersion: '16.5.2',
        cwd: '/Users/ed/Developer/Atlas/Config',
        branch: 'atlas/fixes',
        contextPercent: 42,
        cachePercent: 99,
        isLight: false,
        statuses: new Map([['fixture', '\x1b[35mcolored\x1b[39m']]),
      },
      240
    );
    const footer = stripAnsi(lines[0]);

    expect(footer).toContain('▲ Atlas v7.1.1');
    expect(footer).toContain('openai/gpt-5.6 ⚡︎ high');
    expect(footer).toContain('π 16.5.2');
    expect(footer).toContain('…/Atlas/Config');
    expect(footer).toContain(' atlas/fixes');
    expect(footer).toContain('42%');
    expect(footer).toContain(' 99%');
    expect(Array.from(footer).filter(glyph => glyph === '⛁')).toHaveLength(24);
    expect(lines[1]).toBe('\x1b[35mcolored\x1b[39m');
    expect(stripAnsi(lines[1])).toBe('colored');
  });
});
