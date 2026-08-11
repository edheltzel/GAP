// @ts-nocheck
// This repository executes the contract with Bun and does not install a local TypeScript toolchain.
import { describe, expect, it } from 'bun:test';
import { VERSION } from '@earendil-works/pi-coding-agent';
import atlasStatusLine from '../../pi/.pi/agent/extensions/atlas-status-line';

function stripAnsi(text: string): string {
  return Bun.stripANSI(text);
}

describe('Atlas Pi status line', () => {
  it('renders Pi identity before model metadata without retired Atlas identity', () => {
    const handlers = new Map<string, (...args: unknown[]) => unknown>();
    const pi = {
      getThinkingLevel: () => 'high',
      on: (event: string, handler: (...args: unknown[]) => unknown) => {
        handlers.set(event, handler);
      }
    };
    let footer: { render(width: number): string[] } | undefined;
    const footerData = {
      getExtensionStatuses: () => new Map(),
      getGitBranch: () => 'fix/reconcile',
      onBranchChange: () => () => {}
    };
    const ctx = {
      getContextUsage: () => ({ percent: 42 }),
      mode: 'tui',
      model: { provider: 'openai', id: 'gpt-5.6' },
      sessionManager: {
        getBranch: () => [],
        getCwd: () => '/tmp/Atlas/Config'
      },
      ui: {
        setFooter: (factory: (...args: unknown[]) => typeof footer) => {
          footer = factory(
            { requestRender: () => {} },
            { name: 'eldritch' },
            footerData
          );
        }
      }
    };

    atlasStatusLine(pi);
    handlers.get('session_start')?.({}, ctx);

    const rendered = stripAnsi(footer?.render(120)[0] ?? '');
    expect(rendered).toContain(`π ${VERSION} · openai/gpt-5.6 ⚡︎ high`);
    expect(rendered).not.toContain('▲ Atlas');
    expect(rendered).not.toContain('Atlas v');
  });
});
