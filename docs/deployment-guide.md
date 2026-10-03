# Deployment Guide

Deploys to GitHub Pages via `.github/workflows/deploy-pages.yml`. Every push to
`main` rebuilds both datasets and redeploys the whole site.

One-time setup: **Settings → Pages → Source: GitHub Actions**.

## What the workflow does

1. Checkout, Go toolchain (version taken from `parser/go.mod`), Node 24,
   `npm ci` in `web/`
2. Parser, crawler and assembler test suites, `npm test` and `npm run lint` in
   `web/`, `govulncheck` over all three Go modules
3. `go -C assembler run ./cmd/assemble` — the whole pipeline: compile the
   parser, build and verify each database into `.build/public/db/`, run the web
   build, assemble `_site/`. The databases are restored from the Actions cache
   when nothing that determines them has changed, and only the site is built
4. `actions/upload-pages-artifact` + `actions/deploy-pages`
5. After deploying, each published database is read back from the live URL and
   must begin `SQLite format 3` — proof the site serves a database rather than
   an error page or a truncated upload

Pull requests run steps 1–3 and stop. The deploy job is guarded to `main`, so a
branch is verified end to end without touching the live site. The concurrency
group is keyed by ref rather than shared, because a shared lane let a
pull-request run cancel an in-flight `main` deploy while every check stayed
green.

The database build dominates the runtime: roughly 348 MB of Excel to parse. It
is cached in Actions, keyed on `data/**`, `parser/**` and `datasets.json`, so a
push that touches none of those — web code, `docs/`, the workflow — restores the
databases instead of rebuilding them. There are no `restore-keys`: a near-miss
would publish databases built from inputs the commit does not describe.

The key is by path, not by what the change means, so it over-invalidates: a
comment in `datasets.json` or a word in `parser/README.md` costs a full rebuild
though neither can move a row. That is the safe direction to be wrong in, and
narrowing the globs would mean remembering to extend them for every future path
that *can* change a database. The cost is minutes on the occasional docs push.

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
4. Add its presentation to `CONTENT` in `web/src/lib/datasets.js`

Nothing else. The assembler and the router both read the registry, and the
frontend adapts to whichever columns the dataset populates. The last two steps
check each other, so forgetting either fails rather than half-working.

## What the databases are published as

One uncompressed `<id>.sqlite3` per dataset. Pages gzips it on the wire anyway,
so a `.gz` artifact would only mean decompressing twice.

`.sqlite3` is the only accepted name, and that is deliberate: it leaves `.db`
free, so site assembly can treat any stray `.db`, `.db.gz`, `.sqlite30` or
SQLite journal (`-journal`, `-wal`, `-shm`) in the output as the mistake it is
and fail the job. An earlier pipeline wrote a database into the source tree and
relied on an `rm` step to keep it out of the artifact — one missing line away
from publishing something it did not mean to.

## Notes

- **A deploy invalidates what returning visitors kept.** Every rebuild lays
  SQLite pages out differently, so the file — and the ETag the browser stored it
  under — changes even when the data does not. The visitor downloads once more
  and keeps that copy until the next deploy.
- **The 100 MB file limit is a Git limit, not a Pages one.** It applies to
  files committed to a repository; the databases are built in CI and uploaded
  as a Pages artifact, and the documented Pages limits are a 1 GB published
  site and 100 GB/month of bandwidth, with no per-file figure. The two
  databases are 142 MB and 119 MB.
- **Total artifact is about 263 MB**, comfortably inside the 1 GB site limit.
  Dropping the search index the range-request design needed halved both files.
- **Pages compresses the databases on the wire, which is a benefit here.** The
  extension is unknown to Pages, so the file is served as
  `application/octet-stream`, which `mime-db` marks compressible, and the
  browser decompresses it transparently: 142 MB stored becomes about 31 MB
  delivered for 2016, and 119 MB becomes about 36 MB for 2017. Note that the
  smaller database is the larger download: the two compress differently, so
  neither transfer figure can be derived from the stored size.
- **Verify the bytes, not the headers.** A bare `curl -sI` reports success
  whatever the host does. Read the file's first bytes instead:

  ```bash
  curl -s -r 0-15 https://<user>.github.io/thptqg/db/2016.sqlite3 | head -c 16
  # must print: SQLite format 3
  ```

## Rollback

Deploys are stateless snapshots. Revert the commit on `main` and push; the next
run rebuilds the older state. There is no data to migrate.

## Troubleshooting

| Symptom | Typical cause |
| --- | --- |
| Blank page, 404 on assets | `paths.base` in `svelte.config.js` does not match the repo name |
| `Failed to fetch database: 404` | Dataset id in `datasets.json` does not match the file in `db/` |
| A route 404s | The site step did not run, or the id is missing from `datasets.json` |
| Download gate never finishes | The file is not being served, or the tab ran out of memory holding it. The check above must print `SQLite format 3` |
| Tab crashes on a phone | The database needs 142 MB of memory for 2016, 119 MB for 2017; a low-memory device may have the tab killed |
| Deploy fails on assembly | A stray database artefact reached the output — a journal, a `.gz`, or a name from an earlier design; the error names the files |
| Missing rows after a data update | Unknown Excel header — check the per-file row counts the parser prints |
| Visitor still sees old data after a deploy | Their stored copy is keyed by ETag, so this should not happen. Confirm the deployed file's `ETag` header actually changed |
