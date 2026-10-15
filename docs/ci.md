# CI and static export

## GitHub Action

The repository root is a composite action. It installs muninn, runs `muninn validate` (failing the job on
errors), and on pull requests lists the business pages whose `sources` the change edits:

```yaml
name: muninn
on: [pull_request]
permissions:
  contents: read
  pull-requests: write      # to comment
jobs:
  muninn:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: lamngockhuong/muninn@v1
        with:
          dir: muninn          # content directory
          comment: true        # post or update one PR comment
          # version: v0.1.0    # default: latest release
```

The comment (and the job summary) shows each affected feature, screen, flow or module with its review
state and the changed files. It only warns: the person changing code is often not the person who reviews
the business text. The comment is updated in place on later pushes, found by its
`<!-- muninn-status -->` marker. Output `affected` holds the number of affected pages.

The same report is available locally:

```sh
muninn status --dir muninn --changed-since origin/main --format markdown
```

## Static export

```sh
muninn export --dir muninn --out site/
```

writes the web app, every API response as JSON and the referenced images. The site is read-only, uses hash
URLs, and works from any sub-path, so GitHub Pages or an internal S3 bucket can host it for readers who do
not run the binary. Serve it with any static server, for example `python3 -m http.server -d site`.
`--out` must be empty or a previous export. Export owns that directory and replaces it as a
whole, so keep hand-written files such as `CNAME` elsewhere and copy them in your publish step.
