# Development

## Layout

```
cmd/muninn/            entry point
internal/content/      parse frontmatter, load the content directory, cross-file checks
internal/schema/       validate documents against the embedded JSON Schema
internal/review/       content and source hashes, computed review states, writing the review block
internal/server/       HTTP API, SSE live reload, file watcher
internal/capture/      capture scripts, chromedp backend, shared export of screenshots and boxes
internal/export/       static site export
internal/webui/        embeds the web app (static/ is written by `make web`)
internal/cli/          cobra commands
schema/                muninn.schema.json and capture.schema.json (the content contract)
web/                   React app (Vite, TanStack Router and Query, Tailwind)
skills/muninn-gen/     Claude Code skill and the Playwright and pinchtab capture scripts
examples/demo/         Acme Expense: a fake app (app/), its content (muninn/) and capture.yaml
```

## Requirements

Go (version in `go.mod`), Node 22+ with pnpm, and Chrome or Chromium for capture tests.

## Common tasks

| Task | Command |
|---|---|
| Build the binary with the web app | `make build` (writes `bin/muninn`) |
| Run Go and web unit tests | `make test` |
| Web end-to-end tests (Playwright, on a copy of the demo) | `make e2e` |
| Compare capture backends on the demo app | `make capture-e2e` |
| Lint (go vet, golangci-lint, Biome) | `make lint` |
| Develop with hot reload | `make dev`, then open http://localhost:5174 |
| Serve the demo with the embedded app | `make demo` |

`make dev` runs the API from `go run -tags dev` on port 7078 against `examples/demo/muninn` and Vite on
port 5174, which proxies `/api` and `/content` to it. Reviewing in dev mode writes to the demo files;
discard those changes before committing.

To recapture the demo screenshots, serve the app and run the capture script:

```sh
python3 -m http.server 5173 -d examples/demo/app &
ACME_PASSWORD=demo go run ./cmd/muninn capture --script examples/demo/capture.yaml
find examples/demo/muninn -name '*.capture.json' -delete
```

If boxes moved, update the hotspots in the affected `screen.md` and `flow.md` files.

## Rules for this repository

- Content, fixtures and demos use the fictional Acme Expense app only. Never commit names, screenshots or
  notes from a real project or customer. CI checks a private list of terms kept in the
  `MUNINN_FORBIDDEN_TERMS` secret.
- The schema is a public contract. Changes that would break existing content need a new format version.
- Keep `skills/muninn-gen/scripts/measure.js` identical to `internal/capture/measure.js` (a test checks it).

## Releasing

Push a `v*` tag. The release workflow builds the web app, then goreleaser builds linux, macOS and Windows
binaries for amd64 and arm64, with checksums, into a draft GitHub release. Publish the draft after
checking it, and move the `v1` tag used by the GitHub Action when the release is compatible.
