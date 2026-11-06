// Theme flash check. System scheme is forced opposite the saved choice, both
// directions, on the landing page, CPU throttled 4x.
// A frame counts only once the browser has painted. Chrome's blank
// pre-navigation frame is not one of ours (it is pure black or pure white).
//
// When the landing hero canvas is present, this also samples text colours
// against the glyph pixels under each text box. The token contrast gate does
// not cover that.

import http from 'node:http';
import fs from 'node:fs';
import path from 'node:path';
import { chromium } from 'playwright-core';
import { PNG } from 'pngjs';

const root = path.resolve('dist');
const chrome = process.env.CHROME_PATH || '/usr/bin/google-chrome';
const dark = [11, 17, 19];
const light = [246, 245, 241];

function serve(dir) {
  return http.createServer((req, res) => {
    const url = decodeURIComponent((req.url || '/').split('?')[0]);
    let file = path.join(dir, url);
    if (url.endsWith('/')) file = path.join(file, 'index.html');
    else if (!path.extname(file)) {
      if (fs.existsSync(file + '.html')) file += '.html';
      else if (fs.existsSync(path.join(file, 'index.html'))) file = path.join(file, 'index.html');
    }
    fs.readFile(file, (err, data) => {
      if (err) {
        res.statusCode = 404;
        res.end('not found');
        return;
      }
      const types = {
        '.html': 'text/html; charset=utf-8',
        '.css': 'text/css',
        '.js': 'text/javascript',
        '.svg': 'image/svg+xml',
        '.woff2': 'font/woff2',
        '.xml': 'application/xml',
      };
      res.setHeader('content-type', types[path.extname(file)] || 'application/octet-stream');
      res.end(data);
    });
  });
}

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
const parse = (value) => {
  const v = value.trim();
  if (v[0] === '#') return [1, 3, 5].map((i) => parseInt(v.slice(i, i + 2), 16));
  return (v.match(/[\d.]+/g) || []).slice(0, 3).map(Number);
};
const dist = (a, b) => Math.hypot(a[0] - b[0], a[1] - b[1], a[2] - b[2]);

function themeOf(rgb) {
  if (dist(rgb, dark) <= 28) return 'dark';
  if (dist(rgb, light) <= 28) return 'light';
  return '';
}

const cases = [
  { system: 'light', saved: 'dark', expect: 'dark' },
  { system: 'dark', saved: 'light', expect: 'light' },
  { system: 'light', saved: null, expect: 'light' },
  { system: 'dark', saved: null, expect: 'dark' },
];

const pages = ['/'];

const server = serve(root);
await new Promise((resolve) => server.listen(0, '127.0.0.1', resolve));
const base = `http://127.0.0.1:${server.address().port}`;
const browser = await chromium.launch({
  executablePath: chrome,
  args: ['--no-sandbox', '--disable-dev-shm-usage'],
});

let failed = false;
const lines = [];

for (const pagePath of pages) {
  for (const item of cases) {
    const context = await browser.newContext({
      viewport: { width: 1280, height: 800 },
      deviceScaleFactor: 1,
      colorScheme: item.system,
    });
    const page = await context.newPage();
    const client = await context.newCDPSession(page);
    await client.send('Emulation.setCPUThrottlingRate', { rate: 4 });
    await page.addInitScript((saved) => {
      try {
        if (saved) localStorage.setItem('gap-theme', saved);
        else localStorage.removeItem('gap-theme');
      } catch (e) {}
      window.__frames = [];
      const sample = () => {
        const cs = getComputedStyle(document.documentElement);
        window.__frames.push({
          t: performance.now(),
          theme: document.documentElement.dataset.theme || '',
          bg: cs.backgroundColor,
        });
        requestAnimationFrame(sample);
      };
      requestAnimationFrame(sample);
    }, item.saved);
    await page.goto(base + pagePath, { waitUntil: 'load' });
    await page.waitForTimeout(350);
    const result = await page.evaluate(() => {
      const paint = performance.getEntriesByType('paint');
      const first = paint.find((entry) => entry.name === 'first-contentful-paint') || paint[0];
      return {
        themeAt: window.__gapThemeAt ?? null,
        firstPaint: first ? first.startTime : null,
        theme: document.documentElement.dataset.theme || '',
        frames: window.__frames,
      };
    });
    const painted = result.frames.filter((frame) => result.firstPaint != null && frame.t >= result.firstPaint);
    const wrong = painted.filter((frame) => {
      const seen = themeOf(parse(frame.bg)) || frame.theme;
      return seen !== item.expect;
    });
    const late = result.themeAt != null && result.firstPaint != null && result.themeAt > result.firstPaint;
    const label = `${pagePath} system=${item.system} saved=${item.saved || 'none'} expect=${item.expect}`;
    if (wrong.length || late || result.theme !== item.expect) failed = true;
    lines.push(
      `${wrong.length || late || result.theme !== item.expect ? 'FAIL' : 'ok  '} ${label} themeAt=${
        result.themeAt == null ? 'n/a' : result.themeAt.toFixed(0)
      }ms firstPaint=${result.firstPaint == null ? 'n/a' : result.firstPaint.toFixed(0)}ms painted=${
        painted.length
      } wrong=${wrong.length}`,
    );
    await context.close();
  }
}

const hero = await (async () => {
  const context = await browser.newContext({
    viewport: { width: 1280, height: 800 },
    deviceScaleFactor: 1,
    colorScheme: 'dark',
  });
  const page = await context.newPage();
  await page.addInitScript(() => localStorage.setItem('gap-theme', 'dark'));
  await page.goto(base + '/?still=1', { waitUntil: 'load' });
  const has = await page.locator('#field').count();
  await context.close();
  return has > 0;
})();

if (!hero) {
  lines.push('hero pixels: skipped (no canvas). The token gate does not cover glyph pixels.');
} else {
  const viewports = [
    { width: 1280, height: 800 },
    { width: 390, height: 844 },
  ];
  for (const theme of ['dark', 'light']) {
    for (const viewport of viewports) {
    const context = await browser.newContext({
      viewport,
      deviceScaleFactor: 1,
      colorScheme: theme,
    });
    const page = await context.newPage();
    await page.addInitScript((value) => localStorage.setItem('gap-theme', value), theme);
    await page.goto(base + '/?still=1', { waitUntil: 'networkidle' });
    await page.waitForTimeout(600);
    const boxes = await page.evaluate(() => {
      const acc = getComputedStyle(document.documentElement).getPropertyValue('--acc').trim();
      const acc2 = getComputedStyle(document.documentElement).getPropertyValue('--acc2').trim();
      const pick = [];
      const add = (name, el, color) => {
        if (!el) return;
        const range = document.createRange();
        range.selectNodeContents(el);
        const rects = [...range.getClientRects()];
        const r = rects.length
          ? {
              x: Math.min(...rects.map((box) => box.left)),
              y: Math.min(...rects.map((box) => box.top)),
              w: Math.max(...rects.map((box) => box.right)) - Math.min(...rects.map((box) => box.left)),
              h: Math.max(...rects.map((box) => box.bottom)) - Math.min(...rects.map((box) => box.top)),
            }
          : el.getBoundingClientRect();
        pick.push({ name, x: r.x, y: r.y, w: r.w ?? r.width, h: r.h ?? r.height, color: color || getComputedStyle(el).color });
      };
      const heading = document.querySelector('h1');
      add('headline', heading?.firstChild, heading ? getComputedStyle(heading).color : '');
      add('headline line 2', document.querySelector('h1 .l2'));
      add('agent accent', document.querySelector('h1 .l3'), acc);
      add('agent accent end', document.querySelector('h1 .l3'), acc2);
      add('eyebrow', document.querySelector('.eyebrow'));
      add('sub', document.querySelector('.sub'));
      add('command', document.querySelector('.cmd .t'));
      add('prompt', document.querySelector('.cmd .d'));
      add('copy', document.querySelector('.cmd .cp'));
      add('hint', document.querySelector('.hint'));
      add('button', document.querySelector('.btn'));
      // Keep panels and the shade. Clear each element's own colour, not only
      // an inherited one, so the sample is the backdrop under the glyphs.
      document.querySelectorAll('.copy, .copy *, .top, .top *, .foot, .foot *').forEach((el) => {
        el.style.color = 'transparent';
        el.style.webkitTextFillColor = 'transparent';
      });
      const accentLine = document.querySelector('h1 .l3');
      if (accentLine) accentLine.style.background = 'none';
      return pick;
    });
    const png = PNG.sync.read(await page.screenshot());
    let worst = { name: '', ratio: 99 };
    for (const box of boxes) {
      const x0 = Math.max(0, Math.floor(box.x));
      const y0 = Math.max(0, Math.floor(box.y));
      const x1 = Math.min(png.width, Math.ceil(box.x + box.w));
      const y1 = Math.min(png.height, Math.ceil(box.y + box.h));
      const fg = parse(box.color);
      let low = 99;
      for (let y = y0; y < y1; y += 2) {
        for (let x = x0; x < x1; x += 2) {
          const i = (y * png.width + x) * 4;
          const ratio = contrast(fg, [png.data[i], png.data[i + 1], png.data[i + 2]]);
          if (ratio < low) low = ratio;
        }
      }
      if (low < worst.ratio) worst = { name: box.name, ratio: low };
      if (low < 4.5) failed = true;
      lines.push(`  ${low >= 4.5 ? 'ok  ' : 'FAIL'} hero ${theme} ${viewport.width} ${box.name} ${low.toFixed(2)}`);
    }
    lines.push(`hero ${theme} ${viewport.width} lowest ${worst.ratio.toFixed(2)} (${worst.name})`);
    await context.close();
    }
  }
}

await browser.close();
server.close();
console.log(lines.join('\n'));
console.log(failed ? 'FAIL' : 'PASS');
if (failed) process.exit(1);
