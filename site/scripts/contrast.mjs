import { chromium } from 'playwright-core';
import { serveSite } from './serve.mjs';

const palettes = {
  dark: {
    '--bg': '#0b1113', '--s1': '#111a1d', '--s2': '#172327',
    '--text': '#ededed', '--t2': '#a1a1a1', '--t3': '#8c8c8c', '--l2': '#b4b4b4',
    '--callout': '#d0d0d0', '--btn-bg': '#ededed', '--btn-fg': '#000000', '--btn-hv': '#ffffff',
    '--acc': '#2dd4bf', '--acc2': '#5eead4', '--mid': '#3fe0cb',
  },
  light: {
    '--bg': '#f6f5f1', '--s1': '#fcfbf8', '--s2': '#ecebe6',
    '--text': '#111111', '--t2': '#4a4a4a', '--t3': '#5f5f5f', '--l2': '#5a5a5a',
    '--callout': '#2c2c2c', '--btn-bg': '#111111', '--btn-fg': '#f6f5f1', '--btn-hv': '#000000',
    '--acc': '#0f766e', '--acc2': '#0d6e66', '--mid': '#0b6a62',
  },
};
const lin = (v) => {
  const s = v / 255;
  return s <= 0.04045 ? s / 12.92 : ((s + 0.055) / 1.055) ** 2.4;
};
const lum = (c) => 0.2126 * lin(c[0]) + 0.7152 * lin(c[1]) + 0.0722 * lin(c[2]);
const contrast = (a, b) => {
  const x = lum(a);
  const y = lum(b);
  return (Math.max(x, y) + 0.05) / (Math.min(x, y) + 0.05);
};
const over = (fg, bg) => fg.slice(0, 3).map((v, i) => v * fg[3] / 255 + bg[i] * (1 - fg[3] / 255));
const equal = (a, b) => a.every((v, i) => v === b[i]);
const { base, close } = await serveSite();
let browser;
let failed = false;
try {
  browser = await chromium.launch({
    executablePath: process.env.CHROME_PATH || '/usr/bin/google-chrome',
    args: ['--no-sandbox', '--disable-dev-shm-usage'],
  });
  for (const theme of ['dark', 'light']) {
    for (const javaScriptEnabled of [true, false]) {
      const context = await browser.newContext({
        colorScheme: javaScriptEnabled ? (theme === 'dark' ? 'light' : 'dark') : theme,
        javaScriptEnabled,
      });
      if (javaScriptEnabled) await context.addInitScript((value) => localStorage.setItem('gap-theme', value), theme);
      for (const pagePath of ['/', '/404.html']) {
        const page = await context.newPage();
        await page.goto(base + pagePath, { waitUntil: 'load' });
        const got = await page.evaluate((palette) => {
          const probe = document.createElement('span');
          document.body.append(probe);
          const canvas = document.createElement('canvas');
          canvas.width = canvas.height = 1;
          const ctx = canvas.getContext('2d');
          const color = (value) => {
            probe.style.color = value;
            ctx.clearRect(0, 0, 1, 1);
            ctx.fillStyle = getComputedStyle(probe).color;
            ctx.fillRect(0, 0, 1, 1);
            return Array.from(ctx.getImageData(0, 0, 1, 1).data);
          };
          const actual = Object.fromEntries(Object.keys(palette).map((key) => [key, color(`var(${key})`)]));
          const expected = Object.fromEntries(Object.entries(palette).map(([key, value]) => [key, color(value)]));
          const rgb = Object.fromEntries(['bg', 's1', 'ink', 'acc', 'acc2', 'hot'].map((key) => [key, color(`rgb(var(--${key}-rgb))`)]));
          const body = getComputedStyle(document.body);
          const bodyColors = { background: color(body.backgroundColor), text: color(body.color) };
          const tints = { note: color('rgba(var(--acc-rgb), 0.07)'), tip: color('rgba(var(--acc2-rgb), 0.08)') };
          const hairlines = Object.fromEntries(['--b1', '--b2', '--b3', '--b4'].map((key) => [key, color(`var(${key})`)]));
          probe.remove();
          return { actual, expected, rgb, bodyColors, tints, hairlines };
        }, palettes[theme]);
        const label = `${pagePath} ${theme}${javaScriptEnabled ? '' : ' no-js'}`;
        const v = got.actual;
        const rows = [];
        const check = (ok, message) => {
          if (!ok) failed = true;
          console.log(`  ${ok ? 'ok  ' : 'FAIL'} ${message}`);
        };
        console.log(label);
        for (const key of Object.keys(v)) check(equal(v[key], got.expected[key]), `${key} matches GAP palette`);
        for (const [key, token] of Object.entries({ bg: '--bg', s1: '--s1', ink: '--text', acc: '--acc', acc2: '--acc2' })) {
          check(equal(got.rgb[key], v[token]), `${key}-rgb matches resolved ${token}`);
        }
        check(equal(got.rgb.hot, [214, 255, 246, 255]), 'hero highlight matches GAP palette');
        check(equal(got.bodyColors.background, got.expected['--bg']), 'rendered body background matches GAP palette');
        check(equal(got.bodyColors.text, got.expected['--text']), 'rendered body text matches GAP palette');
        const add = (kind, label, ratio) => rows.push({ kind, label, ratio });
        for (const fg of ['--text', '--t2', '--t3', '--l2', '--acc', '--acc2']) {
          for (const surface of ['--bg', '--s1', '--s2']) add('text', `${fg} on ${surface}`, contrast(v[fg], v[surface]));
        }
        add('text', '--btn-fg on --btn-bg', contrast(v['--btn-fg'], v['--btn-bg']));
        const note = over(got.tints.note, v['--bg']);
        const tip = over(got.tints.tip, v['--bg']);
        add('text', '--callout on note tint', contrast(v['--callout'], note));
        add('text', '--callout on tip tint', contrast(v['--callout'], tip));
        add('text', '--acc on note tint', contrast(v['--acc'], note));
        add('text', '--acc2 on tip tint', contrast(v['--acc2'], tip));
        for (const fg of ['--acc', '--acc2', '--mid']) {
          for (const surface of ['--bg', '--s1', '--s2']) add('line', `${fg} on ${surface}`, contrast(v[fg], v[surface]));
        }
        add('line', 'note border on its tint', contrast(v['--acc'], note));
        add('line', 'tip border on its tint', contrast(v['--acc2'], tip));
        for (const row of rows) check(row.ratio >= (row.kind === 'text' ? 4.5 : 3), `${row.kind} ${row.ratio.toFixed(2)} ${row.label}`);
        console.log('  decorative hairlines: ' + Object.entries(got.hairlines).map(([key, c]) => `${key} ${contrast(over(c, v['--bg']), v['--bg']).toFixed(2)}`).join(', '));
        await page.close();
      }
      await context.close();
    }
  }
} finally {
  await browser?.close();
  await close();
}
console.log(failed ? 'FAIL' : 'PASS');
if (failed) process.exit(1);
