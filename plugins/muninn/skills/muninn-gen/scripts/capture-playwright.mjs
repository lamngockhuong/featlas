#!/usr/bin/env node
// Playwright capture backend. Runs the same capture.yaml as `muninn capture`:
//   node capture-playwright.mjs <capture.yaml> [--headful]
// Needs `muninn` on PATH (or MUNINN_BIN) for the shared export step.
import { chromium } from 'playwright';
import path from 'node:path';
import { finalize, measureExpression, resolveUrl, runScript } from './capture-lib.mjs';

const [file, ...flags] = process.argv.slice(2);
if (!file) {
  console.error('usage: capture-playwright.mjs <capture.yaml> [--headful]');
  process.exit(2);
}
const timeout = 30_000;

async function makeBackend(script, dir) {
  const browser = await chromium.launch({ headless: !flags.includes('--headful') });
  const context = await browser.newContext({
    viewport: { width: script.viewport.w, height: script.viewport.h },
    deviceScaleFactor: script.deviceScaleFactor,
  });
  const page = await context.newPage();
  page.setDefaultTimeout(timeout);
  const css = (selector) => page.locator(selector).first();
  return {
    goto: (target) => page.goto(resolveUrl(script, target)),
    fill: ({ selector, value }) => css(selector).fill(value),
    click: (selector) => css(selector).click(),
    select: ({ selector, value }) => css(selector).selectOption(value),
    upload: ({ selector, file: f }) => css(selector).setInputFiles(path.join(dir, f)),
    waitFor: ({ selector, url, ms }) => {
      if (selector) return css(selector).waitFor({ state: 'visible' });
      if (url) return page.waitForURL((u) => u.href.includes(url));
      return page.waitForTimeout(ms);
    },
    capture: async (capture) => {
      // Measure, then screenshot, with no other action in between.
      const measured = await page.evaluate(measureExpression(capture));
      const png = await page.screenshot({ type: 'png' });
      return finalize({ backend: 'playwright', script, dir, capture, measured, png });
    },
    close: () => browser.close(),
  };
}

runScript(file, makeBackend).catch((err) => {
  console.error(`error: ${err.message}`);
  process.exit(1);
});
