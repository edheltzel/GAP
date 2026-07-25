import { describe, expect, it } from 'bun:test';
import type { ExtensionCommandContext } from '@earendil-works/pi-coding-agent';
import { visibleWidth, type KeybindingsManager, type TUI } from '@earendil-works/pi-tui';
import { fitLineToWidth, formatModelStatus, GhostOverlayComponent } from './index';

function createOverlay(onHide: () => void = () => {}): GhostOverlayComponent {
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
