# System Architecture

## Overview

A **static serverless** design: the entire dataset is packaged into a single SQLite file, gzip-compressed, and served as a static asset via GitHub Pages. The browser downloads it, decompresses it, and queries it in-process using `sql.js` (SQLite compiled to WebAssembly).

```
┌─────────────────┐   build      ┌──────────────────────┐
│  data/*.xlsx    │ ───────────▶ │ scripts/             │
│  (mixed formats)│              │  build-database.js   │
└─────────────────┘              │  (Node + xlsx +      │
                                 │   better-sqlite3)    │
                                 └──────────┬───────────┘
                                            │
                                            ▼
                                 ┌──────────────────────┐
                                 │ public/thptqg2016.db │
                                 └──────────┬───────────┘
                                            │ gzip -9 (CI)
                                            ▼
                                 ┌──────────────────────┐
                                 │ dist/thptqg2016.db.gz│
                                 │ dist/assets/*        │  ◀── Vite build
                                 └──────────┬───────────┘
                                            │ upload-pages-artifact
                                            ▼
                                    GitHub Pages CDN
                                            │
                                            ▼
                         ┌──────────────────────────────────┐
                         │ Browser                          │
                         │  ┌────────────────────────────┐  │
                         │  │ useSqlite hook             │  │
                         │  │  fetch(.db.gz) + stream    │  │
                         │  │  DecompressionStream gzip  │  │
                         │  │  sql.js WASM (from CDN)    │  │
                         │  └─────────────┬──────────────┘  │
                         │                ▼                  │
                         │  ┌────────────────────────────┐  │
                         │  │ React UI                   │  │
                         │  │  - SearchForm / ScoreTable │  │
                         │  │  - CustomQuery (SQL editor)│  │
                         │  └────────────────────────────┘  │
                         └──────────────────────────────────┘
```

## Build-time data flow

1. Developer drops Excel files into `data/`.
2. `npm run build:db` reads every file; for each one:
   - Detects the header row against a `KNOWN_HEADERS` set.
   - Picks a format: `separate-scores` (one column per subject) vs. `mapped` (single `DIEM_THI` string).
   - Parses each row into a canonical 18-column record.
   - `INSERT OR REPLACE` into SQLite (primary key = `so_bao_danh` handles duplicates).
3. `VACUUM` shrinks the file.
4. CI runs `gzip -k -9` → `.db.gz`.

## Runtime flow

1. Page loads → React mounts → `useSqlite("thptqg2016.db.gz")`.
2. Streaming fetch with a progress bar (driven by `Content-Length`).
3. `DecompressionStream("gzip")` decompresses on the fly.
4. `sql.js` loads its WASM from `https://sql.js.org/dist/sql-wasm.wasm`.
5. `new SQL.Database(Uint8Array)` — DB is now in RAM.
6. Each search / query → `db.prepare()` + `stmt.step()` loop → render.

## Design decisions

| Concern | Choice | Rationale |
|---------|--------|-----------|
| Storage | Static SQLite file | No backend needed; dataset is frozen |
| Compression | gzip in CI, `DecompressionStream` in browser | Native browser API; no extra library |
| WASM hosting | `sql.js.org` CDN | Smaller self-hosted artifact |
| Diacritics search | Pre-computed `ho_ten_ascii` column | `LOWER(REPLACE(...))` at query time defeats the index |
| SQL safety | Leading-keyword allowlist | `sql.js` is in-memory so writes don't persist, but the allowlist prevents user confusion |
| Row caps | 100 (lookup), 1000 (SQL) | Keep DOM render sizes reasonable |

## Risks and limitations

- **DB size**: tens of MB gzipped — slow links have a visible wait; mitigated by the progress bar.
- **Browser memory**: the full DB lives in RAM; older mobile devices may OOM.
- **Dependency on `sql.js.org`**: if that CDN is unreachable, WASM fails to load.
- **Excel format drift**: a new source file with an unseen header layout needs a new branch in `detectFormat()`.
