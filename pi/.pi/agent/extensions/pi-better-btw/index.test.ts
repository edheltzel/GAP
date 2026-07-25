import { describe, expect, it } from 'bun:test';
import type { ExtensionCommandContext } from '@earendil-works/pi-coding-agent';
import { visibleWidth, type KeybindingsManager, type TUI } from '@earendil-works/pi-tui';
import { fitLineToWidth, GhostOverlayComponent } from './index';

function createOverlay(): GhostOverlayComponent {
  const tui = {
    terminal: { columns: 120, rows: 40 },
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
    'gpt-5.6-sol',
    () => {},
    () => {},
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
    expect(lines[1]).not.toContain('ctrl+s');
    expect(lines[2]).toBe(`│${' '.repeat(78)}│`);
    expect(lines.at(-2)).toContain('up/down scroll │ ctrl+s hide │ esc close');
  });

  it('clips and pads ANSI content to an exact width', () => {
    const fitted = fitLineToWidth('\x1b[31m123456789\x1b[39m', 6);

    expect(visibleWidth(fitted)).toBe(6);
  });
});
