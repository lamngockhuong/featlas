# Capturing screenshots

Hotspots are only useful when the boxes match the pixels. Every capture backend therefore measures the
elements and takes the screenshot in the same DOM state, with no action, navigation or wait in between,
and writes the same result file. Three backends are supported:

| Backend | Use it when | How |
|---|---|---|
| chromedp | You want nothing but muninn and Chrome | `muninn capture --script capture.yaml` |
| Playwright | The project already uses Playwright, or sign-in is complex | `node plugins/muninn/skills/muninn-gen/scripts/capture-playwright.mjs capture.yaml` |
| pinchtab | A pinchtab server is already running | `node plugins/muninn/skills/muninn-gen/scripts/capture-pinchtab.mjs capture.yaml` |

The Node backends hand their PNG and measurements to `muninn capture-finalize`, so crop, resize, box
conversion and WebP encoding happen in one place for all three.

## Script

```yaml
baseUrl: http://localhost:5173
viewport: { w: 1366, h: 768 }
deviceScaleFactor: 1      # optional
maxWidth: 1440            # optional; wider shots are scaled down with their boxes
steps:
  - goto: /login.html
  - fill: { selector: "#email", value: "emma@acme.test" }
  - fill: { selector: "#password", value: "${env:ACME_PASSWORD}" }
  - click: "#login-form button[type=submit]"
  - waitFor: { url: /claims.html }
  - capture:
      out: muninn/screens/claim-list/shots/default
      elements:
        status-filter: "[data-testid=status-filter]"
        first-row: "[data-testid=claims-table] tbody tr:first-child"
      crop: null          # or a CSS selector, or [x, y, w, h] in CSS pixels
```

- Steps: `goto`, `fill`, `click`, `select`, `upload`, `waitFor` (`selector`, `url` substring or `ms`),
  `capture`. Each step has exactly one action.
- `out` is relative to the script file. A capture writes `<out>.webp` and `<out>.capture.json`.
- Secrets only come from the environment through `${env:NAME}`. A missing variable stops the run before
  the browser starts.
- `muninn capture --profile .muninn-profile` keeps cookies between runs; keep that directory out of Git.
- muninn finds Chrome or Chromium on PATH, in standard install locations, or a Chromium installed by
  Playwright. Set `--chrome` or `MUNINN_CHROME` otherwise. It never downloads a browser.

The complete demo script is [`examples/demo/capture.yaml`](../examples/demo/capture.yaml).

## Result: `capture.json`

```json
{
  "backend": "chromedp",
  "url": "http://localhost:5173/claims.html",
  "viewport": { "w": 1366, "h": 768 },
  "capturedAt": "2026-10-14T11:30:25Z",
  "shot": "default.webp",
  "elements": [
    { "key": "status-filter", "selector": "[data-testid=status-filter]", "text": "All", "box": [81, 126, 115, 34] }
  ]
}
```

Boxes are in pixels of the exported image (device pixel ratio applied, crop subtracted, resize applied).
The schema is `muninn schema --entity capture`; `muninn validate-capture <file>` checks a result against
its image. Copy boxes from here into screen hotspots and flow steps.

## pinchtab notes

The pinchtab backend uses pinchtab's HTTP API (`/navigate`, `/action`, `/emulation/viewport`,
`/evaluate`, `/screenshot`). It needs `PINCHTAB_TOKEN` and JavaScript evaluation enabled on the server
(`security.allowEvaluate`, or `pinchtab server --yolo` for one run). `upload` steps are not supported.
pinchtab's own `capture` command pairs a screenshot with an accessibility snapshot, but its nodes cannot be
matched to CSS selectors exactly, so muninn does not use it.

## Comparing backends

`make capture-e2e` runs the demo script through chromedp and Playwright (and pinchtab when `PINCHTAB_URL`
is set) and fails when any box differs by more than 2px. Fonts differ between machines, so compare results
from the same machine only.
