#!/usr/bin/env node
// pinchtab capture backend. Drives a running pinchtab server over its HTTP API
// with the same capture.yaml as `muninn capture`:
//   PINCHTAB_URL=http://127.0.0.1:9867 node capture-pinchtab.mjs <capture.yaml>
// Each capture step evaluates the shared measuring function, then takes a PNG
// screenshot right after it, so boxes and pixels come from one DOM state.
// PINCHTAB_TOKEN is the server token (`pinchtab config get server.token`).
// The server must allow evaluation: security.allowEvaluate or --yolo.
import { finalize, measureExpression, resolveUrl, runScript } from './capture-lib.mjs';

const [file] = process.argv.slice(2);
if (!file) {
  console.error('usage: capture-pinchtab.mjs <capture.yaml>');
  process.exit(2);
}
const base = (process.env.PINCHTAB_URL || 'http://127.0.0.1:9867').replace(/\/+$/, '');
const headers = { 'Content-Type': 'application/json' };
if (process.env.PINCHTAB_TOKEN) headers.Authorization = `Bearer ${process.env.PINCHTAB_TOKEN}`;

async function call(method, path, body) {
  const res = await fetch(base + path, { method, headers, body: body && JSON.stringify(body) });
  if (!res.ok) throw new Error(`pinchtab ${method} ${path}: ${res.status} ${await res.text()}`);
  return res;
}

// /evaluate needs evaluation enabled on the server (security.allowEvaluate,
// or `pinchtab server --yolo` for one run). It answers { result: <value> }.
async function evaluate(expression) {
  const res = await call('POST', '/evaluate', { expression, awaitPromise: true });
  return (await res.json()).result;
}

const js = (s) => JSON.stringify(s);

async function poll(check, what, ms = 30_000) {
  const end = Date.now() + ms;
  while (Date.now() < end) {
    if (await check()) return;
    await new Promise((r) => setTimeout(r, 100));
  }
  throw new Error(`timed out waiting for ${what}`);
}

async function makeBackend(script, dir) {
  const visible = (sel) =>
    evaluate(`(() => { const el = document.querySelector(${js(sel)}); if (!el) return false; const r = el.getBoundingClientRect(); return r.width > 0 && r.height > 0; })()`);
  const viewport = () =>
    call('POST', '/emulation/viewport', {
      width: script.viewport.w,
      height: script.viewport.h,
      deviceScaleFactor: script.deviceScaleFactor,
      mobile: false,
    });
  return {
    // Navigation can open a new tab, which starts without the emulated
    // viewport, so the viewport is applied after every goto.
    goto: async (target) => {
      await call('POST', '/navigate', { url: resolveUrl(script, target) });
      await viewport();
      await evaluate('1'); // fails fast when evaluation is disabled
    },
    fill: async ({ selector, value }) => {
      await poll(() => visible(selector), selector);
      await call('POST', '/action', { kind: 'fill', selector, text: value });
    },
    click: async (selector) => {
      await poll(() => visible(selector), selector);
      // pinchtab's action guard answers 409 navigation_changed after a click
      // that navigates. The click happened; the next waitFor step syncs.
      const res = await fetch(`${base}/action`, { method: 'POST', headers, body: JSON.stringify({ kind: 'click', selector }) });
      if (res.ok) return;
      const text = await res.text();
      if (res.status !== 409 || !text.includes('navigation_changed')) throw new Error(`pinchtab POST /action: ${res.status} ${text}`);
    },
    select: async ({ selector, value }) => {
      await poll(() => visible(selector), selector);
      await evaluate(`(() => { const el = document.querySelector(${js(selector)}); el.value = ${js(value)}; el.dispatchEvent(new Event('change', { bubbles: true })); })()`);
    },
    upload: () => {
      throw new Error('upload is not supported by the pinchtab backend; use chromedp or Playwright');
    },
    waitFor: ({ selector, url, ms }) => {
      if (selector) return poll(() => visible(selector), selector);
      if (url) return poll(async () => String(await evaluate('location.href')).includes(url), `URL containing ${url}`);
      return new Promise((r) => setTimeout(r, ms));
    },
    capture: async (capture) => {
      const measured = await evaluate(measureExpression(capture));
      const res = await call('GET', '/screenshot?raw=true&format=png');
      const png = Buffer.from(await res.arrayBuffer());
      return finalize({ backend: 'pinchtab', script, dir, capture, measured, png });
    },
    close: async () => {},
  };
}

runScript(file, makeBackend).catch((err) => {
  console.error(`error: ${err.message}`);
  process.exit(1);
});
