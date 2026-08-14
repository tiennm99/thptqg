# thptqg

Exam-score lookup for Vietnam's national high school graduation exam. Four
stages, each independent: `crawler/` (Go) fetches spreadsheets into `data/<id>/`,
`parser/` (Go) turns them into `<id>.db`, `assembler/` (Go) verifies, compresses
and builds `_site/`, and `web/` (SvelteKit + Tailwind) serves every dataset from
one app.

`datasets.json` at the root is the registry the Go stages and the web app both
read. See [`docs/`](./docs/) for architecture, pipeline and deployment.

## Verifying a change

Run parser and assembler:

```bash
go -C parser test ./...
go -C assembler test ./...
```

Add the web checks when frontend files changed:

```bash
(cd web && npm test && npm run lint && npm run build)
```

`npm run lint` is ESLint. The web app is plain JavaScript — no TypeScript, no
type-check step.

**Do not run the crawler suite as part of routine verification.** The crawler is
not part of the build — `data/<id>/` is committed and a crawl only refreshes it
by hand — so its tests do not gate ordinary changes. Run them only when the
change actually touches `crawler/`. CI still runs all three modules.

`parser/internal/reader` is the slow package (~9s) because the fidelity suite
hashes every real input file. That is the point of it; do not skip it.

## Things that look wrong but are not

- **`data/2016/` filenames are content hashes and must stay verbatim.** The
  parser sorts inputs bytewise and inserts last-wins, so filenames would decide
  which row survives a duplicate exam number. Renaming them also breaks
  `parser/testdata/reader-fidelity-hashes.tsv`, which is keyed by full path.
- **`parser/testdata/reader-fidelity-hashes.tsv` is frozen and cannot be
  regenerated.** A mismatch is a reader bug until proven otherwise, never a cue
  to refresh the file. `parser/cmd/dumpcells` narrows a failure to the cell.
- **`ToAscii` filters the literal range U+0300..U+036F, not `unicode.Mn`.** It
  must match `toAscii` in `web/src/lib/to-ascii.js`, or accent-insensitive search
  silently misses rows. `to-ascii.test.js` pins the pairs both sides must agree
  on.
- **The 2016 files use four different layouts, and detection is per sheet.**
  Two of them publish scores in one column per subject instead of a `DIEM_THI`
  sentence, and one puts a three-row ministry title block above its header.
  `parser/internal/ingest/detect2016.go` holds the header tables; they are
  observations about 119 specific files, not a rule to generalise.
- **`paths.relative` is false in `web/svelte.config.js`.** Every route is
  prerendered to its own file and asset URLs stay absolute, so the copy of
  `index.html` serving as `404.html` works at any depth. No SPA 404-fallback is
  used — a fallback would break the `?q=` deep links.
- **`dbSizeMb` in `datasets.json` is a build guard, not just a label.** The
  assembler refuses to publish an artifact that falls below a ratio of it.
- **The databases ship uncompressed, as `<id>.sqlite3`.** The browser reads
  byte ranges of them, and a range of a gzip stream is not a range of the
  database. The host must not apply `Content-Encoding` either — check with
  `curl -sI` after a deploy.
- **Every query the site runs must be index-driven.** Over range requests an
  unindexed query fetches the whole table. Hence no index on `ho_ten` (nothing
  can use one), `name_word` for name search, partial indexes for the score
  presets, and the footer count read from `datasets.json` instead of
  `COUNT(*)`.

## Conventions

- Comments state current behavior and why. History belongs in git and `docs/`,
  not in code comments.
- Go files are CRLF in this checkout, so `gofmt -l` flags every file. It is not
  usable as a formatting gate as things stand.
- Conventional commits, no AI references.
