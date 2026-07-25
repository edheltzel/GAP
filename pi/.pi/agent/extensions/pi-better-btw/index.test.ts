import { describe, expect, it } from 'bun:test';
import type { ExtensionCommandContext } from '@earendil-works/pi-coding-agent';
import { visibleWidth, type KeybindingsManager, type TUI } from '@earendil-works/pi-tui';
import {
  fitLineToWidth,
  formatModelStatus,
  GhostOverlayComponent,
  OVERLAY_MIN_HEIGHT,
  resolveOverlayHeight,
  resolveOverlayMaxHeight,
} from './index';

function createOverlay(
  onHide: () => void = () => {},
  terminal: { columns: number; rows: number } = { columns: 120, rows: 40 }
): GhostOverlayComponent {
  const tui = {
    terminal,
    requestRender() {},
  } as unknown as TUI;
  const theme = {
    fg: (_color: string, text: string) => text,
    bold: (text: string) => text,
  } as unknown as ExtensionCommandContext['ui']['theme'];

  return new GhostOverlayComponent(
    tui,
    theme,
    {} as KeybindingsManager,
    '/Users/ed/Developer/Atlas/Config',
    () => 'gpt-5.6-sol (high)',
    () => {},
    onHide,
    () => {}
  );
}

describe('GhostOverlayComponent layout', () => {
  it('renders every framed row at the requested width', () => {
    const lines = createOverlay().render(80);

    expect(lines.every(line => visibleWidth(line) === 80)).toBe(true);
    expect(lines[0]).toBe(`╭${'─'.repeat(78)}╮`);
    expect(lines.at(-1)).toBe(`╰${'─'.repeat(78)}╯`);
  });

  it('keeps a quiet title area and moves controls to the bottom', () => {
    const lines = createOverlay().render(80);

    expect(lines[1]).toContain('better pi');
    expect(lines[1]).not.toContain('ctrl+h');
    expect(lines[2]).toBe(`│${' '.repeat(78)}│`);
    expect(lines.at(-2)).toContain('esc close • up/down scroll • ctrl+h hide');
  });

  it('hides on ctrl+h and leaves backspace to the input', () => {
    let hidden = 0;
    const overlay = createOverlay(() => {
      hidden += 1;
    });

    overlay.handleInput('\x7f'); // backspace must not be swallowed as ctrl+h
    expect(hidden).toBe(0);

    overlay.handleInput('\x08'); // ctrl+h
    expect(hidden).toBe(1);
  });

  it('shows the model and thinking level on the top row, not the footer', () => {
    const lines = createOverlay().render(80);
    const footer = lines.at(-5) ?? '';

    expect(lines[1]).toContain('gpt-5.6-sol (high)');
    expect(footer).not.toContain('gpt-5.6-sol');
  });

  it('omits the thinking level only when it is off', () => {
    expect(formatModelStatus('gpt-5.6-sol', 'high')).toBe('gpt-5.6-sol (high)');
    expect(formatModelStatus('gpt-5.6-sol', 'off')).toBe('gpt-5.6-sol');
  });

  it('clips and pads ANSI content to an exact width', () => {
    const fitted = fitLineToWidth('\x1b[31m123456789\x1b[39m', 6);

    expect(visibleWidth(fitted)).toBe(6);
  });
});

describe('overlay box', () => {
  // Pi composites only the rows that fit the box `overlayOptions` reserves and
  // silently slices the rest, so a frame taller than the cap loses its bottom
  // border and the session shows through where the frame should be.
  const SIZES = [
    { columns: 120, rows: 40 },
    { columns: 120, rows: 36 },
    { columns: 100, rows: 30 },
    { columns: 100, rows: 24 },
    { columns: 80, rows: 20 },
    { columns: 80, rows: 18 },
  ];

  for (const size of SIZES) {
    it(`fits the reserved box at ${size.columns}x${size.rows}`, () => {
      const width = Math.floor(size.columns * 0.9);
      const lines = createOverlay(() => {}, size).render(width);

      expect(lines.length).toBeLessThanOrEqual(resolveOverlayMaxHeight(size.rows));
      expect(lines.length).toBe(resolveOverlayHeight(size.rows));
      expect(lines.every(line => visibleWidth(line) === width)).toBe(true);
      expect(lines.at(-1)).toBe(`╰${'─'.repeat(width - 2)}╯`);
    });

    it(`fits the reserved box at ${size.columns}x${size.rows} with a long input`, () => {
      const width = Math.floor(size.columns * 0.9);
      const overlay = createOverlay(() => {}, size);
      for (const char of 'x'.repeat(width * 3)) overlay.handleInput(char);
      const lines = overlay.render(width);

      expect(lines.length).toBeLessThanOrEqual(resolveOverlayMaxHeight(size.rows));
      expect(lines.at(-1)).toBe(`╰${'─'.repeat(width - 2)}╯`);
    });
  }

  it('never asks for more rows than Pi reserves', () => {
    for (let rows = 8; rows <= 200; rows++) {
      expect(resolveOverlayHeight(rows)).toBeLessThanOrEqual(resolveOverlayMaxHeight(rows));
    }
  });

  it('draws a whole frame at every terminal height that can hold one', () => {
    for (let rows = 8; rows <= 200; rows++) {
      if (resolveOverlayMaxHeight(rows) < OVERLAY_MIN_HEIGHT) continue;
      const lines = createOverlay(() => {}, { columns: 100, rows }).render(90);

      expect(lines.length).toBe(resolveOverlayHeight(rows));
      expect(lines[0]).toBe(`\u256d${'\u2500'.repeat(88)}\u256e`);
      expect(lines.at(-1)).toBe(`\u2570${'\u2500'.repeat(88)}\u256f`);
    }
  });
});
