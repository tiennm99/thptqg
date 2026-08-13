# thptqg

Tra cứu điểm thi THPT Quốc gia — exam-score lookup for Vietnam's national high
school graduation exam. Client-side SQL (sql.js) over a SQLite database built
from the ministry's raw `.xls` score files by the Go `xlsxread` parser.

Live at **[tiennm99.github.io/thptqg](https://tiennm99.github.io/thptqg/)**.

| Dataset | Exam | Candidates | Site |
| --- | --- | --- | --- |
| `2016` | 2016 | 877,461 | [/2016/](https://tiennm99.github.io/thptqg/2016/) |
| `2017` | 2017 | 861,068 | [/2017/](https://tiennm99.github.io/thptqg/2017/) |

Two earlier 2017 publications (`2017-old`, `2017-old2`) were kept for a while
because they disagreed with the current one. They have been removed; they remain
in git history.

## Layout

```
web/                  the frontend — one Vite app serving both datasets and the hub
  src/datasets.js       the dataset ids and their per-dataset content
  src/router.js         pathname → dataset
  scripts/              site assembly
crawler/              Go — re-fetches the source spreadsheets
  internal/sources/     per dataset: which article to read, how to name its files
  internal/article/     pulls the download links out of that article
  internal/fetch/       concurrent, resumable downloading
go-parser/            Go — Excel to SQLite
  internal/schema/      canonical 22-column table: DDL, INSERT, subject regexes
  configs/<id>.yml      per-dataset parse rules only, no SQL
  scripts/              database build, parity verification
data/<id>/            raw Excel files, one directory per dataset
docs/                 architecture, data pipeline, deployment
```

`web/` is the only npm workspace; `crawler/` and `go-parser/` are independent Go
modules. The one cross-boundary import is `web/src/datasets.js`, which
`go-parser/scripts/build-db.js` reads for the dataset list and expected sizes.

The dataset id is one identifier end to end:

```
data/2017/ → go-parser/configs/2017.yml → db/2017.db.gz → /thptqg/2017/
```

## Build

```bash
npm ci
npm run build:go       # compile the parser
npm run build:db       # build + gzip both databases (add an id for just one)
npm run build:site     # one Vite build, then assemble into _site/
npx serve _site
```

The source spreadsheets are committed, so a crawl is only needed to refresh
them:

```bash
npm run crawl:2016     # re-fetch data/2016/
npm run crawl:2017     # re-fetch data/2017/
```

Each reads the download links out of the article that published the dataset, so
no link list is kept in the repository. Crawling is idempotent — files already
present are skipped — and is never part of the build.

Pushing to `main` runs the same steps in
`.github/workflows/deploy-pages.yml` and publishes to GitHub Pages.

## Adding a dataset

1. Put the Excel files in `data/<id>/`
2. Add `go-parser/configs/<id>.yml` — sheet mode, column indices, validation
   guards. No SQL; the schema is canonical.
3. Add an entry to `DATASETS` in `web/src/datasets.js`

Everything else follows: the build script, the site assembly and the router all
read that one list, and the UI adapts to whichever columns the dataset fills.

## Docs

See [`docs/`](./docs/) — [overview](./docs/project-overview.md),
[architecture](./docs/system-architecture.md),
[data pipeline](./docs/data-pipeline.md),
[deployment](./docs/deployment-guide.md).
