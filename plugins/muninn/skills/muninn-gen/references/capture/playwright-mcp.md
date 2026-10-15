# Capture with Playwright MCP

Use this when the session has Playwright MCP tools but you cannot run Node scripts. The result must be the
same `capture.json` as the other backends, so the measuring and the screenshot must happen in the same DOM
state, with nothing in between.

1. Set the viewport from the capture script (`browser_resize` to `viewport.w` × `viewport.h`).
2. Run the script steps with the MCP tools: `browser_navigate` (goto), `browser_type` or `browser_fill_form`
   (fill), `browser_click`, `browser_select_option`, `browser_file_upload`, `browser_wait_for`.
3. For each `capture` step:
   1. Call `browser_evaluate` with the function in `scripts/measure.js`, passing
      `{ elements: [[key, selector], ...sorted by key], cropSelector: "<selector or empty>" }`.
      It returns `{ dpr, url, elements, crop }` in CSS pixels.
   2. Immediately call `browser_take_screenshot` with PNG output for the viewport (not full page).
      Do not click, scroll, hover or wait between the two calls.
   3. Write the raw data to a temporary JSON file:

      ```json
      { "backend": "playwright", "url": "<url>", "viewport": { "w": 1366, "h": 768 },
        "dpr": 1, "maxWidth": 1440, "out": "<abs path without extension>",
        "screenshot": "<abs path of the PNG>", "elements": [ ...from measure... ], "crop": null }
      ```

   4. Run `muninn capture-finalize --input <that file>`. It writes `<out>.webp` and `<out>.capture.json`
      with boxes converted to image pixels.
4. Check with `muninn validate-capture <out>.capture.json`.
