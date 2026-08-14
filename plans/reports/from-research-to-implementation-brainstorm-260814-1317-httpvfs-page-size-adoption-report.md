# Brainstorm: which httpvfs best practices to adopt

2026-08-14 13:17. Follows
[the research report](./web-sqlite-range-hosting-research-260814-1317-httpvfs-best-practices-report.md).
Branch `feat/httpvfs-range-queries`.

## Problem

Research surfaced three upstream practices we do not follow. Decide which are
worth the change, knowing nothing can be verified in a browser on this machine.

## Codebase context (scout)

| Concern | Touch points |
| --- | --- |
| Page size | `parser/internal/writer/writer.go:41-47` (PRAGMA must precede the DDL — page size is fixed once a table exists), `:185` (VACUUM already applies it), `web/src/lib/sqlite.svelte.ts:18,53` |
| Chunked mode | `databases.go:41` Extension, size guard, **`Clean()` deletes anything not `<id>.sqlite3` — would eat every chunk**, `site.go:135-138,153`, `datasets.ts:91-97`, `datasets.json`, ~15 tests |
| Library swap | `sqlite.svelte.ts:1-3,50-58,76,82`, `package.json:15`; all consumers go through `RemoteDatabase`, so the blast radius is one file |

## Options evaluated

### A. page_size 1024 + requestChunkSize 1024 — ADOPTED

- Upstream consensus: phiresky and mmomtchev both recommend 1024.
- Honest sizing for *our* pattern: row fetches 400 KB → 100 KB per search;
  the index walk is sequential so bytes are unchanged and only the request
  count rises, which prefetch read-heads collapse. Net ≈ 300 KB saved per
  search — bandwidth, not latency.
- Cost: ~10 lines; file size +5% (528 → ~555 MB total, still under the 1 GB
  Pages limit); both databases rebuilt.

### B. serverMode chunked — REJECTED

- Only benefit is CDN cache efficiency. GitHub Pages serves everything with
  `Cache-Control: max-age=600`, and each deploy relays out SQLite pages anyway,
  so cross-deploy caching is zero either way.
- Cost: split step, config JSON, `Clean()`/guards/`dbOf()`/`datasets.json`
  rework, ~15 tests.
- Complexity buying a benefit the host cancels. Revisit only behind a CDN with
  long TTLs.

### C. swap to sqlite-wasm-http — DEFERRED

- For: maintained (Dec 2025), official SQLite WASM instead of a 2022 fork;
  matches this repo's posture on stale dependencies. Swap is one file.
- Against: its differentiator (shared cache) needs COOP/COEP headers GitHub
  Pages cannot send, so we would get the synchronous fallback and the
  maintenance benefit only. And the current integration has never run in a
  browser — swapping now means two unverified variables and no way to tell
  which broke.
- Revisit after the current build is verified live.

## Decision

Adopt **A only**.

## Implementation notes

1. `PRAGMA page_size = 1024` in `writer.OpenDB`, between `sql.Open` and
   `db.Exec(schema.DDL)`. The existing VACUUM in `Finish` applies it.
2. `CHUNK_BYTES = 1024` in `web/src/lib/sqlite.svelte.ts`; it feeds
   `requestChunkSize` and must equal the page size.
3. Rebuild both databases, update `dbSizeMb` in `datasets.json` to the new
   sizes, re-run the assembler guards.

## Risks

- The benefit is arithmetic plus upstream authority, not measurement. The byte
  counter in the SQL tab is the check, once deployed.
- Prefetch deliberately overfetches ahead of the cursor, so the 25 MB search
  budget may trip earlier than a strict page count suggests. Tune after a real
  measurement, not before.

## Unresolved questions

1. Does 1024 actually beat 4096 for our queries in a browser?
2. Is Fastly's caching of ranges over a 300 MB object good enough that chunked
   mode stays unnecessary?
