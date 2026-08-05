import { afterEach, describe, expect, it } from 'bun:test';
import {
  chmodSync,
  mkdirSync,
  mkdtempSync,
  readFileSync,
  rmSync,
  writeFileSync
} from 'node:fs';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';

const REPO_ROOT = resolve(import.meta.dir, '../..');
const INSTALLER = join(REPO_ROOT, 'install-omp-plugins.sh');
const temporaryDirectories: string[] = [];

type RunOptions = {
  marketplaceList?: string;
  pluginJson?: object;
  failMatch?: string;
  dryRun?: boolean;
};

function createFixture(): { directory: string; bin: string; log: string } {
  const directory = mkdtempSync(join(tmpdir(), 'atlas-omp-plugins-'));
  temporaryDirectories.push(directory);
  const bin = join(directory, 'bin');
  const log = join(directory, 'omp.log');
  mkdirSync(bin);
  writeFileSync(log, '');
  const omp = join(bin, 'omp');
  writeFileSync(
    omp,
    `#!/bin/bash
printf '%s\\n' "$*" >> "$FAKE_OMP_LOG"
case "$*" in
  "plugin marketplace list") printf '%s' "$FAKE_MARKETPLACE_LIST" ;;
  "plugin list --json") printf '%s' "$FAKE_PLUGIN_JSON" ;;
esac
if [ -n "\${FAKE_FAIL_MATCH:-}" ] && [[ "$*" == *"$FAKE_FAIL_MATCH"* ]]; then
  exit 17
fi
`
  );
  chmodSync(omp, 0o755);
  return { directory, bin, log };
}

function runInstaller(manifest: string, options: RunOptions = {}) {
  const fixture = createFixture();
  const manifestPath = join(fixture.directory, 'manifest.txt');
  writeFileSync(manifestPath, manifest);
  const result = Bun.spawnSync({
    cmd: [
      '/bin/bash',
      INSTALLER,
      ...(options.dryRun === false ? [] : ['--dry-run']),
      manifestPath
    ],
    cwd: REPO_ROOT,
    env: {
      ...process.env,
      PATH: `${fixture.bin}:${process.env.PATH ?? ''}`,
      FAKE_OMP_LOG: fixture.log,
      FAKE_MARKETPLACE_LIST: options.marketplaceList ?? 'Configured Marketplaces:\n',
      FAKE_PLUGIN_JSON: JSON.stringify(options.pluginJson ?? { npm: [], marketplace: [] }),
      FAKE_FAIL_MATCH: options.failMatch ?? ''
    }
  });
  return {
    exitCode: result.exitCode,
    stdout: result.stdout.toString(),
    stderr: result.stderr.toString(),
    calls: readFileSync(fixture.log, 'utf8').trim().split('\n').filter(Boolean)
  };
}

function marketplaceList(name: string, source: string): string {
  return `Configured Marketplaces:\n\n  ${name}  ${source}\n`;
}

function pluginState(id: string, scope: 'user' | 'project'): object {
  return {
    npm: [],
    marketplace: [
      {
        id,
        scope,
        entries: [{ scope, version: '1.0.0' }]
      }
    ]
  };
}

afterEach(() => {
  for (const directory of temporaryDirectories.splice(0)) {
    rmSync(directory, { recursive: true, force: true });
  }
});

describe('OMP marketplace plugin installer', () => {
  it('updates the declared marketplace and upgrades an existing user plugin', () => {
    const result = runInstaller(
      'EveryInc/compound-engineering-plugin=compound-engineering@compound-engineering-plugin\n',
      {
        marketplaceList: marketplaceList(
          'compound-engineering-plugin',
          'EveryInc/compound-engineering-plugin'
        ),
        pluginJson: pluginState(
          'compound-engineering@compound-engineering-plugin',
          'user'
        )
      }
    );

    expect(result.exitCode).toBe(0);
    expect(result.stdout).toContain('omp plugin marketplace update compound-engineering-plugin');
    expect(result.stdout).toContain(
      'omp plugin upgrade --scope user compound-engineering@compound-engineering-plugin'
    );
  });

  it('installs user scope when only a project-scoped copy exists', () => {
    const result = runInstaller('example/source=plugin@example-marketplace\n', {
      marketplaceList: marketplaceList('example-marketplace', 'example/source'),
      pluginJson: pluginState('plugin@example-marketplace', 'project')
    });

    expect(result.exitCode).toBe(0);
    expect(result.stdout).toContain('omp plugin install --scope user plugin@example-marketplace');
    expect(result.stdout).not.toContain('omp plugin upgrade --scope user plugin@example-marketplace');
  });

  it('rejects a changed marketplace source before planning plugin changes', () => {
    const result = runInstaller('new/source=plugin@example-marketplace\n', {
      marketplaceList: marketplaceList('example-marketplace', 'old/source')
    });

    expect(result.exitCode).not.toBe(0);
    expect(result.stderr).toContain(
      'Marketplace "example-marketplace" is registered from "old/source", not "new/source"'
    );
    expect(result.stdout).not.toContain('omp plugin marketplace update');
    expect(result.stdout).not.toContain('omp plugin upgrade');
    expect(result.stdout).not.toContain('omp plugin install');
  });

  it('validates the complete manifest before invoking OMP', () => {
    const result = runInstaller(
      'example/source=first@example-marketplace\nmalformed-entry\n'
    );

    expect(result.exitCode).not.toBe(0);
    expect(result.stderr).toContain('Invalid manifest entry: malformed-entry');
    expect(result.calls).toEqual([]);
  });

  it('rejects extra manifest delimiters before invoking OMP', () => {
    for (const malformed of [
      'example/source=second=plugin@example-marketplace',
      'example/source=second@plugin@example@marketplace'
    ]) {
      const result = runInstaller(
        `example/source=first@example-marketplace\n${malformed}\n`
      );

      expect(result.exitCode).not.toBe(0);
      expect(result.stderr).toContain(`Invalid manifest entry: ${malformed}`);
      expect(result.calls).toEqual([]);
    }
  });

  it('adds a shared marketplace once before planning both plugin installs', () => {
    const result = runInstaller(
      'example/source=first@example-marketplace\nexample/source=second@example-marketplace\n'
    );

    expect(result.exitCode).toBe(0);
    expect(result.stdout.match(/marketplace add example\/source/g)).toHaveLength(1);
    expect(result.stdout).not.toContain('marketplace update example-marketplace');
    expect(result.stdout).toContain('install --scope user first@example-marketplace');
    expect(result.stdout).toContain('install --scope user second@example-marketplace');
  });

  it('propagates OMP failures without attempting the plugin operation', () => {
    const result = runInstaller('example/source=plugin@example-marketplace\n', {
      marketplaceList: marketplaceList('example-marketplace', 'example/source'),
      failMatch: 'plugin marketplace update',
      dryRun: false
    });

    expect(result.exitCode).toBe(17);
    expect(result.calls).toContain('plugin marketplace update example-marketplace');
    expect(result.calls.some((call) => call.includes('plugin upgrade'))).toBeFalse();
    expect(result.calls.some((call) => call.includes('plugin install'))).toBeFalse();
  });

  it('keeps OMP provisioning after the stow body in scoped and aggregate recipes', () => {
    for (const recipe of ['omp', 'update', 'link']) {
      const result = Bun.spawnSync({ cmd: ['just', '--dry-run', recipe], cwd: REPO_ROOT });
      const output = result.stdout.toString() + result.stderr.toString();
      expect(result.exitCode).toBe(0);
      expect(output).toContain('--no-folding');
      const stowBody = output.lastIndexOf('stow ');
      expect(stowBody).toBeGreaterThanOrEqual(0);
      expect(output.indexOf('./install-omp-plugins.sh')).toBeGreaterThan(stowBody);
    }
  });
});
