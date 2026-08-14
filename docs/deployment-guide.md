# Deployment Guide

Deploys to GitHub Pages via `.github/workflows/deploy-pages.yml`. Every push to
`main` rebuilds both datasets and redeploys the whole site.

One-time setup: **Settings → Pages → Source: GitHub Actions**.

## What the workflow does

1. Checkout, Go toolchain, Node 24, `npm ci` in `web/`
2. Parser, crawler and assembler test suites, web lint, `govulncheck` over all three modules
3. `go -C assembler run ./cmd/assemble` — the whole pipeline: compile the
   parser, build and verify each database, compress it into `.build/public/db/`,
   run the web build, assemble `_site/`
4. `actions/upload-pages-artifact` + `actions/deploy-pages`

The database build dominates the runtime: roughly 348 MB of Excel is parsed on
every deploy.

## Resulting URLs

```
https://<user>.github.io/thptqg/
https://<user>.github.io/thptqg/2016/
https://<user>.github.io/thptqg/2017/
```

`/thptqg/2017/old/` and `/thptqg/2017/old2/` were the pre-flattening URLs for
the two removed 2017 archives. They are no longer served; like any unknown path
they now render the hub via `404.html`.

## Local reproduction

```bash
(cd web && npm ci)
go -C assembler run ./cmd/assemble
npx serve _site
```

To rebuild a single dataset, or only the site:

```bash
go -C assembler run ./cmd/assemble db 2017
go -C assembler run ./cmd/assemble site
```

## Base path

`svelte.config.js` sets `paths.base: "/thptqg"`. If you fork under a different
repo name, update it to match — assets are referenced absolutely, so a mismatch
shows up as a blank page with 404s on `/_app/...`.

## Adding a dataset

1. Put the Excel files in `data/<id>/`
2. Add `parser/configs/<id>.yml` with the parse rules — sheet mode, column
   indices, SBD validation, header tokens, blank-row stripping. No SQL: the
   schema is canonical and lives in `parser/internal/schema/schema.go`
3. Add an entry to `datasets.json` — id, `expectedRows`, `dbSizeMb`
4. Add its presentation to `CONTENT` in `web/src/datasets.js`

Nothing else. The assembler and the router both read the registry, and the
frontend adapts to whichever columns the dataset populates. The last two steps
check each other, so forgetting either fails rather than half-working.

## Why no uncompressed database can ship

The assembler deletes the source once compression succeeds, so the raw file
does not survive the build, and it then fails the job if any `.db`,
`.db-journal`, `.db-wal` or `.db-shm` reached the output.

Both guards exist because the previous pipeline wrote a 100+ MB uncompressed
database into the source tree and relied on an `rm` step to keep it out of the
artifact — one missing line away from publishing it.

## Notes

- **The database is not cacheable across deploys.** Every rebuild lays SQLite
  pages out differently, so the file changes even when the data does not. Only
  the pages a query touches are fetched, so this costs far less than it used
  to, but a deploy does invalidate what a returning visitor had cached.
- **The 100 MB file limit is a Git limit, not a Pages one.** It applies to
  files committed to a repository; the databases are built in CI and uploaded
  as a Pages artifact, and the documented Pages limits are a 1 GB published
  site and 100 GB/month of bandwidth, with no per-file figure. The two
  databases are 302 MB and 247 MB.
- **Total artifact is about 552 MB**, inside the 1 GB site limit but with less
  headroom than before: a third dataset of this size would not fit. The fallback
  is `sql.js-httpvfs`'s chunked mode, which splits a database into parts.
- **The server must not compress the databases.** Ranges of a compressed body
  address the wrong bytes, and the library refuses to open a file whose HEAD
  carries a `Content-Encoding`. `.sqlite3` is an unknown type to Pages, so it is
  served as `application/octet-stream` and left alone — verify after a deploy.

## Rollback

Deploys are stateless snapshots. Revert the commit on `main` and push; the next
run rebuilds the older state. There is no data to migrate.

## Troubleshooting

| Symptom | Typical cause |
| --- | --- |
| Blank page, 404 on assets | `paths.base` in `svelte.config.js` does not match the repo name |
| `Failed to fetch database: 404` | Dataset id in `datasets.json` does not match the file in `db/` |
| A route 404s | The site step did not run, or the id is missing from `datasets.json` |
| Database fails to open | The host compressed it. `curl -sI …/db/<id>.sqlite3` must show no `content-encoding`; ranges of a compressed body are unusable |
| Every query is slow or huge | It is not using an index. `EXPLAIN QUERY PLAN` it: a `SCAN` means the browser is fetching the whole table |
| Deploy fails on assembly | An uncompressed database artefact reached the output; the error names the files |
| Missing rows after a data update | Unknown Excel header — check the per-file row counts the parser prints |
