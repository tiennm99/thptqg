# Deployment Guide

Deploys to GitHub Pages via `.github/workflows/deploy-pages.yml`. Every push to
`main` rebuilds all four datasets and redeploys the whole site.

One-time setup: **Settings → Pages → Source: GitHub Actions**.

## What the workflow does

1. Checkout, Go toolchain, Node 24, `npm ci`
2. `npm run build:go` — one parser binary
3. `npm run build:db` — builds and gzips all four databases into
   `.build/public/db/`
4. `npm run build:site` — one Vite build, then `scripts/assemble-site.js`
5. `actions/upload-pages-artifact` + `actions/deploy-pages`

The database build dominates the runtime: roughly 419 MB of Excel is parsed on
every deploy.

## Resulting URLs

```
https://<user>.github.io/thptqg/
https://<user>.github.io/thptqg/2016/
https://<user>.github.io/thptqg/2017/
https://<user>.github.io/thptqg/2017-old/
https://<user>.github.io/thptqg/2017-old2/
```

`/thptqg/2017/old/` and `/thptqg/2017/old2/` were the pre-flattening URLs. They
are still served, and the router rewrites them to the flat form with the query
string intact.

## Local reproduction

```bash
npm ci
npm run build:go
npm run build:db      # all four; pass an id to build just one
npm run build:site    # vite build + assemble into _site/
npx serve _site
```

To rebuild a single dataset:

```bash
node go-parser/scripts/build-db.js 2017-old
```

## Base path

`vite.config.js` sets `base: "/thptqg/"`. If you fork under a different repo
name, update it to match — assets are referenced absolutely, so a mismatch shows
up as a blank page with 404s on `/assets/...`.

## Adding a dataset

1. Put the Excel files in `data/<id>/`
2. Add `parser/configs/<id>.yml` with the parse rules — sheet mode, column
   indices, SBD validation, header tokens, blank-row stripping. No SQL: the
   schema is canonical and lives in `go-parser/internal/schema/schema.go`
3. Add an entry to `DATASETS` in `src/datasets.js`

Nothing else. The build script, the site assembly and the router all read that
one list, and the frontend adapts to whichever columns the dataset populates.

## Why no uncompressed database can ship

`build-db.js` runs `gzip -9` **without** `-k`, so the raw file does not survive
the build. `assemble-site.js` then fails the job if any `.db`, `.db-journal`,
`.db-wal` or `.db-shm` reached the output.

Both guards exist because the previous pipeline wrote a 100+ MB uncompressed
database into the source tree and relied on an `rm` step to keep it out of the
artifact — one missing line away from publishing it.

## Notes

- **The gzipped database is not cacheable across deploys.** Every rebuild
  produces a different `.db.gz`, because SQLite does not lay pages out
  deterministically. First-visit users pay the full download; later visits hit
  browser cache until the next deploy.
- **GitHub Pages caps individual files at 100 MB.** The largest gzipped database
  is about 48 MB. Uncompressed they run 135–234 MB and would not fit — which is
  why the browser decompresses via `DecompressionStream`.
- **No server-side compression is assumed.** The app fetches the `.gz` bytes
  directly rather than relying on `Content-Encoding: gzip`; Pages does not
  reliably compress arbitrary paths on the fly.
- **Total artifact is about 177 MB**, well inside the 1 GB site limit.

## Rollback

Deploys are stateless snapshots. Revert the commit on `main` and push; the next
run rebuilds the older state. There is no data to migrate.

## Troubleshooting

| Symptom | Typical cause |
| --- | --- |
| Blank page, 404 on assets | `base` in `vite.config.js` does not match the repo name |
| `Failed to fetch database: 404` | Dataset id in `src/datasets.js` does not match the file in `db/` |
| A route 404s | `assemble-site.js` did not run, or the id is missing from `DATASETS` |
| WASM fails to load | `sql.js.org` unreachable — self-host `sql-wasm.wasm` and update `SQL_WASM_URL` in `use-sqlite.js` |
| Deploy fails on assembly | An uncompressed database artefact reached the output; the error names the files |
| Missing rows after a data update | Unknown Excel header — check the per-file row counts the parser prints |
