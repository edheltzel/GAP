// Hero spin check. The globe rotates slowly and steadily; ?still, ?t= and
// prefers-reduced-motion stay completely still.
//
//   (a) motion allowed: two frames a fixed time apart differ in the disc interior
//       (a rotating surface) while the rim ring keeps its silhouette (rotation,
//       not drift). The interior must change much more than the flicker-only
//       baseline changes it, measured in the same run.
//   (b) ?still, ?t= and reduced-motion: frames identical across time, and
//       byte-identical to the output of the baseline hero script (the hero before
//       the spin, scripts/fixtures/hero-baseline.js) served on the same page.
//   (c) per-frame cost on a 4x CPU throttle: main-thread task time per #field
//       draw, median of 5 pairs (pair order alternates so order bias cancels),
//       no worse than the baseline script in the same run (RATIO_MAX is a ratio,
//       not a fixed millisecond limit), plus fillText calls per draw (a
//       deterministic count) against the baseline.
//       The cost gate guards against per-frame work creeping into the spin (trig
//       or allocation inside the draw loop). Timing noise on a shared box is
//       about +-10%, so RATIO_MAX sits above it and still catches a real regression.
//
// The baseline is a frozen copy of the built hero script from before the spin
// (shader-01 head 675b2cc). The check swaps it into a second server over the same
// dist, so both sides run the same markup, CSS, browser and machine load.
//
// Usage: node scripts/hero-spin.mjs [distDir]   (default ./dist)
// Env: CHROME_PATH, RUNS (default 5), ONLY=cost to run just (c) while tuning.

import http from 'node:http';
import fs from 'node:fs';
import path from 'node:path';
import { chromium } from 'playwright-core';

const root = path.resolve(process.argv[2] || 'dist');
const chrome = process.env.CHROME_PATH || '/usr/bin/google-chrome';
const runs = Number(process.env.RUNS || 5);
const baselineSrc = fs.readFileSync(new URL('./fixtures/hero-baseline.js', import.meta.url));

const SAMPLE_GAP_MS = 3000; // fixed time between the two motion frames
const RIM_SAME = 0.2; // rim ring change must stay at or under this (fraction of ink)
const INTERIOR_GAIN = 3; // interior change must be this many times the flicker-only change
const INTERIOR_MIN = 0.05; // ...and at least this fraction of the interior ink
const RATIO_MAX = 1.25; // frame cost vs baseline, median of pairs (above timing noise)
const TEXT_MAX = 1.1; // fillText calls per draw vs baseline (deterministic count)
const CELL_W = 6;
const CELL_H = 10;

const heroFile = (() => {
  const dir = path.join(root, '_astro');
  const hit = fs.readdirSync(dir).find((n) => n.endsWith('.js') && fs.readFileSync(path.join(dir, n), 'utf8').includes('hero stage missing'));
  if (!hit) throw new Error('hero script not found in ' + dir);
  return '/_astro/' + hit;
})();

function serve(dir, override) {
  return http.createServer((req, res) => {
    const url = decodeURIComponent((req.url || '/').split('?')[0]);
    if (override && url === heroFile) {
      res.setHeader('content-type', 'text/javascript');
      res.setHeader('cache-control', 'no-store');
      res.end(override);
      return;
    }
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
      };
      res.setHeader('content-type', types[path.extname(file)] || 'application/octet-stream');
      res.setHeader('cache-control', 'no-store');
      res.end(data);
    });
  });
}

const listen = async (srv) => {
  await new Promise((r) => srv.listen(0, '127.0.0.1', r));
  return `http://127.0.0.1:${srv.address().port}`;
};

async function open(browser, base, { theme = 'dark', reduced = false, viewport = { width: 1280, height: 800 }, query = '', init } = {}) {
  const context = await browser.newContext({
    viewport,
    deviceScaleFactor: 1,
    colorScheme: theme,
    reducedMotion: reduced ? 'reduce' : 'no-preference',
  });
  const page = await context.newPage();
  await page.addInitScript((t) => localStorage.setItem('gap-theme', t), theme);
  if (init) await page.addInitScript(init);
  await page.goto(base + '/' + query, { waitUntil: 'load' });
  return { context, page };
}

// Mean alpha per 6x10 cell of #field, plus the stage geometry.
const cellMeans = () => {
  const f = document.getElementById('field');
  const st = document.getElementById('stage');
  const w = st.clientWidth;
  const h = st.clientHeight;
  const tmp = document.createElement('canvas');
  tmp.width = f.width;
  tmp.height = f.height;
  const c = tmp.getContext('2d', { willReadFrequently: true });
  c.drawImage(f, 0, 0);
  const d = c.getImageData(0, 0, f.width, f.height).data;
  const cols = Math.ceil(w / 6);
  const rows = Math.ceil(h / 10);
  const out = new Float32Array(cols * rows);
  for (let y = 0; y < f.height; y++) {
    const r = Math.min(rows - 1, Math.floor(y / 10));
    for (let x = 0; x < f.width; x++) out[r * cols + Math.min(cols - 1, Math.floor(x / 6))] += d[(y * f.width + x) * 4 + 3] / 255 / 60;
  }
  return { w, h, cols, rows, cells: Array.from(out) };
};

function geometry(w, h, cols, rows) {
  const narrow = w < 860;
  const radius = narrow ? w * 0.36 : Math.min(h * 0.4, w * 0.25);
  const cx = narrow ? w * 0.6 : w * 0.72;
  const cy = narrow ? h * 0.7 : h * 0.5;
  const rim = [];
  const inner = [];
  for (let r = 0; r < rows; r++) {
    for (let c = 0; c < cols; c++) {
      const rr = Math.hypot(((c + 0.5) * CELL_W - cx) / radius, ((r + 0.5) * CELL_H - cy) / radius);
      if (rr > 0.92 && rr < 1.08) rim.push(r * cols + c);
      else if (rr < 0.85) inner.push(r * cols + c);
    }
  }
  return { rim, inner };
}

const delta = (a, b, idx) => {
  let diff = 0;
  let ink = 0;
  for (const i of idx) {
    diff += Math.abs(a[i] - b[i]);
    ink += Math.max(a[i], b[i]);
  }
  return ink > 0 ? diff / ink : 0;
};

async function twoFrames(browser, base, query = '') {
  const { context, page } = await open(browser, base, { query });
  await page.waitForFunction(() => performance.now() > 3000, null, { timeout: 30000, polling: 100 });
  const a = await page.evaluate(cellMeans);
  await page.waitForTimeout(SAMPLE_GAP_MS);
  const b = await page.evaluate(cellMeans);
  await context.close();
  const g = geometry(a.w, a.h, a.cols, a.rows);
  return { rim: delta(a.cells, b.cells, g.rim), inner: delta(a.cells, b.cells, g.inner), g, a, b };
}

async function stillShot(browser, base, opts) {
  const { context, page } = await open(browser, base, opts);
  await page.waitForLoadState('networkidle');
  await page.waitForTimeout(800);
  const one = await page.locator('#field').screenshot();
  await page.waitForTimeout(3000);
  const two = await page.locator('#field').screenshot();
  await context.close();
  return { one, two };
}

// Frame cost: main-thread task time per #field draw, after the reveal.
async function frameCost(browser, base) {
  const { context, page } = await open(browser, base, {
    init: () => {
      window.__draws = 0;
      window.__js = 0;
      window.__text = 0;
      let t0 = 0;
      const clear = CanvasRenderingContext2D.prototype.clearRect;
      const rect = CanvasRenderingContext2D.prototype.fillRect;
      const text = CanvasRenderingContext2D.prototype.fillText;
      CanvasRenderingContext2D.prototype.clearRect = function (...a) {
        if (this.canvas.id === 'field') {
          window.__draws++;
          t0 = performance.now();
        }
        return clear.apply(this, a);
      };
      CanvasRenderingContext2D.prototype.fillText = function (...a) {
        if (this.canvas.id === 'field') window.__text++;
        return text.apply(this, a);
      };
      CanvasRenderingContext2D.prototype.fillRect = function (...a) {
        const r = rect.apply(this, a);
        if (this.canvas.id === 'field' && t0) {
          window.__js += performance.now() - t0;
          t0 = 0;
        }
        return r;
      };
    },
  });
  const client = await context.newCDPSession(page);
  await client.send('Performance.enable');
  await client.send('Emulation.setCPUThrottlingRate', { rate: 4 });
  const task = async () => (await client.send('Performance.getMetrics')).metrics.find((m) => m.name === 'TaskDuration').value;
  await page.waitForFunction(() => performance.now() > 2500, null, { timeout: 60000, polling: 100 });
  const t0 = await task();
  const s0 = await page.evaluate(() => [window.__draws, window.__js, window.__text]);
  await page.waitForTimeout(4000);
  const t1 = await task();
  const s1 = await page.evaluate(() => [window.__draws, window.__js, window.__text]);
  await context.close();
  const n = s1[0] - s0[0];
  return { draws: n, taskMs: ((t1 - t0) * 1000) / n, jsMs: (s1[1] - s0[1]) / n, textPerDraw: (s1[2] - s0[2]) / n };
}

const med = (a) => {
  const v = [...a].sort((x, y) => x - y);
  return v[Math.floor((v.length - 1) / 2)];
};

const serverNew = serve(root);
const serverOld = serve(root, baselineSrc);
const baseNew = await listen(serverNew);
const baseOld = await listen(serverOld);
const browser = await chromium.launch({ executablePath: chrome, args: ['--no-sandbox', '--disable-dev-shm-usage'] });

const lines = [];
let failed = false;
const check = (ok, text) => {
  if (!ok) failed = true;
  lines.push(`${ok ? 'ok  ' : 'FAIL'} ${text}`);
};
const pct = (v) => (v * 100).toFixed(1) + '%';

const onlyCost = process.env.ONLY === 'cost';

// (a) motion
if (!onlyCost) {
const mNew = await twoFrames(browser, baseNew);
const mOld = await twoFrames(browser, baseOld);
lines.push(`motion ${SAMPLE_GAP_MS}ms apart: rim change ${pct(mNew.rim)}, interior change ${pct(mNew.inner)} (baseline script: rim ${pct(mOld.rim)}, interior ${pct(mOld.inner)})`);
check(mNew.rim <= RIM_SAME, `(a) rim ring keeps its silhouette: change ${pct(mNew.rim)} <= ${pct(RIM_SAME)}`);
check(
  mNew.inner >= INTERIOR_MIN && mNew.inner >= INTERIOR_GAIN * mOld.inner,
  `(a) disc interior rotates: change ${pct(mNew.inner)} >= max(${pct(INTERIOR_MIN)}, ${INTERIOR_GAIN}x flicker-only ${pct(mOld.inner)})`,
);

// (b) still, ?t=, reduced motion
const stills = [
  ['?still=1 dark 1280', { query: '?still=1' }],
  ['?still=1 light 1280', { query: '?still=1', theme: 'light' }],
  ['?still=1 dark 390', { query: '?still=1', viewport: { width: 390, height: 844 } }],
  ['?t=2 dark 1280', { query: '?t=2' }],
  ['reduced-motion dark 1280', { reduced: true }],
  ['reduced-motion light 390', { reduced: true, theme: 'light', viewport: { width: 390, height: 844 } }],
];
for (const [name, opts] of stills) {
  const n = await stillShot(browser, baseNew, opts);
  const o = await stillShot(browser, baseOld, opts);
  check(n.one.equals(n.two), `(b) ${name}: identical across 3s`);
  check(n.one.equals(o.one), `(b) ${name}: identical to the baseline hero output`);
}

}

// (c) frame cost, pairs with alternating order (new-first, then baseline-first, ...)
const costs = [];
for (let i = 0; i < runs; i++) {
  let n;
  let o;
  if (i % 2 === 0) {
    n = await frameCost(browser, baseNew);
    o = await frameCost(browser, baseOld);
  } else {
    o = await frameCost(browser, baseOld);
    n = await frameCost(browser, baseNew);
  }
  costs.push({ n, o });
}
const ratioTask = med(costs.map((c) => c.n.taskMs / c.o.taskMs));
const ratioJs = med(costs.map((c) => c.n.jsMs / c.o.jsMs));
const textRatio = med(costs.map((c) => c.n.textPerDraw / c.o.textPerDraw));
const f1 = (v) => v.toFixed(1);
lines.push(`frame cost at 4x CPU, per draw (this / baseline): task ${costs.map((c) => `${f1(c.n.taskMs)}/${f1(c.o.taskMs)}`).join(' ')} ms; js ${costs.map((c) => `${f1(c.n.jsMs)}/${f1(c.o.jsMs)}`).join(' ')} ms; draws ${costs.map((c) => `${c.n.draws}/${c.o.draws}`).join(' ')}; fillText/draw ${costs.map((c) => `${Math.round(c.n.textPerDraw)}/${Math.round(c.o.textPerDraw)}`).join(' ')}`);
check(ratioTask <= RATIO_MAX, `(c) main-thread time per draw is ${ratioTask.toFixed(2)}x the baseline (median of ${runs}; median ${f1(med(costs.map((c) => c.n.taskMs)))} vs ${f1(med(costs.map((c) => c.o.taskMs)))} ms) <= ${RATIO_MAX}x`);
check(ratioJs <= RATIO_MAX, `(c) script time per draw is ${ratioJs.toFixed(2)}x the baseline (median ${f1(med(costs.map((c) => c.n.jsMs)))} vs ${f1(med(costs.map((c) => c.o.jsMs)))} ms) <= ${RATIO_MAX}x`);
check(textRatio <= TEXT_MAX, `(c) fillText calls per draw are ${textRatio.toFixed(2)}x the baseline (median ${Math.round(med(costs.map((c) => c.n.textPerDraw)))} vs ${Math.round(med(costs.map((c) => c.o.textPerDraw)))}) <= ${TEXT_MAX}x`);

await browser.close();
serverNew.close();
serverOld.close();
console.log(lines.join('\n'));
console.log(failed ? 'FAIL' : 'PASS');
if (failed) process.exit(1);
