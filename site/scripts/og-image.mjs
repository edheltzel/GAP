// Social preview check. Reads the built site in dist/ (run `npm run build`
// first). For the landing page, the docs index and one deeper docs page:
// exactly one of each image tag (og:image, og:image:width, og:image:height,
// og:image:alt, twitter:image, twitter:image:alt, twitter:card), one shared
// absolute https image URL on every page, twitter:card is summary_large_image,
// and alt text is present and matches the site wording. dist/og.png must exist,
// be exactly 1200x630 (IHDR header bytes, no decoder) and be under 300 KB.

import fs from 'node:fs';
import path from 'node:path';

const root = path.resolve('dist');
const pages = ['/'];
const alt = 'GAP: Global Agent Policies. One set of rules for every coding agent.';
const keys = ['og:image', 'og:image:width', 'og:image:height', 'og:image:alt', 'twitter:image', 'twitter:image:alt', 'twitter:card'];
const maxBytes = 300 * 1024;

let failed = false;
const lines = [];
const report = (ok, text) => {
  if (!ok) failed = true;
  lines.push(`${ok ? 'ok  ' : 'FAIL'} ${text}`);
};

// Every <meta> in the page head, as { key, content }. The key is the property
// (Open Graph) or the name (Twitter, description).
function metas(html) {
  const head = html.slice(0, html.indexOf('</head>'));
  const out = [];
  for (const tag of head.match(/<meta\b[^>]*>/g) || []) {
    const attr = (name) => (tag.match(new RegExp(`\\s${name}="([^"]*)"`)) || [])[1];
    const key = attr('property') ?? attr('name');
    if (key) out.push({ key, content: attr('content') ?? '' });
  }
  return out;
}

const urls = {};
for (const page of pages) {
  const file = path.join(root, page, 'index.html');
  if (!fs.existsSync(file)) {
    report(false, `${page} not built (${path.relative('.', file)} missing)`);
    continue;
  }
  const found = metas(fs.readFileSync(file, 'utf8'));
  const values = {};
  for (const key of keys) {
    const hits = found.filter((m) => m.key === key);
    report(hits.length === 1, `${page} has ${hits.length} ${key} (need exactly 1)`);
    values[key] = hits.length === 1 ? hits[0].content : undefined;
  }
  report(values['twitter:card'] === 'summary_large_image', `${page} twitter:card=${values['twitter:card']} (need summary_large_image)`);
  report(values['og:image:width'] === '1200', `${page} og:image:width=${values['og:image:width']} (need 1200)`);
  report(values['og:image:height'] === '630', `${page} og:image:height=${values['og:image:height']} (need 630)`);
  report(values['og:image:alt'] === alt, `${page} og:image:alt=${JSON.stringify(values['og:image:alt'])}`);
  report(values['twitter:image:alt'] === alt, `${page} twitter:image:alt=${JSON.stringify(values['twitter:image:alt'])}`);
  for (const key of ['og:image', 'twitter:image']) {
    const value = values[key];
    report(!!value && /^https:\/\/[^/]+\/\S+$/.test(value), `${page} ${key}=${value} (need absolute https URL)`);
    if (value) urls[`${page} ${key}`] = value;
  }
  // The image lives on the same origin as the page's own og:url.
  const ogUrl = found.find((m) => m.key === 'og:url')?.content;
  if (ogUrl && values['og:image']) {
    const origin = new URL(ogUrl).origin;
    report(values['og:image'] === `${origin}/og.png`, `${page} og:image is ${origin}/og.png`);
  }
}

// One URL for every tag on every page.
const distinct = [...new Set(Object.values(urls))];
report(distinct.length === 1 && Object.keys(urls).length === pages.length * 2, `image URL identical on all pages and both tags (${distinct.length} distinct: ${distinct.join(', ') || 'none'})`);

// The file itself.
const png = path.join(root, 'og.png');
if (!fs.existsSync(png)) {
  report(false, 'dist/og.png is missing');
} else {
  const bytes = fs.readFileSync(png);
  const signature = bytes.subarray(0, 8).equals(Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]));
  const ihdr = bytes.subarray(12, 16).toString('latin1') === 'IHDR';
  report(signature && ihdr, 'dist/og.png has a PNG signature and IHDR chunk');
  const width = bytes.readUInt32BE(16);
  const height = bytes.readUInt32BE(20);
  report(width === 1200 && height === 630, `dist/og.png is ${width}x${height} (need 1200x630)`);
  report(bytes.length < maxBytes, `dist/og.png is ${(bytes.length / 1024).toFixed(0)} KB (need < 300 KB)`);
}

console.log(lines.join('\n'));
console.log(failed ? 'FAIL' : 'PASS');
if (failed) process.exit(1);
