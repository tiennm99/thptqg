# Research Report: sql.js-httpvfs best practices

Conducted 2026-08-14 13:17 (Asia/Saigon). Context: two static SQLite files
(2016 = 288.6 MB, 2017 = 237.7 MB) on GitHub Pages, branch
`feat/httpvfs-range-queries`.

## Executive summary

Our implementation matches upstream guidance on the thing that matters most —
index design — and diverges on one measurable parameter: **page size**. Both
phiresky (sql.js-httpvfs) and mmomtchev (sqlite-wasm-http) recommend
`page_size = 1024`; we shipped 4096. For our access pattern (~100 scattered
single-row reads per search) that is a real 4× overfetch on the row-fetch half
of a query.

Two findings reduce risk rather than add work. Prefetching with three virtual
read heads makes a sequential scan cost a *logarithmic* number of requests, so
our byte estimates hold but latency is better than assumed. And the canonical
demo hosts a **670 MiB** database on GitHub Pages, so 289 MB is precedented.

One finding is new and worth a decision: **chunked mode** (split file + JSON
config) exists specifically to make CDN caching effective for large databases,
which matters because GitHub Pages serves everything with `Cache-Control:
max-age=600`.

## Methodology

- Sources: 5 (1 primary blog, 2 project READMEs, 1 recent practitioner
  writeup, 1 search on Pages/Fastly caching), plus direct reading of the
  installed `sql.js-httpvfs@0.8.12` bundle in a previous session.
- Date range: 2021 (canonical post) → March 2026 (practitioner writeup).
- Gemini CLI absent → WebSearch/WebFetch.

## Key findings

### 1. Page size: recommended 1024, we use 4096

Both projects say the same thing. phiresky set 1 KiB pages "to balance request
overhead against bandwidth efficiency"; sqlite-wasm-http says "it is highly
recommended to decrease your SQLite page size to 1024 bytes for maximum
performance" (`PRAGMA page_size=1024; VACUUM`).

`requestChunkSize` must match the page size.

Measured on our 2016 file *before* the schema change:

| page_size | file size |
| --- | --- |
| 1024 | 235.8 MB |
| 4096 | 223.5 MB |
| 8192 | 221.8 MB |

So 1024 costs ~5.5% file size. What it buys: a scattered row read fetches 1 KB
instead of 4 KB. Our search does ~100 of those, so the row-fetch half of a
search drops from ~400 KB to ~100 KB. The index-walk half is sequential and
benefits from prefetch either way.

**Verdict: switch to 1024 + `requestChunkSize: 1024`.** Our workload is
dominated by scattered single-row reads, which is exactly the case small pages
serve.

### 2. Prefetch changes request count, not bytes

"Three separate virtual read heads" detect sequential access and grow request
sizes exponentially, so "index scans or table scans reading more than a few KiB
of data will only cause a number of requests that is logarithmic in the total
byte length."

Consequence for our analysis: a full table scan still transfers ~127 MB (bytes
are bytes), but in tens of requests rather than tens of thousands. Our byte
budget is the right guardrail; a request-count budget would not be.

### 3. Index design — we already comply

- Covering indexes: put every column the query needs *in* the index, else
  SQLite does "another random access (unpredictable) read and thus HTTP request
  to retrieve the actual value for every data point". This is exactly why
  `name_word` carries `ho_ten_ascii`.
- Column order decides which lookups are cheap.
- Verify with `EXPLAIN QUERY PLAN`; a `SCAN` means the whole table crosses the
  network. We did this for every query the app issues.

### 4. Chunked mode exists for CDN caching

`serverMode: "chunked"` splits the database into parts (10 MB is the commonly
cited size) with a JSON config. Stated benefit: "CDN caching much more
effective" for large databases.

Relevant because **GitHub Pages sets `Cache-Control: max-age=600`** — ten
minutes — on everything. Range responses are cached by Fastly per object; with
one 289 MB object the practical caching story is weaker than with 29 chunks
that a CDN edge can hold whole. Chunked mode also sidesteps any future per-file
concern.

Cost: a build step to split files + emit config, and every deploy invalidates
all chunks anyway (SQLite page layout is not deterministic across rebuilds).

### 5. Hosting facts confirmed

- Range requests work on Pages "out of the box" — matches our own probe (206 +
  correct `Content-Range`).
- CORS headers (`Access-Control-Allow-Origin`, `Access-Control-Allow-Headers:
  Range`) only matter cross-origin. Ours is same-origin — non-issue.
- 670 MiB database on Pages is the canonical demo. 289 MB is not exotic.
- `Content-Encoding` remains the one fatal case: the library discards
  `Content-Length` and throws when a HEAD carries a non-identity encoding.
  Unknown extensions like `.sqlite3` are served `application/octet-stream` and
  left alone.

### 6. Alternatives

| | sql.js-httpvfs | sqlite-wasm-http |
| --- | --- | --- |
| WASM base | own sql.js fork (~3.36 era) | official `@sqlite.org/sqlite-wasm` |
| Last release | 0.8.12, Sept 2022 | 1.2.0, Dec 2023; activity into Dec 2025 |
| Self-description | "demo-level code… not for high stability" | "experimental" |
| Concurrency | one worker | multiple connections, shared cache |
| Shared cache needs | — | `SharedArrayBuffer` → COOP/COEP headers |
| On GitHub Pages | works | works, but **Pages cannot set COOP/COEP**, so it falls back to the synchronous backend without shared cache |
| Module format | CJS+ESM | **ES6 only** |

Both are self-declared experimental. sqlite-wasm-http's advantage (maintained,
official WASM) is real; its headline feature (shared cache) is unavailable on
Pages precisely because Pages cannot send cross-origin isolation headers.

## Implementation recommendations

1. **Change page size to 1024 and `requestChunkSize` to 1024.** Parser sets
   `PRAGMA page_size=1024` before DDL; VACUUM already runs. Cost ~+5% file
   size; benefit ~4× less overfetch per row read.
2. **Keep `serverMode: "full"` for now.** Chunked mode's benefit is CDN cache
   efficiency, which Pages' 10-minute TTL blunts. Revisit if measured repeat-
   visit cost is bad.
3. **Keep sql.js-httpvfs.** Switching to sqlite-wasm-http buys a maintained
   dependency but loses nothing we use, and its differentiator does not work on
   Pages. Note it as the escape hatch.
4. **Verify `Content-Encoding` after deploy** — the single fatal hosting case.
5. Keep the byte budget; drop any idea of a request-count budget.

## Common pitfalls

- Unindexed query → whole table over the network. `EXPLAIN QUERY PLAN` is the
  check.
- Page size mismatched with `requestChunkSize` → every logical page read spans
  two requests.
- Serving the database compressed → library refuses to open it.
- Assuming CDN caching helps: on Pages, `max-age=600`.

## References

- [Hosting SQLite databases on GitHub Pages — phiresky](https://phiresky.github.io/blog/2021/hosting-sqlite-databases-on-github-pages/)
- [phiresky/sql.js-httpvfs](https://github.com/phiresky/sql.js-httpvfs)
- [mmomtchev/sqlite-wasm-http](https://github.com/mmomtchev/sqlite-wasm-http)
- [Query SQLite on GitHub Pages with sql.js-httpvfs (Mar 2026)](https://recca0120.github.io/en/2026/03/07/sql-js-httpvfs-static-hosting/)
- [sqlite3 WebAssembly documentation](https://sqlite.org/wasm)
- [GitHub Pages asset caching discussion](https://github.com/orgs/community/discussions/11884)

## Unresolved questions

1. Does 1024 measurably beat 4096 *for our queries*? Only a browser with the
   byte counter can answer; the estimate says yes for row fetches.
2. Does Fastly cache 206 responses for a 289 MB object well enough that chunked
   mode is unnecessary? Needs a deployed measurement.
3. Does the read-head prefetch overfetch on our index-range walks (fetching
   ahead beyond `LIMIT 100`)? Unknown without instrumentation.
