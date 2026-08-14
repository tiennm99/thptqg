# thptqg

Tra cứu điểm thi THPT Quốc gia — exam-score lookup for Vietnam's national high
school graduation exam. Client-side SQL over a SQLite database read in place by
HTTP range request, built
from the published `.xls`/`.xlsx` score files by the Go `parser` module. Where
those files come from: [data pipeline](./docs/data-pipeline.md#sources).

Live at **[tiennm99.github.io/thptqg](https://tiennm99.github.io/thptqg/)**.

| Dataset | Exam | Candidates | Site |
| --- | --- | --- | --- |
| `2016` | 2016 | 877,460 | [/2016/](https://tiennm99.github.io/thptqg/2016/) |
| `2017` | 2017 | 861,068 | [/2017/](https://tiennm99.github.io/thptqg/2017/) |

Two earlier 2017 publications (`2017-old`, `2017-old2`) were kept for a while
because they disagreed with the current one. They have been removed; they remain
in git history.

## Layout

The repository is one directory per pipeline stage, plus the two stores they
pass between them.

```
crawler/      Go   — re-fetches the source spreadsheets      → data/
parser/       Go   — Excel to SQLite                          data/ → .db
assembler/    Go   — verifies, compresses, builds, assembles  .db + web/ → _site/
web/          npm  — the frontend, one SvelteKit app for every dataset
data/<id>/         raw Excel files, one directory per dataset
datasets.json      the registry: which datasets exist, and their expected size
docs/              architecture, data pipeline, deployment
```

Each stage runs on its own and hands its output to the next through the stores.
`web/` is the only npm project; the three stages are independent Go modules.

`datasets.json` is the contract between them. It is JSON because Go and the web
app both read it and neither needs a dependency to do so; presentation stays in
`web/src/lib/datasets.js`, keyed by id, which fails loudly if the two disagree.

The dataset id is one identifier end to end:

```
data/2017/ → parser/configs/2017.yml → db/2017.sqlite3 → /thptqg/2017/
```

## Build

```bash
(cd web && npm ci)
go -C assembler run ./cmd/assemble        # databases, then the site, into _site/
npx serve _site
```

That one command compiles the parser, builds and verifies each database against
its registry row count, compresses it, builds the web app and assembles `_site` —
refusing to continue if a database is short, an artifact looks truncated, or one
is missing altogether. Sub-steps when iterating:

```bash
go -C assembler run ./cmd/assemble db 2017   # one database
go -C assembler run ./cmd/assemble site      # web build and _site only
go -C assembler run ./cmd/assemble verify A B  # compare two sets of databases
(cd web && npm run dev)                      # the app against staged databases
```

The source spreadsheets are committed, so a crawl is only needed to refresh
them:

```bash
go -C crawler run ./cmd/crawl 2016
go -C crawler run ./cmd/crawl 2017
```

Each reads the download links out of the article that published the dataset, so
no link list is kept in the repository. Crawling is idempotent — files already
present are skipped — and is never part of the build.

Pushing to `main` runs the same steps in
`.github/workflows/deploy-pages.yml` and publishes to GitHub Pages.

## Adding a dataset

1. Put the Excel files in `data/<id>/`
2. Add `parser/configs/<id>.yml` — sheet mode, column indices, validation
   guards. No SQL; the schema is canonical.
3. Add an entry to `datasets.json` with its expected row count and size
4. Add the matching presentation to `CONTENT` in `web/src/lib/datasets.js`

Everything else follows: the assembler, the router and the hub all read the
registry, and the UI adapts to whichever columns the dataset fills. Steps 3 and 4
check each other, so forgetting either one fails rather than half-working.

## Docs

See [`docs/`](./docs/) — [overview](./docs/project-overview.md),
[architecture](./docs/system-architecture.md),
[data pipeline](./docs/data-pipeline.md),
[deployment](./docs/deployment-guide.md).
