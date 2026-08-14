# System Architecture

Static site, no backend. The SQLite file stays on the server and the browser
reads the pages a query touches over HTTP range requests, via `sql.js-httpvfs`
(SQLite compiled to WebAssembly behind a virtual file system). A lookup costs a
few hundred KB; nothing downloads the database.

One frontend, one parser, one schema, two datasets.

## Data flow

Each stage is a directory; `data/` and `_site/` are the stores they hand work
through. `assembler/` sequences everything from the parser onwards.

```
        ▲  crawler/   (Go — manual refresh only, never part of the build)
data/<id>/*.xls(x)
        │
        ▼  parser/    (Go, one binary, one config per dataset)
   .build/public/db/<id>.sqlite30
        │
        ▼  assembler/ — row count and size must match datasets.json
   .build/public/db/<id>.sqlite30   (uncompressed: ranges of a gzip stream
        │                            are not ranges of the database)
        ▼  assembler/ → npm run build (SvelteKit static, assets = .build/public)
   web/dist/
        │
        ▼  assembler/ — one index.html per dataset; every database must be present
   _site/   →  GitHub Pages
        │
        ▼  browser
   sql.js-httpvfs asks for pages → HTTP range requests → results client-side
```

## The dataset id

One identifier ties the whole pipeline together:

```
data/2017/  →  parser/configs/2017.yml  →  db/2017.sqlite30  →  /thptqg/2017/
```

`datasets.json` at the repository root declares the ids once, with the row count
and artifact size the assembler enforces. It is JSON rather than a module
because the assembler is a Go program and the web app is not, and JSON is the
only format both parse without a dependency.

Presentation — titles, labels, search examples, SQL presets — stays in
`web/src/lib/datasets.js`, keyed by id. That file cross-checks the two: a registry
entry with no content, or content for a dataset that was never built, throws at
module load rather than rendering a page with no title or a link to a database
that does not exist.

| id | Exam | Rows | Source |
| --- | --- | --- | --- |
| `2016` | 2016 | 877,460 | `dtnt.bacninh.edu.vn` |
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

The route is `web/src/routes/[dataset]/`, and its entry generator reads the same
`datasets.json` the assembler does, so the set of pages and the set of databases
cannot drift apart. Unknown paths fall through to the hub.

SvelteKit prerenders one HTML file per route, each with its own `<title>`.
Asset URLs stay absolute (`paths.relative: false`), so the copy of the hub that
serves as `404.html` resolves its assets from any depth. **No SPA 404-fallback
redirect is used** — the usual hack rewrites URLs and would interfere with the
deep links.

## Serving both exam years without branching

No component contains a per-dataset conditional. Two mechanisms do the work:

- **All-NULL columns are hidden.** `score-table.svelte` drops any column where
  every row in the result set is NULL, so 2016 rows surface Cụm thi / GT / Đức /
  Nhật and 2017 rows surface KHTN / KHXH / GDCD / Nga.
- **Incomplete admission blocks are skipped.** `computeBlocks()` only returns a
  block when the student has all three subjects, so one block list covers both
  years: GDCD blocks self-exclude on 2016, German and Japanese blocks
  self-exclude wherever those languages were not sat.

Anything genuinely per-dataset — title, source, database size, search examples,
SQL presets — lives in `web/src/lib/datasets.js`.

## Exam ID formats

`web/src/lib/query-mode.js` decides whether a query is an exam ID or a name, and
is shared by the dataset page and `search-form.svelte` (they previously held
separate copies and had drifted apart on exactly this rule). `query-mode.test.js`
covers every form in the table below.

| Form | Example | Where |
| --- | --- | --- |
| 8 digits | `49008235` | 2017 — first two digits are the province |
| 9 digits with leading zero | `017006021` | 2016 |
| 2-4 letters then digits | `BAL000001` | 2016 — exam cluster code |

The letter-prefixed form is the majority case for 2016: 624,424 of 877,460
candidates (71.2%); the remaining 253,036 are all 9-digit. Letter prefixes are
upper-cased before lookup, so `bal000001` resolves.

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
| Storage | Static SQLite file, read by range request | No backend; the datasets are frozen, and a lookup needs a few pages of them |
| Reading mode | `serverMode: "chunked"` over a single chunk | The only mode whose config accepts the file length. In full mode the worker hardcodes it to `undefined` and falls back to a HEAD request, which Pages answers with the gzipped size. One chunk means the index is always 0, hence the published name `<id>.sqlite30` |
| Compression | None | A byte range of a gzip stream is not a byte range of the database |
| WASM hosting | Bundled with the app | `sql.js-httpvfs` ships its own build; one less third-party runtime dependency |
| Diacritics search | Pre-computed `ho_ten_ascii`, indexed word by word | `LOWER(REPLACE(...))` at query time defeats the index, and `LIKE '%x%'` reads the whole table |
| Row count in the footer | Read from `datasets.json` | `COUNT(*)` scans an index — 20 MB over range requests |
| Page size | 4 KiB, matched by `requestChunkSize` | One HTTP request is one page, and the worker issues them serially over synchronous XHR at ~40 ms each. Request count, not size, is what a search waits on: 390 requests moved 608 KB in 17 s. Four times the page is a quarter of the pages per scan. The 1 KiB both httpvfs libraries suggest optimises bytes per seek instead |
| SQL safety | Leading-keyword allowlist | `sql.js` is in-memory so writes cannot persist; the allowlist prevents confusion |
| Row caps | 100 (lookup), 1000 (SQL) | Keeps DOM render sizes reasonable |
| Routing | SvelteKit file routes, prerendered | Each dataset gets a real HTML file with its own title |
| Styling | Tailwind, with tier colours as CSS variables | Tier classes are chosen at runtime, which no utility generator can see |

### Considered and not taken

- **Splitting the database into several chunks.** The site uses chunked mode,
  but over one chunk (see above). Real splitting would let a CDN cache each part
  whole; GitHub Pages serves everything with `Cache-Control: max-age=600`, and
  every rebuild relays SQLite's pages so the file changes even when the data
  does not, so that caching is cancelled by the host. Worth revisiting behind a
  CDN with long TTLs, and it is the fallback if a single 288 MB file ever
  becomes a problem.
- **`sqlite-wasm-http`.** Maintained, and built on the official SQLite WASM
  rather than a 2022 fork, which is the better long-term footing. It does not
  help here: its worker sizes the file from a HEAD request's `Content-Length`
  exactly as `sql.js-httpvfs` does, and its `Options` has no field for the
  length, so on Pages it would silently take the gzipped size instead of
  failing. Its shared-cache backend needs COOP/COEP, which Pages cannot send,
  but it ships a fallback backend that does not — so isolation is not the
  blocker, the missing length option is.
- **Substring name search.** `LIKE '%x%'` cannot use an index, so it read the
  whole student table. `name_word` keeps search by any word of a name without
  it.

## Risks and limitations

- **Unindexed queries are expensive.** The SQL tab can express a query that
  walks the table, which over range requests means fetching 100+ MB. A byte
  budget stops one before it gets that far, and the tab warns before it opens.
- **`Content-Encoding` on a ranged response would break everything.** A range
  of a compressed body addresses the wrong bytes. In practice browsers prevent
  it: the Fetch standard requires `Accept-Encoding: identity` on any request
  carrying a `Range` header. GitHub Pages *does* gzip the un-ranged response —
  `application/octet-stream` is compressible in `mime-db` — which is why the
  file length is probed with a range request and passed as
  `databaseLengthBytes` rather than left to the library's HEAD.
  `db-probe.js` checks the returned bytes
  start with the SQLite magic, so a host that ever compresses a ranged response
  fails loudly instead of returning nonsense.
- **`sql.js-httpvfs` is unmaintained** (0.8.12, September 2022) and ships its
  own SQLite WASM. `sqlite-wasm-http`, on the official build, is the fallback.
- **Hosted size.** 526 MB for both datasets against the 1 GB GitHub Pages
  limit; a third dataset of this size would not fit.
- **Excel format drift.** A new source file with an unseen header layout needs a
  new branch in `parser/internal/ingest/detect2016.go` or a new config.
