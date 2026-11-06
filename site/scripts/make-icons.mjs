// Regenerates public/favicon.ico (16, 32 and 48px PNG frames) and
// public/apple-touch-icon.png (180px) from public/favicon.svg.
// Run after editing the SVG:  node scripts/make-icons.mjs
// Corners stay transparent. iOS paints transparent pixels black, which is
// close to the site's dark #0b1113, so the apple icon is not filled.

import fs from 'node:fs';
import { chromium } from 'playwright-core';

const chrome = process.env.CHROME_PATH || '/usr/bin/google-chrome';
const svg = fs.readFileSync('public/favicon.svg');
const uri = 'data:image/svg+xml;base64,' + svg.toString('base64');

const browser = await chromium.launch({
  executablePath: chrome,
  args: ['--no-sandbox', '--disable-dev-shm-usage'],
});

async function render(size) {
  const context = await browser.newContext({ viewport: { width: size, height: size }, deviceScaleFactor: 1 });
  const page = await context.newPage();
  await page.setContent(
    `<style>html,body{margin:0;background:transparent}img{display:block;width:${size}px;height:${size}px}</style><img id="f" src="${uri}">`,
  );
  await page.waitForFunction(() => {
    const img = document.getElementById('f');
    return img.complete && img.naturalWidth > 0;
  });
  const png = await page.screenshot({ omitBackground: true, clip: { x: 0, y: 0, width: size, height: size } });
  await context.close();
  return png;
}

const sizes = [16, 32, 48];
const frames = [];
for (const size of sizes) frames.push({ size, png: await render(size) });

const header = Buffer.alloc(6);
header.writeUInt16LE(0, 0);
header.writeUInt16LE(1, 2);
header.writeUInt16LE(frames.length, 4);
const dir = Buffer.alloc(16 * frames.length);
let offset = 6 + dir.length;
frames.forEach((frame, i) => {
  const e = i * 16;
  dir[e] = frame.size;
  dir[e + 1] = frame.size;
  dir.writeUInt16LE(1, e + 4); // colour planes
  dir.writeUInt16LE(32, e + 6); // bits per pixel
  dir.writeUInt32LE(frame.png.length, e + 8);
  dir.writeUInt32LE(offset, e + 12);
  offset += frame.png.length;
});
fs.writeFileSync('public/favicon.ico', Buffer.concat([header, dir, ...frames.map((f) => f.png)]));
fs.writeFileSync('public/apple-touch-icon.png', await render(180));
await browser.close();
console.log('wrote public/favicon.ico (16, 32, 48) and public/apple-touch-icon.png (180)');
