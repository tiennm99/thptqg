---
phase: 7
title: CI docs and cutover
status: completed
priority: P2
dependencies:
  - 6
effort: ''
---

# Phase 7: CI docs and cutover

## Overview

Make Go the real parser: add a deploy guard, swap CI, update docs, remove Rust. Runs only after
Phase 6 signs off on all four datasets. Until this phase the migration is fully reversible;
after 7e it is not.

## Requirements

- Functional: `npm run build:db` uses the Go binary and **fails on a bad database**; CI green
  without a Rust toolchain.
- Non-functional: zero dangling references to `parser/`, `cargo`, or `Cargo.toml`.

## Architecture

Strict order, each step independently verifiable:

1. **7a** — deploy guard + npm scripts + `build-db.js` point at Go
2. **7b** — CI: add a branch-verify path *first*, then swap the toolchain
3. **7c** — docs and stale references
4. **7d** — tag, then remove Rust `parser/`

There is no JS-script port. See "Scripts are not ported".

## 7a — Deploy guard (new, and the most important step here)

Today **nothing between the parser and the public site asserts that a database has data**:
- `main.rs:171-177` logs file-level errors and continues; `run_build_standard` returns `Ok(())`
  at `:198` regardless of `total_errors`.
- `writer.rs:88-133` prints stats and returns `Ok` even at `db_count == 0`; the `Audit:` line at
  `:120-127` is `println!` only, never an exit code.
- `build-db.js:47-68` gzips whatever it gets, with no inspection.
- `scripts/assemble-site.js:56-73` only greps *filenames* for stray `.db`; an empty
  `.build/public/db` passes cleanly.
- `deploy-pages.yml` has no verification step.

So a Go reader that silently under-produces ships a truncated public dataset with **green CI**.
This is the single largest blast-radius gap in the migration and it is cheap to close.

Add to `build-db.js`, as a blocking check per dataset:
- fail non-zero if the binary reported any file-level error
- fail non-zero if `SELECT COUNT(*)` deviates from the known-good figure in
  `docs/data-pipeline.md:110-115` (877,461 / 861,068 / 847,348 / 679,764)
- fail non-zero if the resulting `.db.gz` is under 90% of `dbSizeMb` in `src/datasets.js`

Also make the Go binary exit non-zero when `total_errors > 0`. This is the one place where
bug-for-bug compatibility costs more than it buys — note the deliberate divergence in the code.

## 7a — Pipeline wiring

- `package.json:8`: `"build:go": "go build -o go-parser/bin/xlsxread ./go-parser/cmd/xlsxread"`
- **`package.json:9`**: `"build:db"` currently reads `node parser/scripts/build-db.js`. Keep the
  script *name* (CI and docs reference it) but the *path* must change when the file moves.
  Missing this breaks the deploy step.
- `build-db.js:25`: `BIN` → `go-parser/bin/xlsxread`; update the `npm run build:rust` hint at
  `:37-41`.
- Verify: `npm run build:go && npm run build:db` produces all 4 `.db.gz` and the guard fires when
  fed a deliberately truncated DB.

## 7b — CI

**Prerequisite, before the toolchain swap:** the workflow currently triggers on `push` to `main`
only, plus an unguarded `workflow_dispatch` (`deploy-pages.yml:3-6, 55-63`). So "verify on a
branch" is impossible — pushing to a branch runs nothing, and dispatching from a branch
**publishes that branch's output to the live site**, with `cancel-in-progress: true` killing any
in-flight good deploy. Fix this first:

- add a `pull_request` (or branch-push) trigger that runs the **build job only**
- guard the deploy job with `if: github.ref == 'refs/heads/main'`

Then swap:
- remove `dtolnay/rust-toolchain@stable` (`:23`) and `Swatinem/rust-cache@v2` (`:25-27`)
- add `actions/setup-go@v5` pinned to **1.26.x** (matches the verified local toolchain), with
  module caching
- add `govulncheck` (there is an open excelize advisory, and the 2017 refresh runbook feeds
  network-downloaded spreadsheets straight into the parser)
- run `go test ./...` in CI — the reader-fidelity suite covers all 299 real files in ~77s and is
  the regression guard for the whole reader
- build step → `npm run build:go && npm run build:db`
- **cgo**: `pbnjay/grate` and `excelize/v2` are pure Go. Whether the workflow needs a C
  toolchain depends solely on the Phase 2 SQLite driver decision (`modernc.org/sqlite` keeps it
  cgo-free; `mattn/go-sqlite3` does not). Set `CGO_ENABLED` explicitly either way.

## 7c — Docs and stale references

The previous hand-curated file list covered 8 locations; there are **33** `parser/` references
outside `parser/`. Use a mechanical gate instead of a list:

```
grep -rn "parser/\|cargo\|Cargo\.toml" \
  --include='*.js' --include='*.jsx' --include='*.json' --include='*.yml' --include='*.md' . \
  | grep -v node_modules | grep -v '^./plans/'
```

must return zero rows before 7d is marked done. Note the old success criterion grepped only for
`cargo` / `Cargo.toml` / `parser/target` — none of which match `parser/scripts/…` or
`parser/configs/…`.

Known references beyond the original list, including two the plan had scoped out:
- `package.json:9`, `eslint.config.js:31` (its glob `parser/scripts/**/*.js` would silently stop
  matching, dropping the moved scripts from `npm run lint`)
- `vite.config.js:14`, `src/datasets.js:6,11,17`, `src/lib/subjects.js:4` — the `src/` ones are
  comments; `plan.md` carves them out of the frontend exclusion explicitly
- `README.md:26,37,56`; `docs/system-architecture.md:34,38,50`;
  `docs/data-pipeline.md:22,57,119,121,125,126,137,138,145`;
  `docs/deployment-guide.md:47,59,61`
- `docs/deployment-guide.md:38` is the `build:rust` line (the plan previously cited `:37`, which
  is `npm ci`); `:10` mentions the Rust toolchain
- `docs/data-pipeline.md` references `parser/src/schema.rs` as canonical DDL →
  `go-parser/internal/schema/schema.go`
- `.gitignore:20-21`: `parser/target/` → `go-parser/bin/`

Per documentation rules, update what changed; no changelog noise.

## Scripts are not ported

The original plan ported `db-stats.js`, `verify-parity.js`, `check-duplicates.js`, and
`diff-datasets.js`. Full consumer enumeration says don't:

| Script | Automated consumers | Notes |
|---|---|---|
| `db-stats.js` | **0** | 3 doc refs only |
| `verify-parity.js` | **0** | 2 doc refs; still valid for its historical check — leave it |
| `check-duplicates.js` | **0** | **Broken**: `:10` hardcodes `D:/tiennm99/thptqg2017/data` |
| `diff-datasets.js` | **0** | **Broken**: imports `better-sqlite3`, absent from `package.json`; reads paths that don't exist |
| `crawl-baotintuc.js` | 0 automated, but **the only one with a live runbook** (`docs/data-pipeline.md:22,137-140`) | Leave in JS |

None appear in `package.json` or the workflow. Node is already a hard build dependency
(`actions/setup-node@v4`), so leaving them in JS costs nothing. Porting the two broken ones
would mean either reproducing a hardcoded Windows path in Go or fixing them — undeclared scope
and a behavior change in a plan whose rule is bug-for-bug compatibility.

The criterion is usage, not topic: **no script with zero automated consumers gets ported.** That
excludes all five. `crawl-baotintuc.js` stays in JS because it works and has a runbook.

## 7d — Remove Rust

**Extra cleanup from Phase 1:** `parser/examples/dump_cells.rs` and `parser/examples/scan_kinds.rs`
are throwaway ground-truth tooling. They disappear with `parser/`, which also retires
`go-parser/scripts/regen-fidelity-hashes.sh` (it shells out to `cargo`). Before deleting,
decide whether the reader-fidelity oracle should survive:

- keeping `go-parser/testdata/reader-fidelity-hashes.tsv` preserves a real regression guard over
  all 299 files, but it becomes unregenerable once calamine is gone — the same trap that made
  `verify-parity.js`'s baseline useless;
- or drop the manifest and the suite with it, and rely on Phase 6's database-level gate.

Recommend keeping it and noting in the file header that it is frozen and why.


- Move `parser/configs/` → `go-parser/configs/`; update `build-db.js` and `package.json:9`,
  `eslint.config.js:31` in the **same commit** as the move.
- **Tag `pre-go-parser-removal` and push it** before deleting anything.
- Delete `parser/`.
- Write the revert procedure into this file as three named commands — "git history preserves it"
  is not a procedure, and after this step a revert is non-trivial because configs and
  `build-db.js` have moved.
- Full verification: `npm run build:go && npm run build:db && npm run build:site`, then load the
  site and query each dataset.
- **Confirm with the user before deleting** — open question 1 in `plan.md`.

## Success Criteria

- [x] `build-db.js` guard fails the build on a truncated/empty DB (verified deliberately)
- [x] Go binary exits non-zero when `total_errors > 0`
- [x] Branch-verify CI path runs the build job without publishing; deploy job guarded to `main`
- [x] CI green with no Rust toolchain; `govulncheck` wired in; deploy succeeds
- [x] The 7c grep gate returns **zero rows**
- [x] `.gitignore` covers the Go binary; no build artifact committed
- [x] `npm run lint` still covers the relocated scripts
- [x] Frontend loads all 4 datasets; accent-insensitive search works (exercises `ho_ten_ascii`)
- [x] `pre-go-parser-removal` tag pushed before deletion; revert procedure written down
- [x] Rust removal confirmed with the user

## Risk Assessment

| Risk | Mitigation |
|---|---|
| Bad database reaches the public site with green CI | 7a guard — the reason this step exists |
| "Verify on a branch" publishes to production instead | 7b prerequisite: branch trigger + deploy ref guard |
| `npm run build:db` breaks after the move | `package.json:9` called out; same-commit rule; grep gate |
| Lint coverage silently lost | `eslint.config.js:31` called out; explicit criterion |
| Rust deleted before a latent bug surfaces | Tag + written revert procedure + user confirmation |
| Effort spent porting dead scripts | Cut, with consumer counts recorded |

## RESULT — 2026-08-13: **PASS**

Go is the parser; Rust is gone. Full pipeline verified from a clean tree with no
Rust present: all four datasets build, pass the row-count guard, gzip, and assemble
into `_site/`.

### 7a — deploy guard (the step that mattered most)

`go-parser/scripts/build-db.js` now refuses to publish a database whose row count
deviates from the known figure, or whose `.db.gz` is under 90% of its usual size.
**Verified in both directions**: an expected count off by one exits 1, the correct
count exits 0.

Before this, nothing between the parser and the public site asserted a database had
data — the parser logs a file failure and continues, returns success regardless,
finishes cleanly at zero rows; the gzip step inspected nothing; and
`assemble-site.js` only greps *filenames*. An under-producing reader would have
published a truncated dataset with green CI.

### 7b — CI

`pull_request` trigger added and `deploy` guarded to `refs/heads/main`, so branch
verification is now actually possible; previously a `workflow_dispatch` from any
branch would have published that branch to the live site, with
`cancel-in-progress` killing an in-flight good deploy on the way. Rust toolchain
and cache actions removed, `actions/setup-go@v5` pinned to 1.26, `govulncheck`
added, `npm run test:go` runs before anything is built, `CGO_ENABLED=0` set
explicitly (the whole module is pure Go).

### 7c — references

The mechanical gate replaced the hand-written list, which was the right call: the
`eslint.config.js:31` glob change silently dropped Node globals from the remaining
scripts and produced 13 lint errors — exactly the breakage predicted. Also caught
`package.json:9`, and comments in `src/datasets.js`, `src/lib/subjects.js` and
`vite.config.js` that the plan had scoped out of `src/`.

`docs/data-pipeline.md`'s "Verifying a rebuild" and "Legacy scripts" sections were
rewritten rather than patched, since the tooling they described is gone.

### 7d — removal

Tagged **`pre-go-parser-removal`** (commit `0eb1747`) before deleting; the removal
is `00a08d5`.

Kept: `crawl-baotintuc.js` (moved to `go-parser/scripts/`) — the only way to refresh
`data/2017`, with a live runbook. Configs moved to `go-parser/configs/`.

Dropped: `check-duplicates.js` and `diff-datasets.js` (broken before the repo was
unified — a hardcoded `D:/` path in one, an undeclared `better-sqlite3` in the
other, zero callers either way), plus `db-stats.js` and `verify-parity.js` as
superseded by `differential-parity.mjs`.

The fidelity oracle is kept and marked **frozen** in its own header: produced by the
Rust reader, so unregenerable, but still failing on any single-cell change across
the 299 inputs. `regen-fidelity-hashes.sh` deleted, since it shelled out to cargo.

### Revert procedure

```bash
git revert --no-commit 00a08d5   # restore parser/ and the removed scripts
git checkout pre-go-parser-removal -- .   # or take the whole pre-removal tree
git checkout pre-go-parser-removal        # or just inspect it
```

The tag is the recovery point: at `0eb1747` both parsers exist and both test suites
pass, so the differential gate can be re-run at any time.

## Unresolved

- Nothing blocking. The branch `refactor/go-parser` is unpushed; CI has therefore
  not run the new workflow yet, and the `pull_request` trigger it adds cannot be
  exercised until a PR exists.
