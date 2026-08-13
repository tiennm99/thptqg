---
phase: 6
title: Differential parity gate
status: completed
priority: P1
dependencies:
  - 5
effort: ''
---

# Phase 6: Differential parity gate

## Overview

The decisive phase. Build all 4 datasets with **both** parsers and prove the databases are
equivalent. Nothing in Phases 1-5 is trusted until this passes — earlier tests use synthetic
fixtures and sampled real files; this is the only check against all 418 MB.

This is the entire safety argument for the migration.

## Requirements

- Functional: for each of the 4 datasets, Rust-built and Go-built DBs are **logically**
  equivalent, and both binaries' stdout matches.
- Non-functional: one reproducible command; exits non-zero on any mismatch.

## Do not use `verify-parity.js`

The original plan specified it. It does not work for this comparison, for three independent
reasons:

1. Its core check is "new columns must be all-NULL except an approved allowlist"
   (`verify-parity.js:89-103`). For Rust-vs-Go both DBs have the **identical 22 columns**, so
   `added` is always `[]` and that check is vacuous.
2. The guard at `:104-112` then iterates `APPROVED_RECOVERY` and pushes a failure for every
   column not in `added` — i.e. **7 guaranteed spurious failures** on a perfectly correct port
   (`2016.tieng_nga`, `2017.tieng_duc`, `2017.tieng_nhat`, and 4 more).
3. `APPROVED_RECOVERY` is a module-level `const` and argv is two positional paths (`:47`) —
   there is no flag. "Emptying it for this run" *is* editing the verifier, which this phase
   forbids, and would silently disable the historical check documented at
   `docs/data-pipeline.md:124-131`.

It also **passes silently** when a dataset is absent from both stats files: it iterates
`Object.keys(baseline)` (`:59`) with no expected-dataset set, so a dropped dataset is simply
never compared and the script prints `PARITY OK`.

Leave `verify-parity.js` and its allowlist untouched. They remain valid for the historical
schema-shape check they were built for.

## Architecture

Write one purpose-built comparator, `go-parser/scripts/differential-parity.mjs`, using
`node:sqlite` (already the repo's only SQLite client, via `db-stats.js:16`; no new dependency,
no `sqlite3` CLI needed — there isn't one on this box).

```
cargo build --release --manifest-path parser/Cargo.toml
go build -o go-parser/bin/xlsxread ./go-parser/cmd/xlsxread

for id in 2016 2017 2017-old 2017-old2:
    parser/target/release/xlsxread build --schema parser/configs/$id.yml \
        --input data/$id --output /tmp/rust-$id.db  > /tmp/rust-$id.stdout
    go-parser/bin/xlsxread          build --schema parser/configs/$id.yml \
        --input data/$id --output /tmp/go-$id.db    > /tmp/go-$id.stdout

node go-parser/scripts/differential-parity.mjs
```

The comparator asserts, per dataset:

1. **Both DBs exist** and the dataset set is exactly the 4 ids from `src/datasets.js` — fail
   loudly on a missing dataset rather than skipping it.
2. `SELECT COUNT(*)` identical.
3. Per-column non-NULL `COUNT(<col>)` identical for **all 22** columns.
4. **Full-table hash.** Stream `SELECT * FROM student ORDER BY so_bao_danh` from both DBs in
   lockstep, serialize each row deterministically, and feed a rolling SHA-256. On mismatch,
   report the first 20 differing `so_bao_danh` with their field-level diffs.
   - SQLite has **no `md5()`** (verified: `no such function: md5`; `sha3` likewise absent), so
     the hash must be computed in the host language, not in SQL.
   - `group_concat` is also unusable: pre-3.44 it has no in-aggregate `ORDER BY`, so ordering is
     undefined, and it would materialize a ~150 MB string.
   - Serialization must fix an explicit NULL sentinel and an explicit REAL formatting rule,
     otherwise the hash is not stable across drivers.
5. `PRAGMA table_info(student)` and `PRAGMA index_list(student)` identical.
6. **stdout identical**, modulo the `Size:` line. This is the **only** check that covers the
   `source_rows` / `skipped` / `insert errors` counters — they are computed from the reader's
   row stream and never reach the database, so every DB-level check above is blind to them.
   Concrete case: all 63 `data/2017` files carry a trailing empty sheet, and
   `docs/data-pipeline.md:114` records the result as `861,131 source / 63 empty / 861,068 DB`.
   A Go reader that skips zero-row sheets yields an identical database and identical hash while
   the counters silently become `861,068 / 0`.

Disk is not a constraint: the four raw DBs total ~708 MB per parser (~1.4 GB for both) against
38 GB free. Do **not** clean up between datasets — partial stats files are exactly how a
comparator silently skips a dataset.

## Precedent from Phase 1

The reader gate already proved this exact methodology end to end: a canonical serialisation of
both implementations' output, hashed and compared per unit, with a committed oracle and a
regeneration script. Reuse the shape — `go-parser/testdata/reader-fidelity-hashes.tsv` and
`go-parser/scripts/regen-fidelity-hashes.sh` are the working templates.

It also proved the failure mode this gate exists to catch. Four of the five divergences found
in Phase 1 were invisible to aggregate checks — identical row counts, identical column counts,
wrong values. Two of them (`6.0`→`6`, and CR stripped from 2,233 `ten_cum_thi` values) would
have reached the published database. Only cell-by-cell comparison surfaced them, which is why
the full-table hash below is non-negotiable rather than a nice-to-have.

## Related Code Files

- Create: `go-parser/scripts/differential-parity.mjs`
- Do not modify: `parser/scripts/verify-parity.js`, `parser/scripts/db-stats.js`

## Implementation Steps

1. Write the comparator with all 6 checks. It must exit non-zero on any mismatch.
2. Build both binaries; build all 8 databases and capture both stdout streams.
3. Run the comparator.
4. Investigate every discrepancy. Do not adjust the comparison to make it pass, and never
   modify Rust to match Go.
5. Record actual numbers and hashes in this file.

## Decision gate

Same binary discipline as Phase 1.

- **PASS**: all 4 datasets, zero row-count delta, zero per-column non-NULL delta, identical
  full-table hash, identical PRAGMA metadata, identical stdout.
- **FAIL** → escalate to the user with the diff. **Phase 7 does not start.** There is no
  "3 of 4 datasets" pass: shipping Go for three datasets and Rust for one means two toolchains
  in CI forever, which contradicts the entire point of Phase 7.
- **Abandon criterion**: if a divergence proves irreducible after a bounded effort, the outcome
  is *keep Rust and close the plan*. That is a legitimate result, not a failure — state it
  explicitly so the alternative (eroding the gate) never becomes the path of least resistance.

## If parity fails

Expected sources, in likelihood order:

1. **Date-cell stringification** (`ngay_sinh`) — calamine prints the raw serial, excelize
   applies the number format unless `RawCellValue: true`. Should have been caught in Phase 1.
   Note the plan's own out-of-scope rule forbids "accept a documented format change" as a
   resolution: the frontend is out of scope, so this must be fixed on the Go side.
2. **Numeric-cell rendering** in `so_bao_danh` — re-keys the table and cascades into row counts.
3. **Row width / trailing-blank trimming** — excelize trims; every column read is positional
   with `unwrap_or_default()`, so tail columns silently NULL. Shows as differing per-column
   non-NULL counts on `diem_thi`-derived scores (2017) or `tieng_anh` (2016 SeparateScores).
4. **`ToAscii` divergence** — differing `ho_ten_ascii` while `ho_ten` matches. Almost certainly
   the `unicode.Mn` vs literal-range trap.
5. **Score NULL/0.0 handling** in 2016 — differing non-NULL counts on score columns.
6. **Duplicate-SBD ordering.** `INSERT OR REPLACE` is last-wins, so the surviving row depends on
   iteration order. The relevant invariants are (a) the **sorted file list** — Rust collects
   `read_dir` then calls `files.sort()` at `main.rs:97` and `:234`, so raw `read_dir` order is
   never used and "match `fs::read_dir` order" is the wrong target — and (b) **sheet
   enumeration order within a file**, since `sheet_mode = "all"` for 2016 and 2017 and
   overflow sheets can repeat an SBD. 2016 has 3 documented collapsed duplicates
   (`docs/data-pipeline.md:112`), so this changes 3 students' field values while leaving every
   count identical — visible only to the full-table hash.

## Success Criteria

- [x] All 8 databases build without error
- [x] Comparator asserts the dataset set is exactly the 4 expected ids
- [x] Row counts identical for all 4 datasets
- [x] Per-column non-NULL counts identical across all 22 columns × 4 datasets
- [x] Full-table SHA-256 identical for all 4 datasets
- [x] stdout identical (modulo `Size:`) for all 4 datasets
- [x] Schema and index metadata identical
- [x] Differential run is a single reproducible command exiting non-zero on mismatch
- [x] Actual numbers and hashes recorded in this file
- [x] Explicit PASS/FAIL recorded
- [x] Go build wall-time recorded vs Rust (informational)

## Risk Assessment

| Risk | Mitigation |
|---|---|
| Counter divergence invisible to DB checks | stdout comparison is a first-class criterion |
| Comparator silently skips a dataset | Expected-dataset-set assertion |
| Aggregate checks miss row-level corruption | Full-table hash over every row, every column |
| "Strongest check" unimplementable | Host-language hash, no SQL `md5()` dependency |
| Gate eroded under pressure | Binary gate + explicit abandon criterion |
| VACUUM temp space during builds | ~234 MB transient per largest DB; 38 GB free |

## RESULT — 2026-08-13: **PASS — all 4 datasets**

```
--- 2016 ---       rows 877461  sha256 f2655b88be00d6f5  schema OK  stdout identical
--- 2017 ---       rows 861068  sha256 b71bc4178d65003e  schema OK  stdout identical
--- 2017-old ---   rows 847348  sha256 8e7088e346b957bd  schema OK  stdout identical
--- 2017-old2 ---  rows 679764  sha256 260b42af5bee15b1  schema OK  stdout identical
PARITY OK
```

3,265,641 rows compared field-by-field. Per-column non-NULL counts identical across all 22
columns × 4 datasets. `PRAGMA table_info` and `index_list` identical. stdout identical.
Runtime ~79s for the whole gate.

### The gate is proven able to fail

A gate that cannot fail proves nothing, so it was tested negatively: perturbing **one** `toan`
value (3 → 3.25) in a copy of `2017-old2` — one cell out of 14.9M — makes it exit non-zero.

Notably, on that corrupted database the row count and all 22 per-column non-NULL counts still
reported **OK**. Only the full-table hash caught it. That is precisely the failure mode the
hash exists for, and why aggregate checks alone would not have been sufficient.

### Deviations from the phase as planned

- `verify-parity.js` was not used, as specified. The comparator
  `go-parser/scripts/differential-parity.mjs` is purpose-built on `node:sqlite` — no new
  dependency and no `sqlite3` CLI, which does not exist on this machine.
- The hash is computed in the host language. SQLite has no `md5()`, so the originally-planned
  `SELECT md5(group_concat(...))` was never implementable.
- Serialisation fixes an explicit NULL sentinel and a fixed REAL rendering, so the hash is
  stable across two different embedded SQLite versions (Rust ~3.46 vs modernc 3.53.3). Byte
  equality was never the goal and is precluded by the file format.
- The comparator fails loudly on a missing dataset rather than skipping it — the silent-skip
  bug that made `verify-parity.js` unsafe.
