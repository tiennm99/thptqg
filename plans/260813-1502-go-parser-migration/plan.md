---
title: Migrate parser from Rust to Go as side-by-side go-parser/
description: >-
  Build go-parser/ alongside the Rust parser/, validated by differential
  comparison against live Rust output. Rust stays working until parity is signed
  off.
status: completed
priority: P2
branch: main
tags:
  - migration
  - go
  - parser
  - data-integrity
  - tdd
blockedBy: []
blocks: []
created: '2026-08-13T08:21:47.330Z'
createdBy: 'ck:plan'
source: skill
---

# Migrate parser from Rust to Go as side-by-side go-parser/

## Overview

Port the 2.3k-line Rust `parser/` crate to Go under a new `go-parser/` directory. Rust is
untouched and keeps building throughout, so ground truth is regenerable on demand and the
migration is reversible until Phase 7.

**Driver: preference for working in Go.** No defect exists in the Rust parser. Recorded
honestly rather than retrofitted with technical justification — this shapes the plan, because
with no problem to fix, the only measure of success is *behavioral identity with Rust*.

Mode: `--tdd`. Tests come first in every phase. The Rust crate is an executable specification;
**29 of its 63 tests transfer directly** (the `&str`-based `transform` tests). The other 34 are
`calamine::Data`-typed or golden tests; Phase 1 recorded the exact string each `Data` variant
renders to, so they can now be re-derived without guessing.

## Design decisions

| Decision | Choice | Rationale |
|---|---|---|
| Layout | New `go-parser/`, Rust `parser/` untouched | Reversible by construction; both runnable for diffing |
| Validation | Differential vs **live Rust output** | Rust still runs, so no frozen baseline needed |
| Comparator | **Purpose-built**, not `verify-parity.js` | That script's baseline-diff semantics are wrong for a same-schema comparison — it emits 7 spurious failures. See Phase 6 |
| Binary path | `go-parser/bin/xlsxread` | Two parsers writing one path invites confusion. Costs a one-line change at `build-db.js:25` |
| Reader contract | **One** streaming API with a typed `Cell`, defined in Phase 1 | Mirrors Rust's `process_file`; a `[][]string` collapse loses `Data::Empty` and row width |
| SQLite driver | Decided in **Phase 2**, with written rationale | Governs cgo/CI/ARM64 shape and the integrity story; not deferrable to Phase 4 |
| BIFF reader | **`pbnjay/grate`** (Phase 1) | `extrame/xls` corrupted 69% of cells; grate matched calamine on all 67 files |
| `.xls → .xlsx` conversion | **Not needed** (Phase 1, 2026-08-13) | grate reads BIFF exactly, so source data stays untouched. Fallback retired, not exercised |
| Config format | **YAML (.yml)**, read by BOTH parsers (Phase 2) | User preference. Converting only Go would leave two hand-synced copies whose drift Phase 6 would blame on the parser. Amends "parser/ untouched" deliberately |
| JS scripts | **Not ported** | All four candidates have zero automated consumers; two are documented broken |
| Rust removal | After parity sign-off only, behind a tag | Phase 7e |

## Phases

| Phase | Name | Status |
|-------|------|--------|
| 1 | [Scaffold and reader fidelity gate](./phase-01-scaffold-and-biff-reader-gate.md) | Completed |
| 2 | [Schema and config](./phase-02-schema-and-config.md) | Completed |
| 3 | [Transform core](./phase-03-transform-core.md) | Completed |
| 4 | [Reader writer and CLI](./phase-04-reader-writer-and-cli.md) | Completed |
| 5 | [2016 format detection](./phase-05-2016-format-detection.md) | Completed |
| 6 | [Differential parity gate](./phase-06-differential-parity-gate.md) | Completed |
| 7 | [CI docs and cutover](./phase-07-ci-docs-and-script-port.md) | Completed |

Strictly sequential: 1 → 2 → 3 → 4 → 5 → 6 → 7. Phases 1 and 6 are hard gates.

## The dominant risk — RESOLVED in Phase 1 (2026-08-13)

Reader fidelity was the plan's dominant risk. It is now **settled: 299/299 files byte-identical
to calamine**, locked in as a Go test against a committed hash oracle. Full record in
`phase-01-scaffold-and-biff-reader-gate.md`.

- **`extrame/xls` was unusable** — 69% of cells corrupted, 28% lost, charset-independent.
  Replaced with **`pbnjay/grate`**, which matched calamine on all 67 BIFF files. The red-team
  claim that `extrame/xls` read the corpus correctly was wrong; it rested on "opens without
  panic" plus spot-checks, and spot-checks pass because 28% of cells are right.
- **The `.xlsx` date-serial fear was unfounded.** Scanning all 299 files (15.98M cells) found
  **zero `DateTime` cells** — also zero `Int`, `Bool`, `Error`, `DateTimeIso`, `DurationIso`.
  Only `String`, `Empty`, and `Float` occur. `ngay_sinh` is text everywhere.
- **The used-range-origin fear was unfounded.** Every used range in the corpus starts at (0,0).
- **Two real divergences did reach the database** and are fixed: numeric re-rendering
  (`6.0`→`6`, gated on cell type so shared strings like `6.00`, `NAN`, and leading-zero
  `so_bao_danh` are untouched), and XML line-ending normalisation stripping CR from 2,233
  `ten_cum_thi` values.

Consequence for `--tdd`, now unblocked: the 11 `format_detect_2016` and 7 `reader` tests build
fixtures from `calamine::Data` values, and Phase 1 recorded the exact rendering of each variant,
so they can be ported without guessing.

**The `.xls → .xlsx` conversion fallback is no longer needed** and remains unexercised. Source
data is untouched.

## Acceptance criteria

- [x] For all 4 datasets, Go-built and Rust-built DBs are **logically equivalent**: identical
      row counts, identical per-column non-NULL counts across all 22 columns, identical
      `PRAGMA table_info`/`index_list`, and identical sorted full-table SHA-256
- [x] Both binaries emit identical build stdout per dataset, modulo the `Size:` line
      (this is the only check that covers the `source_rows`/`skipped` counters)
- [x] Go tests pass, including reader-fidelity tests against real files
- [x] `npm run build:db` produces four `.db.gz` via the Go binary, with a **row-count guard**
      that fails the build on deviation
- [x] CI green with Go toolchain, Rust actions removed, and a branch-verify path that does not
      publish to production
- [x] Frontend loads all 4 datasets unchanged, including accent-insensitive search

**Explicitly not a criterion:** byte-identical databases. SQLite writes its own version number
into header bytes 96-99, `VACUUM` rewrites page layout per-version, and `gzip -9` without `-n`
stores mtime. Byte equality is precluded by the file format, not merely difficult.

## Out of scope

- Frontend behavior, `src/` logic, `scripts/assemble-site.js`, Vite config
  (**exception**: stale path comments in `src/datasets.js` and `src/lib/subjects.js` must be
  updated in Phase 7 — they reference `parser/` paths that will not exist)
- Schema changes — the 22-column contract is frozen
- Porting the JS helper scripts (see Design decisions)
- Behavior "improvements". Bug-for-bug compatibility is the goal for **everything that reaches
  the database**. For stdout, replicate the per-file row-count line; see Phase 4 on the
  `dataset_label` caveat

## Dependencies

Builds on completed plan `260813-0956-unify-frontend-standard-schema`. No blocking relationship.

Inputs:
- Brainstorm: `plans/reports/from-brainstorm-to-plan-260813-1502-go-parser-side-by-side-migration-report.md`
- Scout: `plans/reports/from-scout-to-brainstorm-260813-1502-rust-to-go-parser-migration-report.md`

## Open questions

1. Keep `parser/` as a reference implementation after parity, or delete it? (Phase 7e assumes
   delete, behind a `pre-go-parser-removal` tag.)
2. `modernc.org/sqlite` is a machine-transpiled SQLite, not the upstream C amalgamation that
   `rusqlite --bundled` vendors. Acceptable for the writer of a published 1.5M-row dataset, or
   use `mattn/go-sqlite3` (real upstream C, cgo cost in CI)? Decided in Phase 2.

## Red Team Review

### Session — 2026-08-13
**Findings:** 39 raw across 4 reviewers → 22 unique (19 accepted, 3 rejected)
**Severity breakdown:** 6 Critical, 10 High, 6 Medium

| # | Finding | Severity | Disposition | Applied To |
|---|---------|----------|-------------|------------|
| 1 | `.xlsx` stringification divergence certain and ungated; BIFF framing wrong | Critical | Accept | Completed |
| 2 | `verify-parity.js` unusable — 7 spurious failures, contradictory instructions | Critical | Accept | Completed |
| 3 | `md5()` does not exist in SQLite; "strongest check" fictional | Critical | Accept | Completed |
| 4 | No deploy guard — empty/truncated DB ships with green CI | Critical | Accept | Completed |
| 5 | "Verify on a branch" unexecutable; `workflow_dispatch` publishes to prod | Critical | Accept | Completed |
| 6 | Counter divergence invisible; 63 trailing empty sheets in `data/2017` | Critical | Accept | Completed |
| 7 | Phase 7e breaks `npm run build:db`; 33 refs vs 8 listed | High | Accept | Completed |
| 8 | "byte-equivalent databases" provably unachievable | High | Accept | plan.md |
| 9 | Phase 3 cites 20 of 29 tests; omitted 9 are the flagged traps | High | Accept | Phase 3 |
| 10 | Two incompatible reader contracts; `[][]string` lossy | High | Accept | Phase 1, 4 |
| 11 | Phase 7d ports 4 scripts with 0 consumers, 2 broken | High | Accept | Phase 7 (cut) |
| 12 | `.xls` golden fixture unbuildable — no Go BIFF writer | High | Accept | plan.md, Phase 4 |
| 13 | Golden fixture port is phantom coverage (`inlineStr` only) | High | Accept | Phase 4 (cut) |
| 14 | No partial-success/abandon procedure at Phase 6 | High | Accept | Phase 6 |
| 15 | `verify-parity.js` silently passes on datasets absent from both files | High | Accept | Phase 6 |
| 16 | PII: committing real rows as testdata breaks documented convention | Medium | Accept | Phase 1 |
| 17 | Duplicate-SBD guidance names `read_dir`; Rust sorts explicitly | Medium | Accept | Phase 6 |
| 18 | `dataset_label` derived from path breaks tempdir stdout comparison | Medium | Accept | Phase 4 |
| 19 | No dependency trust/pinning/`govulncheck` step | Medium | Accept | Phase 1, 2 |
| 20 | Make `.xls`→`.xlsx` conversion unconditional Phase 0 | High | **Reject** | — |
| 21 | Publish DBs as artifacts; drop parser from critical path | High | **Reject** | — |
| 22 | Merge Phases 2-5 into one "port the crate" phase | Medium | **Reject** | — |

**Rejection rationale:**
- **20, 21** — user decisions, not reviewer calls. 20 mutates committed source data and the user
  explicitly deferred it on 2026-08-13. **Superseded by Phase 1**: `pbnjay/grate` reads BIFF
  exactly, so no conversion is needed and source data stays untouched. (The `extrame/xls`
  evidence cited when this was first rejected was itself wrong — but the conclusion holds for
  a better reason.) 21 reverses the user's stated goal of working in Go on the parser.
- **22** — phases map to TDD checkpoints and hydrated tasks. Merging reduces granularity without
  reducing risk. Phase 1's gate framing was re-pointed instead.

**Citation corrections applied:** `transform.rs:161`→`:162`; `deployment-guide.md:37`→`:38`;
"rusqlite statement cache" removed (`writer.rs:73` uses `conn.execute`, not `prepare_cached`);
"doc comment says category M"→ the *inline* comment at `:53` (the doc comment at `:49` is
correct); test counts corrected to 63 total / 29 in `transform.rs`.

### Whole-Plan Consistency Sweep
- Files reread: `plan.md`, `phase-01` … `phase-07` (all 8)
- Decision deltas checked: 19
- Reconciled stale references: dominant-risk framing (plan.md + Phase 1), byte-equivalence
  criterion (plan.md), reader contract (Phases 1 + 4), `verify-parity.js` usage (Phase 6),
  script-port scope (plan.md + Phase 7), test counts (plan.md + Phases 2/3/5), phase title
  "BIFF reader gate" → "reader fidelity gate" (plan.md table + Phase 1)
- Unresolved contradictions: 0
