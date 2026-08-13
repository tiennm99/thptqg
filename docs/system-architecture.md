# System Architecture

Static site, no backend. The browser downloads a compressed SQLite file at boot
and every query runs locally via `sql.js` (SQLite compiled to WebAssembly).

One frontend, one parser, one schema, four datasets.

## Data flow

```
data/<id>/*.xls(x)
        │
        ▼  parser/  (Rust, one binary, one config per dataset)
   .build/public/db/<id>.db
        │
        ▼  gzip -9   (no -k: the raw file does not survive)
   .build/public/db/<id>.db.gz
        │
        ▼  vite build   (publicDir = .build/public)
   dist/
        │
        ▼  scripts/assemble-site.js
   _site/   →  GitHub Pages
        │
        ▼  browser
   sql.js (WASM) opens the .db → queries run client-side
```

## The dataset id

One identifier ties the whole pipeline together:

```
data/2017-old/  →  parser/configs/2017-old.toml  →  db/2017-old.db.gz  →  /thptqg/2017-old/
```

`src/datasets.js` declares the four ids once. The frontend, the database build
(`parser/scripts/build-db.js`) and the site assembly all import that list, so
adding a dataset means adding one entry and one config file.

| id | Exam | Rows | Source |
| --- | --- | --- | --- |
| `2016` | 2016 | 877,461 | Bộ GD&ĐT |
| `2017` | 2017 | 861,068 | baotintuc.vn |
| `2017-old` | 2017 | 847,348 | pre-refresh archive |
| `2017-old2` | 2017 | 679,764 | corrected re-export |

## Canonical schema

Defined once in `parser/src/schema.rs` — DDL, INSERT, column order and the 16
subject regexes. The four TOML configs carry no SQL at all, only per-dataset
parse rules. Config parsing uses `deny_unknown_fields`, so a leftover `[schema]`
block fails loudly instead of looking effective while `schema.rs` drives the
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
`ten_cum_thi` and `gioi_tinh` are empty on the 2017 datasets.

`idx_ten_cum_thi` is partial, so it holds zero entries where the column is
always NULL.

## Routing

URLs are flat, one segment per dataset, and the segment is the id:

```
/thptqg/            hub
/thptqg/2016/
/thptqg/2017/
/thptqg/2017-old/
/thptqg/2017-old2/
```

`src/router.js` is an exact match on that segment. The nested form used before
(`/thptqg/2017/old/`) would have needed longest-prefix matching, since it also
starts with `/thptqg/2017/`. Those two legacy URLs still resolve: the router
rewrites them to the flat equivalent with `history.replaceState`, preserving the
query string so `?q=` deep links survive.

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
SQL presets — lives in `src/datasets.js`.

## Exam ID formats

`src/lib/query-mode.js` decides whether a query is an exam ID or a name, and is
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

Six-level ladder in `scoreTier()` (`src/lib/admission-blocks.js`), paired with a
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
`src/lib/admission-blocks.js` lists the blocks computable from this schema
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
| Routing | Hand-rolled, ~20 lines | Five static routes do not justify a router dependency |

## Risks and limitations

- **Database size.** 38–48 MB gzipped per dataset; slow links wait, mitigated by
  a progress bar.
- **Browser memory.** The full database lives in RAM; older mobile devices may
  run out.
- **`sql.js.org` dependency.** If that CDN is unreachable, the WASM fails to
  load. Self-hosting `sql-wasm.wasm` and updating `SQL_WASM_URL` in
  `use-sqlite.js` is the fix.
- **Excel format drift.** A new source file with an unseen header layout needs a
  new branch in `format_detect_2016.rs` or a new config.
