// Shared parts of the Node capture backends: reading capture.yaml exactly as
// `muninn capture` does, and handing measurements to `muninn capture-finalize`
// so every backend crops, resizes and encodes the same way.
import { execFileSync } from 'node:child_process';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { parse } from 'yaml';

const here = path.dirname(fileURLToPath(import.meta.url));

/** Source of the measuring function, identical to the one embedded in muninn. */
export const measureSource = fs.readFileSync(path.join(here, 'measure.js'), 'utf8');

const ENV_REF = /\$\{env:([A-Za-z_][A-Za-z0-9_]*)\}/g;

function expandEnv(value, missing) {
  if (typeof value === 'string') {
    return value.replace(ENV_REF, (_, name) => {
      if (!(name in process.env)) missing.add(name);
      return process.env[name] ?? '';
    });
  }
  if (Array.isArray(value)) return value.map((v) => expandEnv(v, missing));
  if (value && typeof value === 'object') {
    return Object.fromEntries(Object.entries(value).map(([k, v]) => [k, expandEnv(v, missing)]));
  }
  return value;
}

const ACTIONS = ['goto', 'fill', 'click', 'select', 'upload', 'waitFor', 'capture'];

/** Reads and checks a capture script. Missing ${env:X} fails before any browser starts. */
export function loadScript(file) {
  const missing = new Set();
  const script = expandEnv(parse(fs.readFileSync(file, 'utf8')), missing);
  if (missing.size) throw new Error(`environment variables not set: ${[...missing].sort().join(', ')}`);
  if (!script?.viewport?.w || !script?.viewport?.h) throw new Error('viewport { w, h } is required');
  if (!Array.isArray(script.steps) || script.steps.length === 0) throw new Error('steps is empty');
  script.deviceScaleFactor ??= 1;
  script.maxWidth ??= 1440;
  script.steps.forEach((step, i) => {
    const keys = Object.keys(step).filter((k) => ACTIONS.includes(k));
    if (keys.length !== 1 || Object.keys(step).length !== 1) {
      throw new Error(`step ${i + 1}: needs exactly one of ${ACTIONS.join(', ')}`);
    }
    if (step.capture && !step.capture.out) throw new Error(`step ${i + 1}: capture.out is required`);
  });
  return { script, dir: path.dirname(path.resolve(file)) };
}

export function resolveUrl(script, target) {
  if (target.includes('://')) return target;
  return `${script.baseUrl.replace(/\/+$/, '')}/${target.replace(/^\/+/, '')}`;
}

/** The argument of the measuring function for one capture step. */
export function measureSpec(capture) {
  const elements = Object.entries(capture.elements ?? {}).sort(([a], [b]) => (a < b ? -1 : a > b ? 1 : 0));
  return { elements, cropSelector: typeof capture.crop === 'string' ? capture.crop : '' };
}

/** A JS expression that measures and resolves to { dpr, url, elements, crop }. */
export function measureExpression(capture) {
  return `(${measureSource})(${JSON.stringify(measureSpec(capture))})`;
}

/**
 * Writes <out>.webp and <out>.capture.json through `muninn capture-finalize`.
 * `measured` is the result of the measuring function, `png` a PNG Buffer.
 */
export function finalize({ backend, script, dir, capture, measured, png, muninn = process.env.MUNINN_BIN || 'muninn' }) {
  if (measured.crop?.missing) throw new Error(`crop selector ${JSON.stringify(capture.crop)} matches nothing`);
  const tmp = fs.mkdtempSync(path.join(os.tmpdir(), 'muninn-capture-'));
  try {
    const shot = path.join(tmp, 'shot.png');
    fs.writeFileSync(shot, png);
    const crop = Array.isArray(capture.crop)
      ? { X: capture.crop[0], Y: capture.crop[1], W: capture.crop[2], H: capture.crop[3] }
      : measured.crop
        ? { X: measured.crop.x, Y: measured.crop.y, W: measured.crop.w, H: measured.crop.h }
        : null;
    const raw = {
      backend,
      url: measured.url,
      viewport: { w: script.viewport.w, h: script.viewport.h },
      dpr: measured.dpr,
      maxWidth: script.maxWidth,
      out: path.join(dir, capture.out),
      screenshot: shot,
      elements: measured.elements,
      crop,
    };
    const input = path.join(tmp, 'raw.json');
    fs.writeFileSync(input, JSON.stringify(raw));
    return execFileSync(muninn, ['capture-finalize', '--input', input], { encoding: 'utf8' }).trim();
  } finally {
    fs.rmSync(tmp, { recursive: true, force: true });
  }
}

/** Runs the script steps through a backend adapter and logs written files. */
export async function runScript(file, makeBackend) {
  const { script, dir } = loadScript(file);
  const backend = await makeBackend(script, dir);
  try {
    for (const [i, step] of script.steps.entries()) {
      const [action, arg] = Object.entries(step)[0];
      try {
        const written = await backend[action](arg);
        if (typeof written === 'string') console.log(written);
      } catch (err) {
        throw new Error(`step ${i + 1} (${action}): ${err.message}`);
      }
    }
  } finally {
    await backend.close();
  }
}
