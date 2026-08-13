---
phase: 1
title: "Standard schema and unified parser"
status: pending
priority: P1
dependencies: []
effort: ""
---

# Phase 1: Standard schema and unified parser

## Overview

Move the SQL schema out of the four TOML configs into one Rust module, widen it
to the 22-column union, and prove the four rebuilt databases still match their
current contents. Done in place under the existing `2016/`/`2017/` layout so
parity is measurable before anything moves.

## Context

`2016/tools/xlsxread` and `2017/tools/xlsxread` are the same crate. Verified by
`diff -rq`: `cli.rs`, `error.rs`, `reader.rs` are byte-identical; `audit.rs`,
`config.rs`, `lib.rs`, `main.rs`, `transform.rs`, `writer.rs` differ only where
2016 adds a branch; `format_detect_2016.rs` (549 lines) exists only in 2016.
The 2016 crate is therefore the base to keep — it already builds 2017 datasets
(its `configs/` holds 2017 fixtures used by `tests/golden.rs`).

Current per-dataset schema divergence:

| Column group | 2016 | 2017 / old / old2 |
|---|---|---|
| `ten_cum_thi`, `gioi_tinh` | present | absent |
| `khtn`, `khxh`, `gdcd`, `tieng_nga` | absent | present |
| `tieng_duc`, `tieng_nhat` | present | absent |

## Requirements

**Functional**
- One DDL, one INSERT, one ordered subject list, one regex map — used by all four datasets
- Configs keep only genuinely per-dataset settings
- Rebuilt DBs preserve every value currently produced

**Non-functional**
- No measurable build-time regression (currently a few minutes per dataset)
- DB size growth from the added NULL columns stays under ~2%

## Architecture

**New `parser/src/schema.rs`** (written into `2016/tools/xlsxread/src/` this
phase; relocated in Phase 2) exports:

- `pub const DDL: &str` — the 22-column CREATE TABLE + 3 indexes
- `pub const INSERT_SQL: &str` — 22 positional placeholders
- `pub const IDENTITY_FIELDS: &[&str]` — 6 identity columns in INSERT order
- `pub const SCORE_FIELDS: &[&str]` — 16 subject columns in INSERT order
- `pub fn score_patterns() -> Vec<(&'static str, &'static str)>` — the union of
  all 16 subject regexes (2016's 12 ∪ 2017's 14; both sets share 10)

**`writer.rs` collapses.** `SCORE_FIELDS_2016`, `SCORE_FIELDS_2017`, the
`SCORE_FIELDS` alias, and `insert_row_2016` all disappear. A single `insert_row`
binds identity fields then subject fields, always 22 params.

**`config.rs` shrinks.** `[schema]` and `[insert]` sections are removed from
`DatasetConfig`; `[scores]` becomes optional and unused (delete the field once
no config sets it). `columns` stays `Option<Columns>` — the 2016 detection path
does not use it.

**`main.rs`** keeps the `run_build_2016` / `run_build_standard` split; both now
call the same `insert_row` with `schema::SCORE_FIELDS`.

### Union regex risk — RESOLVED, and it found a real bug

Applying all 16 patterns to every dataset means a source cell containing a
subject the old config never looked for now populates a column it previously
could not.

**Outcome: the gate fired, and the matches were real data, not false positives.**

The pre-refactor configs listed 12 subject regexes (2016) and 14 (2017). Neither
list was complete — Vietnamese candidates could sit German, Japanese and Russian
in *both* exam years. Every affected student previously ended up with **no**
foreign-language score at all. Unifying to 16 patterns recovered 1,691 scores:

| Dataset | Recovered |
|---|---|
| 2016 | 182 × `tieng_nga` |
| 2017 | 93 × `tieng_duc`, 512 × `tieng_nhat` |
| 2017-old | 85 × `tieng_duc`, 484 × `tieng_nhat` |
| 2017-old2 | 22 × `tieng_duc`, 313 × `tieng_nhat` |

Verified genuine, not spurious:
- Across all four datasets every student holds **zero or exactly one** foreign
  language — never two. So the new columns duplicate nothing.
- Each affected student had *all* language columns NULL beforehand
  (e.g. SBD `01003198`: all NULL → `tieng_duc = 8`).
- Counts track dataset size consistently across the three 2017 generations
  (93/85/22 German, 512/484/313 Japanese).
- Score ranges and distributions are ordinary exam values (0–10).

Row counts, all 18 pre-existing per-column non-NULL counts, and the
deterministic student samples were **identical** — nothing was lost.

Accepted as a data-quality fix. The earlier guidance to add a per-config subject
allowlist assumed spurious matches and does **not** apply; suppressing these
would knowingly discard real scores. The exact counts are now encoded as
`APPROVED_RECOVERY` in `verify-parity.js`, so any *other* newly-populated column
— or any drift in these numbers — still fails the gate.

## Related Code Files

- Create: `2016/tools/xlsxread/src/schema.rs`
- Modify: `2016/tools/xlsxread/src/writer.rs` (drop dual insert paths + field lists)
- Modify: `2016/tools/xlsxread/src/config.rs` (drop `[schema]`/`[insert]`/`[scores]`)
- Modify: `2016/tools/xlsxread/src/main.rs` (single insert call site per path)
- Modify: `2016/tools/xlsxread/src/lib.rs` (declare `schema` module)
- Modify: `2016/tools/xlsxread/src/transform.rs` (source patterns from `schema`)
- Modify: `2016/tools/xlsxread/tests/golden.rs` (assert against canonical schema)
- Create: `2016/tools/xlsxread/configs/thptqg2017-data-old2.toml` (copy from 2017 crate)
- Modify: all 4 configs — strip DDL/INSERT/scores down to parse rules
- Delete (Phase 2): `2017/tools/`

## Implementation Steps

1. **Capture the baseline first.** Build all four DBs with the *current* code and
   record, per dataset: `SELECT COUNT(*) FROM student`, and for each column
   `SUM(col IS NOT NULL)`. Store as `plans/reports/parser-parity-baseline.json`.
   Nothing else in this plan is verifiable without this artifact.
2. Copy `thptqg2017-data-old2.toml` into the 2016 crate's `configs/`.
3. Write `schema.rs` with DDL, INSERT, field orders, and the union regex map.
4. Rewrite `writer.rs` to a single `insert_row`; delete `insert_row_2016` and
   the three score-field constants.
5. Strip `[schema]`, `[insert]`, `[scores]` from all four configs; update
   `config.rs` structs to match. Each config should end at ~15–20 lines.
6. Update `main.rs` and `transform.rs` call sites.
7. Update `tests/golden.rs` to assert the canonical column set.
8. `cargo test` — golden tests must pass.
9. Rebuild all four DBs; regenerate the same stats and diff against the baseline.

## Tests / Validation

- `cargo test` — 63 tests (55 unit + 8 golden)
- `cargo clippy --all-targets -- -D warnings`
- Row count per dataset equals baseline exactly
- Per-column non-NULL count equals baseline for every previously-existing column
- Newly-added columns read 0 non-NULL, except the approved recoveries above
- DB file size within 2% of baseline

**Note on the clippy gate:** it was already red before this phase — measured at
**8 errors on the branch base**. This phase's changes introduced none. The
pre-existing lints were cleared in a separate commit so the gate is genuinely
green from here on.

## Success Criteria

- [x] `schema.rs` is the only place DDL/INSERT/subject-order/regexes are written
- [x] All 4 configs under 35 lines, containing no SQL
- [x] `writer.rs` has exactly one insert function
- [x] `cargo test` (63 passing) and `cargo clippy --all-targets -D warnings` green
- [x] All 4 DBs match baseline row counts and per-column non-NULL counts
- [x] Baseline JSON committed under `plans/reports/`
- [x] Config parsing rejects leftover SQL sections (`deny_unknown_fields`)

## Risk Assessment

| Risk | Mitigation |
|---|---|
| Union regexes populate unexpected columns | Baseline diff catches it; fall back to per-config subject allowlist |
| INSERT param order drifts from DDL order | Single `IDENTITY_FIELDS`/`SCORE_FIELDS` source drives both; golden test asserts round-trip |
| Rebuilding 419 MB of source is slow | Run once per verification pass, not per edit; iterate against golden fixtures |
| Baseline skipped under time pressure | Phase 1 is unverifiable without it — treat step 1 as blocking |
