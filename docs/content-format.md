# Content format (version 1)

muninn reads a directory of plain files, `muninn/` by default, that lives in the project repository. AI
generators write the files, people review them on the web, and Git keeps the history. The machine-readable
contract is [`schema/muninn.schema.json`](../schema/muninn.schema.json); print it with `muninn schema`
(or `muninn schema --entity screen`). This page explains the design behind it.

## Principles

- **One entity, one directory** (modules and the glossary excepted). Screenshots sit next to the file that
  describes them, so deleting or renaming a screen leaves no orphaned images.
- **Structured data in YAML frontmatter, prose in the Markdown body.** Diffs stay readable in pull
  requests, and generators get flat, explicit fields instead of deeply nested JSON.
- **Short multilingual text inline, long prose in one file per language.** `title: { en: Claims, ja: 経費申請 }`.
  `screen.md` holds the default language body; `screen.ja.md` holds only a body, no frontmatter.
  Translations never repeat routes, on-screen labels or sample values, because duplicates drift apart.
- **No declared image sizes.** muninn reads the size from the image file. Hotspot boxes are
  `[x, y, w, h]` in pixels of that image.
- **References are typed ids** (`feature: expense-claims`, `screens: [claim-list]`), never prefixed strings.
- **Ids are kebab-case**, unique per entity kind, and equal to the directory or file name.
- **The format is versioned** with `muninn: 1` in `muninn.yaml`. Breaking changes raise the version.

## Layout

```
muninn/
  muninn.yaml                    project configuration
  glossary.yaml                  business terms
  features/<id>/feature.md       (+ feature.<lang>.md)
  screens/<id>/screen.md         (+ screen.<lang>.md, shots/<variant>.webp)
  flows/<id>/flow.md             (+ flow.<lang>.md, shots/step-<n>.webp)
  modules/<id>.md                (+ <id>.<lang>.md)
```

[`examples/demo/muninn`](../examples/demo/muninn) is a complete example for a fictional expense app.

## `muninn.yaml`

| Field | Meaning |
|---|---|
| `muninn` | Format version, `1`. |
| `project.name`, `project.summary` | Shown on the home page. |
| `languages.default`, `languages.available` | Language of the main bodies; languages readers can switch to. |
| `repo.root` | Repository root relative to `muninn.yaml` (default `..`). Every `sources` path resolves against it. |
| `repo.url`, `repo.branch` | Used for code permalinks. |
| `repo.path` | Where `repo.root` sits inside `repo.url` when they differ (a monorepo package, an example folder). |
| `roles` | `{ id, label }` per user role. |
| `areas` | `{ id, label }` per feature group, usually the app's main menu. |
| `owners` | Default reviewers per area id, shown in the review queue. |

## Entities

**Feature** (`features/<id>/feature.md`): a business capability. Frontmatter lists `area`, `roles`,
`summary`, the `screens`, `flows`, `modules` and glossary `terms` it involves, external `specs` links and
`sources`. The body explains the problem it solves, the business rules, and status transitions (mermaid
code blocks are rendered).

**Screen** (`screens/<id>/screen.md`): one screen with `route`, `roles`, and at least one `variant`
(a screen state with its own screenshot). Each variant has numbered `hotspots`: `label`, `desc`, `box`,
`kind` (`control`, `hidden` for interactions with no visible affordance, `link`) and `to` (target screen).
`fields` maps on-screen labels to meaning and data source; `api`, `tables`, `modules` and `links` describe
what is behind the screen; `captured` records how the screenshots were taken.

**Flow** (`flows/<id>/flow.md`): a task a `role` performs, as numbered `steps`. A step's `action` is one of
`click`, `type`, `select`, `check`, `upload`, `run`, `wait`, `result`, and it has exactly one of:

- `screen` (and optionally `variant`): reuse that screen's screenshot;
- `shot`: its own screenshot in `shots/`;
- `run`: a terminal command (`cmd`, `where`), with no `box`.

`type` and `select` steps need `sample`, the real values entered when the screenshot was taken.

**Module** (`modules/<id>.md`): a backend service, frontend area, job or package with `kind`, `paths`,
`endpoints` (with `roles`), `tables` and `dependsOn`.

**Glossary** (`glossary.yaml`): a list of `{ id, term, meaning, related }`.

## Sources

`sources` lists the repository files (or directories, written with a trailing `/`) an entity was written
from. They serve two purposes: readers can jump to the code, and muninn can tell when the code changed after
a person approved the text. A directory is hashed over the files git sees in it (tracked and untracked, minus
everything git ignores), so a local checkout and CI agree. Without git, the root `.gitignore` applies. Paths must stay inside the repository (no `..`).

## Review

Generators write only `review: { status: draft }`. muninn writes everything else when a person reviews on
the web or with `muninn review`:

```yaml
review:
  status: approved                 # draft | approved | changes_requested
  by: Alice                        # name the reviewer typed in
  at: "2026-10-14T09:12:00Z"
  note: ""                         # required for changes_requested
  contentHash: sha256:9f2c...      # entity files without this block, translations, images
  sourceHashes:                    # git blob SHA-1 per source file; directories hash their file list
    web/src/routes/claims/index.tsx: 3b18e512...
```

Only this block changes when a review is recorded; every other byte of the file is kept, comments
included.

The state readers see is **computed**, not just read from the file:

| State | Meaning |
|---|---|
| `draft` | Written by a generator, not reviewed yet. |
| `changes_requested` | A reviewer asked for changes; see `note`. |
| `approved` | A person confirmed it, and nothing changed since. |
| `modified` | Approved, but the entity's own files (text, translations, images) changed since. |
| `outdated` | Approved, but a source file changed, disappeared or was added since. |

`modified` takes precedence over `outdated`; both list their reasons. Re-reviewing clears them.

## Validation

`muninn validate` checks every file against the schema, then across files: unknown ids in any
reference, duplicate ids, ids that differ from their directory, missing images, boxes outside the real image
size, flow step rules, translations without a main file or with frontmatter, and sources that do not exist
(for unreviewed entities; reviewed ones become `outdated` instead). It warns about images over 400 KB and
drafts unchanged for more than 30 days (by file modification time). It exits with status 1 when there are
errors.
