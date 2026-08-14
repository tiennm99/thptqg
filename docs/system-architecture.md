# System Architecture

Static site, no backend. The browser downloads a compressed SQLite file at boot
and every query runs locally via `sql.js` (SQLite compiled to WebAssembly).

One frontend, one parser, one schema, two datasets.

## Data flow

Each stage is a directory; `data/` and `_site/` are the stores they hand work
through. `assembler/` sequences everything from the parser onwards.

```
        ▲  crawler/   (Go — manual refresh only, never part of the build)
data/<id>/*.xls(x)
        │
        ▼  parser/    (Go, one binary, one config per dataset)
   .build/public/db/<id>.db
        │
        ▼  assembler/ — row count must match datasets.json, then gzip
   .build/public/db/<id>.db.gz      (the raw .db does not survive)
        │
        ▼  assembler/ → vite build   (root = web/, publicDir = .build/public)
   web/dist/
        │
        ▼  assembler/ — one index.html per dataset; every database must be present
   _site/   →  GitHub Pages
        │
        ▼  browser
   sql.js (WASM) opens the .db → queries run client-side
```

## The dataset id

One identifier ties the whole pipeline together:

```
data/2017/  →  parser/configs/2017.yml  →  db/2017.db.gz  →  /thptqg/2017/
```

`datasets.json` at the repository root declares the ids once, with the row count
and artifact size the assembler enforces. It is JSON rather than a module
because the assembler is a Go program and the Vite app is not, and JSON is the
only format both parse without a dependency.

Presentation — titles, labels, search examples, SQL presets — stays in
`web/src/datasets.js`, keyed by id. That file cross-checks the two: a registry
entry with no content, or content for a dataset that was never built, throws at
module load rather than rendering a page with no title or a link to a database
that does not exist.

| id | Exam | Rows | Source |
| --- | --- | --- | --- |
| `2016` | 2016 | 877,461 | `dtnt.bacninh.edu.vn` |
| `2017` | 2017 | 861,068 | `baotintuc.vn` |

Full source URLs are in [data-pipeline](./data-pipeline.md#sources); the web
footer links to them per dataset.

## Canonical schema

Defined once in `parser/internal/schema/schema.go` — DDL, INSERT, column order and the 16
subject regexes. The two YAML configs carry no SQL at all, only per-dataset
parse rules. Config parsing sets `KnownFields(true)`, so a leftover `schema:`
block fails loudly instead of looking effective while `schema.go` drives the
build.

```sql
CREATE TABLE student (
  so_bao_danh   TEXT PRIMARY KEY,   -- 2017: 8 digits; 2016: 9 digits or a
                                    -- 2-4 letter cluster code then digits
  ho_ten        TEXT NOT NULL,
  ho_ten_ascii  TEXT NOT NULL,      -- NFD-stripped lowercase, for accent-insensitive search
  ngay_sinh     TEXT,               -- dd/mm/yyyy
  ten_cum_thi   TEXT,               -- 2016 only
  gioi_tinh     TEXT,               -- 2016 only
  toan, ngu_van, vat_ly, hoa_hoc, sinh_hoc, khtn,
  lich_su, dia_ly, gdcd, khxh,
  tieng_anh, tieng_phap, tieng_nga, tieng_duc, tieng_nhat, tieng_trung   REAL
);
CREATE INDEX idx_ho_ten       ON student(ho_ten);
CREATE INDEX idx_ho_ten_ascii ON student(ho_ten_ascii);
CREATE INDEX idx_ten_cum_thi  ON student(ten_cum_thi) WHERE ten_cum_thi IS NOT NULL;
```

Every dataset gets all 22 columns; ones it has no data for are NULL, costing
about a byte per row. `khtn`, `khxh` and `gdcd` are empty on 2016;
`ten_cum_thi` and `gioi_tinh` are empty on 2017.

`idx_ten_cum_thi` is partial, so it holds zero entries where the column is
always NULL.

## Routing

URLs are flat, one segment per dataset, and the segment is the id:

```
/thptqg/            hub
/thptqg/2016/
/thptqg/2017/
```

`web/src/router.js` is an exact match on that segment. The nested form used
before (`/thptqg/2017/old/`) would have needed longest-prefix matching, since it
also starts with `/thptqg/2017/`. Both of those URLs addressed the two removed
2017 archives, so the rewrite that kept them working has gone with them; they
fall through to the hub like any other unknown path.

A single Vite build emits one `index.html`. Because `base` is absolute
(`/thptqg/`), that file references `/thptqg/assets/...` regardless of the
directory it is served from, so the assemble step copies it to every route and
each URL is a real static file. **No SPA 404-fallback redirect is used** — the
usual hack rewrites URLs and would interfere with the deep links.

## Serving both exam years without branching

No component contains a per-dataset conditional. Two mechanisms do the work:

- **All-NULL columns are hidden.** `score-table.jsx` drops any column where
  every row in the result set is NULL, so 2016 rows surface Cụm thi / GT / Đức /
  Nhật and 2017 rows surface KHTN / KHXH / GDCD / Nga.
- **Incomplete admission blocks are skipped.** `computeBlocks()` only returns a
  block when the student has all three subjects, so one block list covers both
  years: GDCD blocks self-exclude on 2016, German and Japanese blocks
  self-exclude wherever those languages were not sat.

Anything genuinely per-dataset — title, source, database size, search examples,
SQL presets — lives in `web/src/datasets.js`.

## Exam ID formats

`web/src/lib/query-mode.js` decides whether a query is an exam ID or a name, and is
shared by `App.jsx` and `search-form.jsx` (they previously held separate copies
and had drifted apart on exactly this rule).

| Form | Example | Where |
| --- | --- | --- |
| 8 digits | `49008235` | 2017 — first two digits are the province |
| 9 digits with leading zero | `017006021` | 2016 |
| 2-4 letters then digits | `BAL000001` | 2016 — exam cluster code |

The letter-prefixed form is the majority case for 2016: 616,593 of 877,461
candidates (70.3%). Letter prefixes are upper-cased before lookup, so
`bal000001` resolves.

## Score tiers

Six-level ladder in `scoreTier()` (`web/src/lib/admission-blocks.js`), paired with a
symbol so meaning is never colour-only.

| Tier | Range | Vietnamese |
| --- | --- | --- |
| common | ≤ 1 | Điểm liệt |
| uncommon | < 5 | Chưa đạt |
| rare | 5–6.5 | Trung bình |
| epic | 6.5–8 | Khá |
| legendary | 8–9 | Giỏi |
| prismatic | 9–10 | Xuất sắc |

## Admission blocks

Vietnamese universities admit on three-subject combinations (khối thi).
`web/src/lib/admission-blocks.js` lists the blocks computable from this schema
(A00–A11, B00–B08, C00–C20, D01–D15, plus D05/D06 for German and Japanese).
`computeBlocks(student)` returns those where all three scores exist, sorted by
total descending.

## Design decisions

| Concern | Choice | Rationale |
| --- | --- | --- |
| Storage | Static SQLite file | No backend; the datasets are frozen |
| Compression | gzip in CI, `DecompressionStream` in the browser | Native API, no extra library |
| WASM hosting | `sql.js.org` CDN | Smaller self-hosted artifact |
| Diacritics search | Pre-computed `ho_ten_ascii` | `LOWER(REPLACE(...))` at query time defeats the index |
| SQL safety | Leading-keyword allowlist | `sql.js` is in-memory so writes cannot persist; the allowlist prevents confusion |
| Row caps | 100 (lookup), 1000 (SQL) | Keeps DOM render sizes reasonable |
| Routing | Hand-rolled, ~15 lines | Three static routes do not justify a router dependency |

## Risks and limitations

- **Database size.** 44–48 MB gzipped per dataset; slow links wait, mitigated by
  a progress bar.
- **Browser memory.** The full database lives in RAM; older mobile devices may
  run out.
- **`sql.js.org` dependency.** If that CDN is unreachable, the WASM fails to
  load. Self-hosting `sql-wasm.wasm` and updating `SQL_WASM_URL` in
  `web/src/hooks/use-sqlite.js` is the fix.
- **Excel format drift.** A new source file with an unseen header layout needs a
  new branch in `parser/internal/ingest/detect2016.go` or a new config.
