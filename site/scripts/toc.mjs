import assert from 'node:assert/strict';
import { chromium } from 'playwright-core';
import { serveSite } from './serve.mjs';

const { base, close } = await serveSite();
let browser;
try {
  browser = await chromium.launch({
    executablePath: process.env.CHROME_PATH || '/usr/bin/google-chrome',
    args: ['--no-sandbox', '--disable-dev-shm-usage'],
  });
  for (const viewport of [{ width: 390, height: 844 }, { width: 844, height: 390 }, { width: 1440, height: 900 }]) {
    const page = await browser.newPage({ viewport, reducedMotion: 'reduce' });
    await page.goto(base + '/docs/');
    await page.evaluate(() => document.fonts.ready);
    const expectSection = async (slug) => {
      await page.waitForFunction((slug) => {
        const mobile = document.querySelector('[data-nb-mobile-toc-select]').value;
        const desktop = document.querySelector('[data-nb-toc-link][aria-current="true"]')?.dataset.nbSlug;
        return mobile === slug && desktop === slug;
      }, slug);
    };
    const middleOfCommands = async () => {
      await page.evaluate(() => {
        const y = (id) => scrollY + document.getElementById(id).getBoundingClientRect().top;
        window.scrollTo(0, (y('commands') + y('change-the-policy')) / 2);
      });
      await expectSection('commands');
    };
    await page.evaluate(() => {
      const heading = document.getElementById('manage-skills');
      window.scrollTo(0, scrollY + heading.getBoundingClientRect().top - innerHeight * 0.2);
    });
    await expectSection('manage-skills');
    await middleOfCommands();
    await page.evaluate(() => window.scrollTo(0, 0));
    await page.waitForFunction(() => document.querySelector('[data-nb-mobile-toc-select]').value === '_top');
    await middleOfCommands();
    await page.reload();
    await expectSection('commands');
    await page.setViewportSize({ width: viewport.width, height: 844 });
    await page.evaluate(() => {
      const heading = document.getElementById('manage-skills');
      window.scrollTo(0, scrollY + heading.getBoundingClientRect().top - 200);
    });
    await expectSection('manage-skills');
    await page.setViewportSize({ width: viewport.width, height: 390 });
    await expectSection('set-up-empryo-once-per-machine');
    await page.setViewportSize(viewport);
    if (viewport.width < 1280) {
      await page.locator('[data-nb-mobile-toc-select]').selectOption('manage-skills');
      await page.waitForFunction(() => Math.abs(document.getElementById('manage-skills').getBoundingClientRect().top) < 150);
      await page.waitForTimeout(1100);
      await expectSection('manage-skills');
      await middleOfCommands();
      await page.locator('[data-nb-mobile-toc-select]').selectOption('_top');
      await page.waitForFunction(() => scrollY === 0);
    } else {
      await page.locator('[data-nb-toc-link][data-nb-slug="manage-skills"]').click();
      await expectSection('manage-skills');
      await page.mouse.wheel(0, -100);
      await middleOfCommands();
    }
    console.log(`PASS TOC scrolling, restoration, resize and navigation ${viewport.width}x${viewport.height}`);
    await page.close();
  }
  const response = await browser.newPage();
  assert.equal((await response.goto(base + '/nimbus-api/coordinates.json')).status(), 404);
  console.log('PASS unconfigured API manifest is absent');
} finally {
  await browser?.close();
  await close();
}
