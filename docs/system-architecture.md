# System Architecture

Static site, no backend. The browser downloads the whole SQLite file once and
queries it in memory with `sql.js` (SQLite compiled to WebAssembly). A dataset
costs about 31 MB to fetch and 142 MB of memory to hold; every query after that
is local — an exam-number lookup is immediate, and a name search scans all
877,460 rows in a few hundred milliseconds.

That is a reversal of the previous design, which read the file where it lay
over HTTP range requests. See [Considered and not taken](#considered-and-not-taken).

One frontend, one parser, one schema, two datasets.

## Data flow

Each stage is a directory; `data/` and `_site/` are the stores they hand work
through. `assembler/` sequences everything from the parser onwards.

```
        ▲  crawler/   (Go — manual refresh only, never part of the build)
data/<id>/*.xls(x)
        │
        ▼  parser/    (Go, one binary, one config per dataset)
   .build/public/db/<id>.sqlite3
        │
        ▼  assembler/ — row count and size must match datasets.json
   .build/public/db/<id>.sqlite3    (uncompressed: the host gzips it on the
        │                            wire, so a .gz would decompress twice)
        ▼  assembler/ → npm run build (SvelteKit static, assets = .build/public)
   web/dist/
        │
        ▼  assembler/ — one index.html per dataset; every database must be present
   _site/   →  GitHub Pages
        │
        ▼  browser
   download gate → whole file → sql.js in memory → queries never leave the tab
```

## The dataset id

One identifier ties the whole pipeline together:

```
data/2017/  →  parser/configs/2017.yml  →  db/2017.sqlite3   →  /thptqg/2017/
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
| Storage | Static SQLite file, downloaded whole | No backend; the datasets are frozen, and one large transfer is something browsers and CDNs are both good at |
| Download gate | Blocking, no dismiss | The page has no answers before the file arrives, and 31 MB of someone's mobile data should be asked for rather than spent silently |
| Compression | None published | The host gzips on the wire, so a `.gz` artifact would only be decompressed twice |
| WASM hosting | Bundled with the app | `sql.js` ships its own build; one less third-party runtime dependency |
| Diacritics search | Pre-computed `ho_ten_ascii` | `LOWER(REPLACE(...))` at query time is far slower over 877,460 rows than a column computed once at build |
| Secondary indexes | None | In memory a full scan costs a few hundred milliseconds; an index costs every visitor megabytes of download. The name index alone was 146 MB |
| Row count in the footer | Read from `datasets.json` | Costs nothing and is the same number the assembler enforces |
| Page size | 4 KiB | SQLite's default, and nothing on the client depends on it any more |
| SQL safety | Leading-keyword allowlist | The copy is the visitor's own, so this guards their session against a typo rather than protecting data |
| Row caps | 100 (lookup), 1000 (SQL) | Keeps DOM render sizes reasonable |
| Routing | SvelteKit file routes, prerendered | Each dataset gets a real HTML file with its own title |
| Styling | Tailwind, with tier colours as CSS variables | Tier classes are chosen at runtime, which no utility generator can see |

### Considered and not taken

- **Reading the file over HTTP range requests** (`sql.js-httpvfs`), which this
  site did until it was measured. Two costs killed it. Reads are serial —
  the worker uses synchronous XHR — so a name search that touched 390 pages
  waited 17 seconds to move 608 KB, and roughly one request per result row is a
  floor no page size removes. And the first visitor after each deploy waited
  ~26 s for the CDN to fill its cache with a 288 MB object. The download pays
  once, up front, visibly.
- **`sqlite-wasm-http`.** Built on the official SQLite WASM and maintained,
  but it sizes the file from a HEAD request's `Content-Length` and exposes no
  option to override it, so on a host that gzips it would silently use the
  compressed size. Same class of problem, less recourse.
- **DuckDB-WASM over Parquet.** Genuinely maintained, async, parallel range
  requests. Its binaries are 32–37 MB before the Parquet extension, which is
  more than the entire database download for a phone looking up one score.
- **Static pre-generated shards**, one file per exam-number bucket. The most
  robust option and the fastest single lookup, but it cannot answer arbitrary
  SQL, and a static file cannot stop early the way `LIMIT` does — a common
  Vietnamese surname would mean fetching a very large posting list.

## Risks and limitations

- **Memory is the binding constraint.** The database lives in the tab's
  WebAssembly memory for as long as the page is open: 142 MB for 2016, 119 MB
  for 2017. A low-memory phone may have the tab killed, which is why the gate
  states the figure before the download starts.
- **The download is repeated every visit.** Nothing is persisted — a reload
  fetches the file again. Caching it in IndexedDB or the Cache API would fix
  that and has not been done.
- **A visitor who will not download cannot use the site.** That is the
  deliberate shape of the gate, and it makes the first impression a 31 MB ask.
- **Hosted size.** 526 MB for both datasets against the 1 GB GitHub Pages
  limit; a third dataset of this size would not fit.
- **Excel format drift.** A new source file with an unseen header layout needs a
  new branch in `parser/internal/ingest/detect2016.go` or a new config.
