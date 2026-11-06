// Internal link check. Reads the built site in dist/ (run `npm run build`
// first). Every same-site href and src in every page, and every destination in
// dist/_redirects, must reach a built file; a #fragment must name an id on that
// page. A link may go through a _redirects rule, as Cloudflare Pages applies
// those before serving files. Cross-site links are not fetched.

import fs from 'node:fs';
import path from 'node:path';

const root = path.resolve('dist');
const origin = 'https://gap.local';

function walk(dir) {
  return fs.readdirSync(dir, { withFileTypes: true }).flatMap((e) => {
    const p = path.join(dir, e.name);
    return e.isDirectory() ? walk(p) : [p];
  });
}

/** The built file a URL path serves, or undefined. */
function fileFor(pathname) {
  const p = decodeURIComponent(pathname);
  for (const c of [p, path.join(p, 'index.html'), `${p}.html`]) {
    const f = path.join(root, c);
    if (f.startsWith(root) && fs.existsSync(f) && fs.statSync(f).isFile()) return f;
  }
  return undefined;
}

const redirects = new Map(
  fs
    .readFileSync(path.join(root, '_redirects'), 'utf8')
    .split('\n')
    .map((l) => l.trim().split(/\s+/))
    .filter((f) => f.length >= 2 && !f[0].startsWith('#'))
    .map(([from, to]) => [from, to]),
);

const ids = new Map();
function idsOf(file) {
  if (!ids.has(file)) {
    const html = fs.readFileSync(file, 'utf8');
    ids.set(file, new Set([...html.matchAll(/\sid="([^"]+)"/g)].map((m) => m[1])));
  }
  return ids.get(file);
}

const failures = [];
let checked = 0;

function check(href, from) {
  const url = new URL(href, new URL(from, origin));
  if (url.origin !== origin) return;
  checked++;
  let target = url;
  const rule = redirects.get(url.pathname);
  if (rule) target = new URL(rule, origin);
  const file = fileFor(target.pathname);
  if (!file) return failures.push(`${from}: ${href} -> no file for ${target.pathname}`);
  const hash = decodeURIComponent((target.hash || url.hash).slice(1));
  if (hash && file.endsWith('.html') && !idsOf(file).has(hash)) failures.push(`${from}: ${href} -> no #${hash} on ${target.pathname}`);
}

const pages = walk(root).filter((f) => f.endsWith('.html'));
for (const file of pages) {
  const from = '/' + path.relative(root, file).split(path.sep).join('/').replace(/(^|\/)index\.html$/, '$1');
  const html = fs.readFileSync(file, 'utf8');
  for (const m of html.matchAll(/<(?:a|link|img|script|source)\b[^>]*?\s(?:href|src)="([^"]+)"/g)) {
    const href = m[1].replaceAll('&amp;', '&');
    if (/^(mailto:|tel:|data:|javascript:)/.test(href)) continue;
    check(href, from);
  }
}
for (const [from, to] of redirects) if (to.startsWith('/')) check(to, `_redirects ${from}`);

for (const f of failures) console.log(`FAIL ${f}`);
console.log(`${pages.length} pages, ${checked} internal links, ${failures.length} broken`);
console.log(failures.length ? 'FAIL' : 'PASS');
process.exit(failures.length ? 1 : 0);
