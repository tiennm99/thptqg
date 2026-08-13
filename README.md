# thptqg

Tra cứu điểm thi THPT Quốc gia — exam-score lookup for Vietnam's national high
school graduation exam. Client-side SQL (sql.js) over a SQLite database built
from the ministry's raw `.xls` score files by the Rust `xlsxread` parser.

Live at **[tiennm99.github.io/thptqg](https://tiennm99.github.io/thptqg/)**.

| Dataset | Exam | Candidates | Site |
| --- | --- | --- | --- |
| `2016` | 2016 | 877,461 | [/2016/](https://tiennm99.github.io/thptqg/2016/) |
| `2017` | 2017 | 861,068 | [/2017/](https://tiennm99.github.io/thptqg/2017/) |
| `2017-old` | 2017 | 847,348 | [/2017-old/](https://tiennm99.github.io/thptqg/2017-old/) |
| `2017-old2` | 2017 | 679,764 | [/2017-old2/](https://tiennm99.github.io/thptqg/2017-old2/) |

The three 2017 datasets are successive publications of the same exam and they
disagree; all three are kept so the differences stay inspectable.

## Layout

```
index.html + src/     the frontend — one app serving all four datasets and the hub
  datasets.js           the four dataset ids and their per-dataset content
  router.js             pathname → dataset
data/<id>/            raw Excel files, one directory per dataset
parser/               the Rust parser
  src/schema.rs         canonical 22-column table: DDL, INSERT, subject regexes
  configs/<id>.yml     per-dataset parse rules only, no SQL
  scripts/              database build, crawler, parity verification
scripts/              site assembly
docs/                 architecture, data pipeline, deployment
```

The dataset id is one identifier end to end:

```
data/2017-old/ → go-parser/configs/2017-old.yml → db/2017-old.db.gz → /thptqg/2017-old/
```

## Build

```bash
npm ci
npm run build:go     # compile the parser
npm run build:db       # build + gzip all four databases (add an id for just one)
npm run build:site     # one Vite build, then assemble into _site/
npx serve _site
```

Pushing to `main` runs the same steps in
`.github/workflows/deploy-pages.yml` and publishes to GitHub Pages.

## Adding a dataset

1. Put the Excel files in `data/<id>/`
2. Add `go-parser/configs/<id>.yml` — sheet mode, column indices, validation
   guards. No SQL; the schema is canonical.
3. Add an entry to `DATASETS` in `src/datasets.js`

Everything else follows: the build script, the site assembly and the router all
read that one list, and the UI adapts to whichever columns the dataset fills.

## Docs

See [`docs/`](./docs/) — [overview](./docs/project-overview.md),
[architecture](./docs/system-architecture.md),
[data pipeline](./docs/data-pipeline.md),
[deployment](./docs/deployment-guide.md).
