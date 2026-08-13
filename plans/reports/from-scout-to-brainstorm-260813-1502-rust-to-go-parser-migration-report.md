# Scout Report — Rust parser migration surface (Rust → Go)

Date: 2026-08-13. Branch: main. Scope: `parser/` crate + its build/verify boundary.

## Relevant Files

### Rust crate (~2.3k LOC)
- `parser/Cargo.toml` — 10 deps. `glob` is **dead** (zero references in `src/` or `tests/`).
- `parser/src/schema.rs` (213) — canonical DDL, INSERT SQL, column order, 16 subject regexes. Single source of truth.
- `parser/src/format_detect_2016.rs` (548) — largest file. Per-file/per-sheet 3-way layout detection for 2016 only.
- `parser/src/transform.rs` (409) — `to_ascii` Vietnamese diacritic strip, score regex, tri-state row validation.
- `parser/src/main.rs` (377) — CLI dispatch, file globbing via `fs::read_dir`, transaction boundaries, counters.
- `parser/src/{reader,writer,config,audit,cli,error,lib}.rs` — sheet iteration, SQLite lifecycle, TOML load, audit, clap.
- `parser/configs/{2016,2017,2017-old,2017-old2}.toml` — per-dataset column maps + validation flags.

### Boundary
- `parser/scripts/build-db.js` — sole caller. Hardcodes `parser/target/release/xlsxread` (line 25).
- `parser/scripts/verify-parity.js` + `db-stats.js` — manual parity tool, **not wired into CI**.
- `parser/tests/golden.rs` (591) — 8 tests, white-box (calls library fns, not the binary).
- `.github/workflows/deploy-pages.yml` — `dtolnay/rust-toolchain` + `Swatinem/rust-cache` (workspaces: parser).
- `package.json:8` — `build:rust` = `cargo build --release --manifest-path parser/Cargo.toml`.

## Contracts a rewrite must preserve

**CLI** (only this is depended on by scripts):
```
xlsxread build --schema parser/configs/<id>.toml --input data/<id> --output .build/public/db/<id>.db
xlsxread audit --schema <cfg> --input <dir> --db <db>     # operator-only, never in CI
```

**Output**: SQLite `student` table, 22 cols (`so_bao_danh TEXT PRIMARY KEY`, `ho_ten`, `ho_ten_ascii`, `ngay_sinh`, `ten_cum_thi`, `gioi_tinh`, + 16 `REAL` scores), 3 indexes incl. partial `idx_ten_cum_thi ... WHERE ten_cum_thi IS NOT NULL`. Frontend `sql.js` reads these names directly.

**stdout**: human-facing only. No script parses it. Exit non-zero kills the npm pipeline (`execFileSync`).

## Migration risk

| Area | Risk | Why |
|---|---|---|
| **Legacy `.xls` (BIFF) reading** | **BLOCKING** | See below. |
| calamine cell→string coercion | HIGH | Dates/numerics reach `ngay_sinh` and the score regexes as whatever calamine's `Data::to_string()` renders. Never inspected in-repo; a Go lib will differ. |
| `to_ascii` exactness | MEDIUM | `transform.rs:56` filters the literal range U+0300–U+036F, **not** Unicode category Mn (despite its own doc comment). Go `unicode.Is(unicode.Mn,·)` is more permissive → must copy the range check. Plus explicit `đ/Đ → d` (NFD does not decompose them). |
| TOML strictness | MEDIUM | `deny_unknown_fields` is load-bearing (has a test). Most Go TOML libs ignore unknown keys by default. |
| Stats/stdout parity | MEDIUM | Tri-state `SkipReason` (BlankRow vs EmptyField vs NonNumericSbd) drives the printed counters; a bool would diverge. Stringly-typed dispatch on `dataset_label.contains("old"/"old2")`. |
| Regex | **NONE** | Rust `regex` and Go `regexp` are both RE2. No backrefs/lookaround/`\p{}` anywhere. |
| SQLite | LOW | Plain SQL text, positional `?`, no named params, no pragmas. VACUUM correctly post-COMMIT. |
| clap/serde/thiserror | TRIVIAL | Idiomatic differences only. |

### The blocker, verified on disk

```
data/2016/      4 .xls  + 115 .xlsx
data/2017/     63 .xls              <-- 286 MB, the largest dataset
data/2017-old/          63 .xlsx
data/2017-old2/         54 .xlsx
```

67 legacy `.xls` files across two datasets. `calamine::open_workbook_auto` reads BIFF and OOXML through one API. **Go has no equivalent** — `excelize` is xlsx-only; legacy `.xls` means `extrame/xls` (lightly maintained, incomplete BIFF coverage) or an external converter. This is the crux of the decision, not a detail.

## Also true

- No stated performance or correctness problem with the Rust parser. It is not the pain point; `rust-cache` keeps warm CI builds cheap. 419 MB of Excel dominates deploy runtime regardless of language.
- `verify-parity.js`'s `APPROVED_RECOVERY` baseline is frozen against the **pre-refactor** implementation and cannot be regenerated. Reusable as a *method* for Rust-vs-Go, but needs a fresh baseline captured from current Rust output first.
- `golden.rs` is white-box (calls `xlsxread::reader::process_file` etc.). Its fixtures are hand-built minimal OOXML + `zip` — that trick ports to Go's `archive/zip` trivially. Black-box CLI tests would be the language-agnostic seam.
- No fixture covers `.xls` at all — all 3 fixtures are xlsx. So the golden suite would not catch an `.xls` regression.

## Unresolved Questions

1. What is the actual motivation for Go? No perf/correctness defect is visible in the repo. Answer determines whether migration is warranted at all.
2. How does calamine render date cells into `ngay_sinh` today? Must be probed empirically against real 2016/2017 files before any Go xlsx library is chosen.
3. Is a fresh Rust-output parity baseline acceptable as the gate, given the original baseline is unregenerable?
