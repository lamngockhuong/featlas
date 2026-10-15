# muninn

A product handbook that AI writes, people review, and one binary serves.

In teams where AI writes much of the code, each person understands only the part they built. muninn keeps a
business-level map of the product next to the code: features, screens with real screenshots and numbered
hotspots, step-by-step flows, the modules and files behind them, and a glossary. An AI skill writes the
content from the source code; a person approves it on the web; muninn tells you when approved content no
longer matches the code.

The name is Muninn ("memory"), one of Odin's two ravens.

## Quick start

```sh
curl -fsSL https://raw.githubusercontent.com/lamngockhuong/muninn/main/install.sh | sh
muninn serve --dir muninn --open
```

On Windows, download the `.zip` for your architecture from the
[releases](https://github.com/lamngockhuong/muninn/releases) page and put `muninn.exe` on your PATH.

Try it on the bundled demo, a fictional expense-claim app:

```sh
git clone https://github.com/lamngockhuong/muninn && cd muninn
muninn serve --dir examples/demo/muninn --open
```

## How it works

```
 AI skill (muninn-gen)          your repository                      muninn binary
 reads routes, guards,   ──►    muninn/muninn.yaml            ◄──    validate · status · review
 schemas; captures screens      muninn/features|screens|flows|modules  serve: web UI + API + live reload
 writes status: draft           src/... (hashed to detect drift)       capture (chromedp) · export
```

- The binary is generic: it reads files, validates them, serves the web UI, records reviews and, optionally,
  captures screenshots. It never calls an LLM, commits or pushes.
- The skill knows stacks. It reads `muninn schema`, writes content, and checks itself with `muninn validate`.
- Content only ever enters as `draft`. Approval happens on the web or with `muninn review`. When an approved
  page's text or images change it becomes **modified**; when its source files change it becomes
  **outdated**.

## Generating content

Install the Claude Code plugin from this repository, then run the skill in your project:

```
/plugin marketplace add lamngockhuong/muninn
/plugin install muninn@muninn
/muninn-gen init
/muninn-gen features
/muninn-gen screens --role <role>
```

See [plugins/muninn/skills/muninn-gen/SKILL.md](plugins/muninn/skills/muninn-gen/SKILL.md) for every subcommand and rule.

## Commands

| Command | Purpose |
|---|---|
| `muninn init [--dir muninn]` | Create `muninn.yaml` and empty content directories |
| `muninn schema [--entity screen]` | Print the JSON Schema of the format (for generators) |
| `muninn validate [dir] [--format text\|json]` | Check schema, references, images, hotspots and sources; exit 1 on errors |
| `muninn status [--changed-since <ref>] [--format text\|json\|markdown]` | List pages by computed review state |
| `muninn review <kind>/<id> --approve\|--request-changes --by <name> [--note]` | Review from the terminal |
| `muninn serve [--host] [--port 7077] [--read-only] [--token] [--open]` | Web UI and API with live reload |
| `muninn capture --script capture.yaml` | Screenshots with element boxes using Chrome |
| `muninn validate-capture <file>` | Check a capture result against its image |
| `muninn export --out site/` | Read-only static site for GitHub Pages or any static host |
| `muninn version` | Print the version |

`serve` binds to `127.0.0.1` and allows reviewing. Binding elsewhere without `--token` switches to
read-only. With a token, reviews need `Authorization: Bearer <token>`; the web UI asks for it once.

## Documentation

- [Content format](docs/content-format.md): directory layout, every entity, review states.
- [Capturing screenshots](docs/capture.md): the three backends and the shared result format.
- [CI and static export](docs/ci.md): the GitHub Action that comments on pull requests, and `export`.
- [Development](docs/development.md): building, testing and releasing muninn itself.

## License

[MIT](LICENSE)
