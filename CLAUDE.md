# thptqg

Exam-score lookup for Vietnam's national high school graduation exam. Four
stages, each independent: `crawler/` (Go) fetches spreadsheets into `data/<id>/`,
`parser/` (Go) turns them into `<id>.db`, `assembler/` (Go) verifies, compresses
and builds `_site/`, and `web/` (Vite + React) serves every dataset from one app.

`datasets.json` at the root is the registry the Go stages and the Vite app both
read. See [`docs/`](./docs/) for architecture, pipeline and deployment.

## Verifying a change

Run parser and assembler:

```bash
go -C parser test ./...
go -C assembler test ./...
```

Add the web checks when frontend files changed:

```bash
(cd web && npm run lint && npm run build)
```

**Do not run the crawler suite as part of routine verification.** The crawler is
not part of the build — `data/<id>/` is committed and a crawl only refreshes it
by hand — so its tests do not gate ordinary changes. Run them only when the
change actually touches `crawler/`. CI still runs all three modules.

`parser/internal/reader` is the slow package (~9s) because the fidelity suite
hashes every real input file. That is the point of it; do not skip it.

## Things that look wrong but are not

- **`data/2016/` filenames are content hashes and must stay verbatim.** The
  parser sorts inputs bytewise and inserts last-wins, so filenames decide which
  row survives a duplicate exam number — 877,464 source rows collapse to
  877,461. Renaming them also breaks `parser/testdata/reader-fidelity-hashes.tsv`,
  which is keyed by full path.
- **`parser/testdata/reader-fidelity-hashes.tsv` is frozen and cannot be
  regenerated.** A mismatch is a reader bug until proven otherwise, never a cue
  to refresh the file. `parser/cmd/dumpcells` narrows a failure to the cell.
- **`ToAscii` filters the literal range U+0300..U+036F, not `unicode.Mn`.** It
  must match `toAscii` in `web/src/App.jsx`, or accent-insensitive search
  silently misses rows.
- **`base` in `web/vite.config.js` is absolute.** One emitted `index.html` is
  copied to every dataset path, so every URL is a real static file and no SPA
  404-fallback is needed — a fallback would break the `?q=` deep links.
- **`dbSizeMb` in `datasets.json` is a build guard, not just a label.** The
  assembler refuses to publish an artifact that falls below a ratio of it.

## Conventions

- Comments state current behavior and why. History belongs in git and `docs/`,
  not in code comments.
- Go files are CRLF in this checkout, so `gofmt -l` flags every file. It is not
  usable as a formatting gate as things stand.
- Conventional commits, no AI references.
