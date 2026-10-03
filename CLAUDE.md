# thptqg

Exam-score lookup for Vietnam's national high school graduation exam. Four
stages, each independent: `crawler/` (Go) fetches spreadsheets into `data/<id>/`,
`parser/` (Go) turns them into `<id>.sqlite3`, `assembler/` (Go) verifies them
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
- **`dbSizeMb` in `datasets.json` is a build guard and a user-facing figure.**
  The assembler refuses to publish an artifact that falls below a ratio of it,
  and the download gate shows it as the memory the tab will need.
- **The browser downloads the whole database and queries it in memory.** The
  dataset page is gated behind that download: `download-gate.svelte` states the
  transfer and memory cost and offers nothing but the download, because there
  is no answer to give without it. `sql.js` holds the file in WebAssembly
  memory for as long as the tab is open, so the memory figure is RAM, not disk.
- **The download is kept in Cache Storage, versioned by ETag** (`db-cache.js`).
  A later visit opens the stored copy without asking, since consent was given
  once and reuse costs no network; a redeploy changes the ETag, so the new
  version replaces the old rather than being served stale. When the server
  cannot be reached at all, any stored version is used — which is what lets the
  site answer offline.
- **The schema carries no secondary indexes, deliberately.** An index saves a
  scan that already takes a few hundred milliseconds in memory, and costs every
  visitor megabytes of download. An earlier design read the file over HTTP
  range requests and needed the opposite — a `name_word` table with one row per
  word of every name, plus partial score indexes — which was more than half the
  published file: 288 MB became 142 MB when they went.
- **The databases ship uncompressed, as `<id>.sqlite3`.** GitHub Pages gzips
  them on the wire anyway, which is where the transfer figure comes from
  (142 MB stored, 31 MB delivered), so publishing a `.gz` would only mean
  decompressing twice.

## Conventions

- Comments state current behavior and why. History belongs in git and `docs/`,
  not in code comments.
- Go files are CRLF in this checkout, so `gofmt -l` flags every file. It is not
  usable as a formatting gate as things stand.
- Conventional commits, no AI references.
