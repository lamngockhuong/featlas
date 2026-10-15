---
name: muninn-gen
description: Generate muninn product-atlas content (features, screens with real screenshots and hotspots, flows, modules, glossary) by reading a project's source code, then validate it with the muninn binary. Use when the user asks to document a product for onboarding, create or refresh muninn content, capture screens, or fix entities that muninn reports as outdated or changes requested. Subcommands - init, features, screens, flows, modules, refresh.
---

# muninn-gen

muninn is a single binary that serves, validates and reviews a product atlas stored as files in the
project (`muninn/` by default). You write the content; people review it on the web. The binary knows no
stack and calls no LLM. You are the part that reads the codebase.

## Non-negotiable rules

1. **Read the contract first.** Run `muninn schema` (or `muninn schema --entity screen`) before writing any
   file. Field names, required fields and enums come from there, not from memory.
2. **Only ever write `review: { status: draft }`.** Never write `approved`, `changes_requested`, `by`, `at`,
   `contentHash` or `sourceHashes`. Those are written by muninn when a person reviews. When you rewrite an
   entity that a person reviewed, reset its review block to `{ status: draft }`.
3. **Every business claim needs a source you actually read.** List the files in `sources` (repository
   relative; directories end with `/`). Not read means not written. When unsure, say so in the body
   ("Not confirmed in code: ...") instead of guessing.
4. **Validate after writing.** Run `muninn validate --dir <dir> --format json`; fix every error and run it
   again, at most three rounds. Report what is still failing instead of hiding it.
5. **Link, do not copy.** Existing specs, screen catalogues and business docs go in `specs` (URLs) and
   `sources` (files). Summarise what a reader needs; do not paste whole documents.
6. **Never write secrets into content.** Credentials come from environment variables or the local,
   git-ignored skill config (below). Sample values in flows must be demo data.
7. **Screens are read-only.** Flows that write data run only with `--allow-writes` **and** a local
   environment, after the project's reset command (see "Flows").

## Files you write

```
muninn/
  muninn.yaml                    # roles, areas, languages, repo  (init)
  glossary.yaml                  # business terms                 (features)
  features/<id>/feature.md       # + feature.<lang>.md bodies
  screens/<id>/screen.md         # + screen.<lang>.md, shots/<variant>.webp
  flows/<id>/flow.md             # + flow.<lang>.md, shots/step-<n>.webp
  modules/<id>.md                # + <id>.<lang>.md
```

- Ids are kebab-case and equal the directory (or module file) name.
- Short multilingual text is inline: `title: { en: Claims, ja: 経費申請 }`. Long prose in another language
  goes in `<base>.<lang>.md` with a body only (no frontmatter). Do not translate on-screen labels, routes
  or sample values.
- Never declare image sizes. Boxes are `[x, y, w, h]` in pixels of the image file.
- Quote inline values that contain commas, colons or `#`: `{ en: "Session, data store and header" }`.
  Unquoted, YAML splits them into extra keys and validation reports confusing errors.
- **Reference only entities that exist.** Validation fails on unknown ids. Write features first without
  `screens`, `flows` and `modules`; each later subcommand adds the ids it creates to the owning feature
  (and to other features that show the same screen). A role-scoped run never references screens of roles
  it did not capture.

## Skill config

`muninn/.gen.yaml` (commit it; no secrets):

```yaml
app:
  baseUrl: http://localhost:3000        # local app URL used for capture
  local: true                           # flows with --allow-writes refuse to run unless true
  reset: make db-reset-seed             # command that restores seed data before write flows
accounts:                               # one seed account per role; values are env variable names
  employee: { user: MUNINN_EMPLOYEE_USER, password: MUNINN_EMPLOYEE_PASSWORD }
  approver: { user: MUNINN_APPROVER_USER, password: MUNINN_APPROVER_PASSWORD }
capture:
  backend: auto                         # auto | playwright | pinchtab | chromedp
  viewport: { w: 1366, h: 768 }
stacks: [tanstack-router, nestjs, drizzle]   # files under references/stacks/ to read
```

Real values live in environment variables, or in `muninn/.gen.local.yaml` with the same shape, which must be
listed in `.gitignore`. Capture scripts reference them as `${env:NAME}`. Keep the capture browser profile
(`--profile .muninn-profile`) out of git too.

## Subcommands

### `/muninn-gen init`

1. If `muninn/muninn.yaml` is missing, run `muninn init --dir muninn`.
2. Detect the stack (package files, framework config). Read the matching files in `references/stacks/`.
   They are hints for where routes, guards, endpoints and tables live. When none matches (plain HTML,
   an in-house framework), find the same four things directly: the router or page files, the auth checks,
   the HTTP handlers and the data layer. Leave `stacks` empty.
3. Find roles (auth guards, role enums, seed users) and areas (the app's main menu or route groups).
4. Propose `muninn.yaml` (roles, areas, languages, `repo.url`, `repo.branch`, `owners` if known) and the skill
   config, show them to the user, and write them only after the user confirms. Roles and areas are product
   decisions; do not invent them silently.

### `/muninn-gen features`

1. From routes, menus and modules, propose a feature list: id, area, roles, one-line summary, and the
   screens and modules it will own. Confirm with the user.
2. Write `features/<id>/feature.md`: what problem it solves, the main business rules, the state machine as
   a mermaid `stateDiagram-v2` when entities have statuses. Collect terms into `glossary.yaml`. Leave
   `screens`, `flows` and `modules` out until those entities exist (see "Files you write").

### `/muninn-gen screens --role <id> [--only a,b] [--capture playwright|pinchtab|chromedp]`

For each screen the role can open (or only the listed ids):

1. Read the route component, its data loading (API calls, queries), guards and the components it renders.
2. Decide the variants worth showing: default, empty, error or permission states, a filled form. Each
   variant needs a reachable URL or a short sequence of steps. Capture a filled-form variant when a flow
   will need real `sample` values but cannot run with `--allow-writes`.
   Screens reached before sign-in (login, password reset) belong to every role: capture them once. A
   screen shared by several roles has one owning `feature`; show role-specific states as variants.
3. Write a capture script (see "Capture") and run it. Every capture step writes `<out>.webp` and
   `<out>.capture.json` with element boxes measured in the same DOM state as the pixels.
4. Write `screen.md`: copy boxes from `capture.json` into hotspots (never estimate boxes from the image),
   number hotspots in reading order (top to bottom, then left to right within a row), mark `kind: hidden` for interactions with no visible affordance (row
   click, hover menu), set `to` when the element opens another screen. Fill `fields` (on-screen label as
   shown, meaning, source column or API field), `api`, `tables`, `modules`, `links`, `sources`, and
   `captured: { at: 2026-10-14, branch: main, backend: chromedp, viewport: "1366x768" }` (viewport is a
   `WxH` string here, not the `{ w, h }` object of the capture script). `route` is the path as the router
   declares it; query-string pages may include the query (`/claim.html?id=<id>`).
   Add the screen id to the owning feature's `screens`.
5. Delete the `*.capture.json` files afterwards unless the project wants to keep them. They hold element
   labels and visible text, never form values, but visible text can still be personal data.

### `/muninn-gen flows --feature <id> [--allow-writes]`

1. Pick the tasks a role performs end to end (submit, approve, export...). One flow per goal.
2. Steps: `click | type | select | check | upload | run | wait | result`. Each step has exactly one of
   `screen` (+ optional `variant`, reusing that screenshot), `shot` (its own screenshot) or `run`
   (terminal command; no box). `type` and `select` steps need `sample` with the real values used.
3. Steps that write data: require `--allow-writes` **and** `app.local: true`. Run `app.reset` first, then
   drive the flow in the capture script, recording the exact values entered into `sample`. For apps that
   keep state in the browser (localStorage, IndexedDB), the reset is a fresh browser profile: run the
   capture without `--profile`, or delete the profile directory.
   Without both, write the flow from code only and reuse read-only screen shots; say so in the body.
4. Add the flow id to the feature's `flows`.

### `/muninn-gen modules`

Add each module id to the `modules` of the features and screens that use it. Describe backend services,
frontend areas, jobs and packages: `kind`, `paths`, `endpoints` (method, path,
roles from guards), `tables`, `dependsOn`, and a body explaining responsibility and non-obvious behaviour.

### `/muninn-gen refresh`

1. Run `muninn status --dir muninn --format json`.
2. For `outdated` entities: read the changed sources listed in `reasons`, update only what changed, recapture
   affected screens, reset `review` to `{ status: draft }`.
3. For `changes_requested`: read `review.note` and fix exactly what the reviewer asked; do not rewrite
   unrelated parts. Reset `review` to `{ status: draft }`.
4. For `modified` (edited after approval by someone): leave it for a person to review; report it.
5. Validate.

## Capture

All three backends read the same script format and write the same `capture.json`
(`muninn schema --entity capture`). Choose with `--capture` or `capture.backend`; with `auto` use pinchtab
if a pinchtab server answers, then Playwright if the project already uses it, otherwise `muninn capture`.

```yaml
baseUrl: http://localhost:3000
viewport: { w: 1366, h: 768 }
steps:
  - goto: /login
  - fill: { selector: "#email", value: "${env:MUNINN_EMPLOYEE_USER}" }
  - fill: { selector: "#password", value: "${env:MUNINN_EMPLOYEE_PASSWORD}" }
  - click: "button[type=submit]"
  - waitFor: { url: /claims }
  - capture:
      out: muninn/screens/claim-list/shots/default      # relative to the script file
      elements: { status-filter: "[data-testid=status-filter]", first-row: "table tbody tr:first-child" }
      crop: null                                        # or a selector, or [x, y, w, h] in CSS px
```

Steps: `goto`, `fill`, `click`, `select`, `upload`, `waitFor` (`selector` | `url` | `ms`), `capture`.
Missing `${env:...}` variables fail before the browser starts.

Keep capture scripts at the repository root as `muninn-capture-<role>.yaml` (so `out` paths start with
`muninn/`) and commit them: `refresh` reruns them to recapture outdated screens. They contain only
`${env:...}` references, never secret values.

- chromedp: `muninn capture --script capture.yaml [--profile .muninn-profile]` (needs Chrome or Chromium).
- Playwright: `node scripts/capture-playwright.mjs capture.yaml` (run `npm install` in `scripts/` once).
  It and pinchtab call `muninn capture-finalize`, so `muninn` must be on PATH (or set `MUNINN_BIN`).
  With Playwright MCP instead, follow `references/capture/playwright-mcp.md`.
- pinchtab: `node scripts/capture-pinchtab.mjs capture.yaml`, see `references/capture/pinchtab.md`.

Prefer stable selectors: `data-testid`, labels, roles. Wait for data to load before a capture step
(`waitFor` a row or a heading), never inside it.

## Finish

Run `muninn validate`, then `muninn status`. Report what you generated, which sources you read, anything
you could not confirm, and remind the user to review on the web (`muninn serve --dir muninn`).
