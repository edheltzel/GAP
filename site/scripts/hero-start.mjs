// Hero start check. Cold cache, network 1.6 Mbps / 150 ms RTT, CPU 4x.
//
// Fails when the landing hero canvas is slow to start or slow to settle, or
// when the hero work delays first paint:
//   - first contentful paint within FCP_OVER_CONTROL_MS of a control load of the
//     same page with its external scripts and preloads removed, measured in the
//     same run so machine load cancels out (the code before the hero change sat
//     ~40ms over the control; preloading both fonts ~230ms, mono only ~190ms,
//     because the fonts compete with the CSS for bandwidth). FCP_MS is only a
//     loose absolute backstop: an absolute 650ms budget fails under CPU load.
//     The allowance is max(100ms, 25% of the control) because the page's own
//     script cost grows with CPU load too (measured +60..+175ms at load 13 on
//     8 cores, still under 25%; mono-only idle was +36%, both fonts +44%).
//   - first non-empty frame on #field within DCL_SLACK_MS of DOMContentLoaded
//     (the old code waited for fonts.ready, ~350ms later). FIRST_FRAME_MS is
//     only a loose absolute backstop; the order-based asserts are the gate.
//   - first frame with real glyphs (a fillText call, not only the haze gradient)
//     within DOT_MS after the first frame
//   - canvas at or above 98% of its final ink (stable from then on) within
//     SETTLE_MS after the first frame
// It also reads the network log: every woff2 the page fetches must be fetched
// exactly once, and any <link rel=preload as=font> must carry crossorigin and
// match a fetched URL (a missing crossorigin makes the browser download twice).
// The page ships no font preloads today: they delayed FCP on this link.
//
// Usage: node scripts/hero-start.mjs [distDir]   (default ./dist)
// Env: CHROME_PATH, RUNS (default 5). Reports medians.

import http from 'node:http';
import fs from 'node:fs';
import path from 'node:path';
import { chromium } from 'playwright-core';

const root = path.resolve(process.argv[2] || 'dist');
const chrome = process.env.CHROME_PATH || '/usr/bin/google-chrome';
const runs = Number(process.env.RUNS || 5);
const DCL_SLACK_MS = 150;
const FCP_OVER_CONTROL_MS = 100;
const FCP_OVER_CONTROL_PCT = 0.25;
const FCP_MS = 1500;
const FIRST_FRAME_MS = 3000;
const DOT_MS = 400;
const SETTLE_MS = 2000;
const WINDOW_MS = 5500;
const FCP_EXTRA_PAIRS = 6; // short page/control pairs, FCP only, to damp noise under load

function serve(dir) {
  return http.createServer((req, res) => {
    const url = decodeURIComponent((req.url || '/').split('?')[0]);
    if (url === '/__control') {
      // Same page with the external scripts and preloads removed: what first paint costs with no hero work.
      const html = fs
        .readFileSync(path.join(dir, 'index.html'), 'utf8')
        .replace(/<script[^>]*\ssrc=[^>]*><\/script>/g, '')
        .replace(/<link[^>]*rel="preload"[^>]*>/g, '');
      res.setHeader('content-type', 'text/html; charset=utf-8');
      res.end(html);
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

const sampler = () => {
  window.__s = [];
  window.__fill = 0;
  const ft = CanvasRenderingContext2D.prototype.fillText;
  CanvasRenderingContext2D.prototype.fillText = function (...args) {
    window.__fill++;
    return ft.apply(this, args);
  };
  const tmp = document.createElement('canvas');
  tmp.width = 80;
  tmp.height = 50;
  const tc = tmp.getContext('2d', { willReadFrequently: true });
  const poll = () => {
    const f = document.getElementById('field');
    if (f && f.width > 0) {
      tc.clearRect(0, 0, 80, 50);
      tc.drawImage(f, 0, 0, 80, 50);
      const d = tc.getImageData(0, 0, 80, 50).data;
      let sum = 0;
      let nz = 0;
      for (let i = 3; i < d.length; i += 4) {
        sum += d[i];
        if (d[i] > 0) nz++;
      }
      window.__s.push([performance.now(), sum / 4000, nz, window.__fill]);
    }
    requestAnimationFrame(poll);
  };
  requestAnimationFrame(poll);
};

async function once(browser, base, route = '/', windowMs = WINDOW_MS) {
  const context = await browser.newContext({
    viewport: { width: 1280, height: 800 },
    deviceScaleFactor: 1,
    colorScheme: 'dark',
  });
  const page = await context.newPage();
  const client = await context.newCDPSession(page);
  await client.send('Network.enable');
  await client.send('Network.setCacheDisabled', { cacheDisabled: true });
  await client.send('Network.emulateNetworkConditions', {
    offline: false,
    latency: 150,
    downloadThroughput: 1.6e6 / 8,
    uploadThroughput: 750e3 / 8,
  });
  await client.send('Emulation.setCPUThrottlingRate', { rate: 4 });
  const fonts = [];
  client.on('Network.requestWillBeSent', (e) => {
    if (/\.woff2(\?|$)/.test(e.request.url)) fonts.push(e.request.url);
  });
  const warnings = [];
  page.on('console', (m) => {
    if (/preload/i.test(m.text())) warnings.push(m.text());
  });
  await page.addInitScript(() => localStorage.setItem('gap-theme', 'dark'));
  await page.addInitScript(sampler);
  await page.goto(base + route, { waitUntil: 'commit' });
  await page.waitForFunction((ms) => performance.now() > ms, WINDOW_MS, { timeout: 60000, polling: 100 });
  const data = await page.evaluate(() => ({
    s: window.__s,
    dcl: performance.getEntriesByType('navigation')[0].domContentLoadedEventEnd,
    fcp: performance.getEntriesByType('paint').find((e) => e.name === 'first-contentful-paint')?.startTime ?? null,
    preloads: [...document.querySelectorAll('link[rel=preload]')].map((l) => ({
      as: l.getAttribute('as'),
      type: l.getAttribute('type'),
      crossorigin: l.hasAttribute('crossorigin'),
      href: l.href,
    })),
  }));
  await context.close();
  const s = data.s;
  const tail = s.slice(-10);
  const final = tail.reduce((a, x) => a + x[1], 0) / tail.length;
  const first = s.find((x) => x[2] > 0);
  const firstDot = s.find((x) => x[2] > 0 && x[3] > 0);
  let stable = null;
  for (let i = s.length - 1; i >= 0; i--) {
    if (s[i][1] < final * 0.98) {
      stable = s[i + 1] ? s[i + 1][0] : null;
      break;
    }
    if (i === 0) stable = s[0][0];
  }
  return {
    dcl: data.dcl,
    fcp: data.fcp,
    first: first?.[0] ?? null,
    firstDot: firstDot?.[0] ?? null,
    stable,
    final,
    fonts,
    preloads: data.preloads,
    warnings,
  };
}

const med = (a) => {
  const v = a.filter((x) => x != null).sort((x, y) => x - y);
  if (v.length < a.length) return null;
  return v[Math.floor((v.length - 1) / 2)];
};

const server = serve(root);
await new Promise((resolve) => server.listen(0, '127.0.0.1', resolve));
const base = `http://127.0.0.1:${server.address().port}`;
const browser = await chromium.launch({ executablePath: chrome, args: ['--no-sandbox', '--disable-dev-shm-usage'] });

const results = [];
const controls = [];
for (let i = 0; i < runs; i++) {
  results.push(await once(browser, base));
  controls.push(await once(browser, base, '/__control', 1500));
}
// Extra short pairs for the FCP comparison only (page and control, alternating).
const pairs = results.map((r, i) => [r.fcp, controls[i].fcp]);
for (let i = 0; i < FCP_EXTRA_PAIRS; i++) {
  const p = await once(browser, base, '/', 1500);
  const c = await once(browser, base, '/__control', 1500);
  pairs.push([p.fcp, c.fcp]);
}
await browser.close();
server.close();

const lines = [];
let failed = false;
const check = (ok, text) => {
  if (!ok) failed = true;
  lines.push(`${ok ? 'ok  ' : 'FAIL'} ${text}`);
};
const f = (v) => (v == null ? 'never' : v.toFixed(0) + 'ms');

const first = med(results.map((r) => r.first));
const fcp = med(pairs.map((p) => p[0]));
const dcl = med(results.map((r) => r.dcl));
const afterDcl = med(results.map((r) => (r.first != null ? r.first - r.dcl : null)));
const dot = med(results.map((r) => (r.firstDot != null && r.first != null ? r.firstDot - r.first : null)));
const settle = med(results.map((r) => (r.stable != null && r.first != null ? r.stable - r.first : null)));
lines.push(`runs=${runs} fcp=${results.map((r) => f(r.fcp)).join('/')} first=${results.map((r) => f(r.first)).join('/')} dcl=${results.map((r) => f(r.dcl)).join('/')} dot-first=${results.map((r) => (r.firstDot != null && r.first != null ? f(r.firstDot - r.first) : 'never')).join('/')} stable98-first=${results.map((r) => (r.stable != null && r.first != null ? f(r.stable - r.first) : 'never')).join('/')}`);
const ctl = med(pairs.map((p) => p[1]));
const over = med(pairs.map((p) => (p[0] != null && p[1] != null ? p[0] - p[1] : null)));
lines.push(`fcp pairs page/control (${pairs.length}): ${pairs.map((p) => `${f(p[0])}/${f(p[1])}`).join(' ')}`);
const overBudget = Math.max(FCP_OVER_CONTROL_MS, FCP_OVER_CONTROL_PCT * (ctl ?? 0));
check(over != null && over <= overBudget, `first contentful paint ${f(fcp)} is ${f(over)} over the no-hero control (${f(ctl)}) <= ${f(overBudget)} (max of ${FCP_OVER_CONTROL_MS}ms and ${FCP_OVER_CONTROL_PCT * 100}% of control)`);
check(fcp != null && fcp <= FCP_MS, `first contentful paint ${f(fcp)} <= ${FCP_MS}ms after navigation start (loose backstop)`);
check(afterDcl != null && afterDcl <= DCL_SLACK_MS, `first non-empty frame ${f(first)} is ${f(afterDcl)} after DOMContentLoaded (${f(dcl)}) <= ${DCL_SLACK_MS}ms`);
check(first != null && first <= FIRST_FRAME_MS, `first non-empty frame ${f(first)} <= ${FIRST_FRAME_MS}ms after navigation start`);
check(dot != null && dot <= DOT_MS, `first glyph frame ${f(dot)} after first frame <= ${DOT_MS}ms`);
check(settle != null && settle <= SETTLE_MS, `canvas >=98% of final ${f(settle)} after first frame <= ${SETTLE_MS}ms`);

// Network log: runs are independent, a font problem in any run fails.
const r0 = results[0];
const pre = r0.preloads.filter((p) => p.as === 'font');
for (const p of pre) {
  check(p.crossorigin && p.type === 'font/woff2', `preload ${path.basename(p.href)} has crossorigin and type=font/woff2`);
}
for (const [i, r] of results.entries()) {
  const preUrls = new Set(r.preloads.filter((p) => p.as === 'font').map((p) => p.href));
  const counts = new Map();
  for (const u of r.fonts) counts.set(u, (counts.get(u) || 0) + 1);
  for (const u of preUrls) {
    const n = counts.get(u) || 0;
    check(n === 1, `run ${i + 1} preloaded ${path.basename(u)} fetched ${n}x (want exactly 1)`);
  }
  for (const [u, n] of counts) {
    if (n !== 1) check(false, `run ${i + 1} font ${path.basename(u)} fetched ${n}x (want exactly 1)`);
  }
  check([...counts.values()].every((n) => n === 1), `run ${i + 1} ${counts.size} woff2 file(s) fetched, each exactly once`);
  check(r.warnings.length === 0, `run ${i + 1} no preload console warnings${r.warnings.length ? ': ' + r.warnings[0] : ''}`);
}

console.log(lines.join('\n'));
console.log(failed ? 'FAIL' : 'PASS');
if (failed) process.exit(1);
