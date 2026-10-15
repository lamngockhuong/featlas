# Capture with pinchtab

`scripts/capture-pinchtab.mjs` drives a running pinchtab server over its HTTP API (tested with
pinchtab 0.15). For each `capture` step it sets the viewport, evaluates `scripts/measure.js` and takes a PNG
screenshot right after it, then calls `muninn capture-finalize`.

Requirements:

- A pinchtab server on `PINCHTAB_URL` (default `http://127.0.0.1:9867`) and its token in `PINCHTAB_TOKEN`
  (`pinchtab config get server.token`).
- JavaScript evaluation enabled: `security.allowEvaluate: true` in the pinchtab config, or start the server
  with `pinchtab server --yolo` for one run. The script fails fast with a clear error otherwise.
- The app's host in `security.allowedDomains` (localhost is allowed by default).
- A browser pinchtab can find (`pinchtab doctor`; set `browser.binary` if needed).

Limitations: `upload` steps are not supported (use chromedp or Playwright for flows that upload files).
A click that navigates returns pinchtab's `navigation_changed` guard response; the script treats it as
success, so follow such clicks with a `waitFor` step.

pinchtab's own `capture` command pairs a screenshot with an accessibility snapshot. Its nodes carry boxes
but no CSS selectors, so muninn does not use it: matching selectors to accessibility nodes is not exact.
