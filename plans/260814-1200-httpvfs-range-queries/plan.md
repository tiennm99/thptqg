# Serve the databases over HTTP range requests

Status: implemented, unverified in a browser

The whole-database download is gone. `sql.js-httpvfs` reads the pages a query
touches, so the databases ship raw as `<id>.sqlite3` — a byte range of a gzip
stream is not a byte range of a database.

## Why the schema had to change first

Measured on the real 2016 database (223.5 MB before, 4 KB pages, 27 rows/page):

| Query | Plan before | Would have fetched |
| --- | --- | --- |
| `so_bao_danh = ?` | SEARCH via PK | ~20 KB |
| `ho_ten_ascii LIKE '%x%'` | SCAN | 127 MB |
| `ho_ten_ascii LIKE 'x%'` | SCAN — the LIKE optimisation needs a NOCASE index | 127 MB |
| `COUNT(*)` | covering scan of idx_ho_ten_ascii | 20 MB |
| preset `ORDER BY toan DESC LIMIT 10` | SCAN + temp b-tree | 127 MB |

So substring search was impossible, prefix search was no better, and the footer
count alone cost 20 MB per page load.

## What shipped

**Parser.** `name_word(word, so_bao_danh, ho_ten_ascii)` WITHOUT ROWID — the
table is the index — plus `name_word_freq(word, n)` and partial indexes on
`toan`, `khtn`, `khxh`. Dropped `idx_ho_ten` and `idx_ho_ten_ascii`: no query
plan could use either.

877,460 names hold 2.87M word entries over a vocabulary of 4,397. A search asks
the frequency table which word is rarest, seeks on that one, and filters the
rest against the `ho_ten_ascii` copy inside the same b-tree — so "buu loc" still
finds "Nguyễn Bửu Lộc", in a few hundred KB.

| Segment | 2016 |
| --- | --- |
| `student` | 137.5 MB |
| `name_word` | 98.7 MB |
| `idx_ten_cum_thi` | 38.1 MB |
| PK autoindex | 15.3 MB |
| `idx_toan` | 12.6 MB |
| **total** | **302.4 MB** (2017: 247.3 MB) |

Written with 1 KiB pages, so a row reached by an index seek costs one 1 KB
request instead of 4 KB: 6.3 rows share a page rather than 27, which is what
turns a 100-row search from ~400 KB of row fetches into ~100 KB.

**Assembler.** Publishes uncompressed; the size guard reads the raw size; the
stray-artifact check now rejects journals, `.db` and `.gz`.

**Web.** `RemoteDatabase` wraps `createDbWorker`. The search tab runs with a
25 MB byte budget, the SQL tab asks for consent and then gets 250 MB, and the
bytes fetched are shown next to the query time. The footer count comes from
`datasets.json`.

## Verified

- Row counts unchanged: 877,460 and 861,068, both through the assembler guards.
- Every query the app issues is index-driven, checked with `EXPLAIN QUERY PLAN`:
  `SEARCH w USING PRIMARY KEY (word>? AND word<?)` then a PK seek per row.
- Real queries seek on 287–17,000 entries: `nguyen buu l` → 287, `tran thi
  phuoc an` → 3,684, `nguyen minh tien` → 16,996.
- GitHub Pages serves byte ranges: `206` with a correct `Content-Range`.

## Not verified

- **Nothing has run in a browser.** No browser exists on the build machine.
- **Per-query bytes are a structural estimate**, not a measurement — the byte
  counter in the SQL tab is the real check, once deployed.
- **`Content-Encoding` on the deployed file.** The library throws if the host
  compresses it. `curl -sI …/db/2016.sqlite3` must show none.
- **A 289 MB file on Pages.** No per-file limit is documented and the site is
  528 MB against a 1 GB limit, but this is the first deploy at that size. The
  fallback is the library's chunked mode.
