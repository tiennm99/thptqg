---
phase: 3
title: Transform core
status: completed
priority: P1
dependencies:
  - 2
effort: ''
---

# Phase 3: Transform core

## Overview

Port `transform.rs` — Vietnamese diacritic stripping, score regex extraction, row validation,
and the fixed-column row transform. Pure functions, no I/O. This is where subtle divergence is
most likely and most invisible.

## Requirements

- Functional: `ToAscii` byte-identical to Rust for all inputs; score parsing identical;
  validation reproduces Rust's **two distinct blank-row paths**, not a bool.
- Non-functional: **all 29** unit tests in `transform.rs`'s test module (`:201-409`) transfer as
  the Go test suite.

## Architecture

Signature mirrors Rust's, which takes `strip_blank_rows` and `all_blank` as explicit
parameters (`transform.rs:101-107`) — a 2-arg Go version structurally cannot reproduce either
blank-row path.

```go
func ToAscii(s string) string

type SkipReason int
const (
    SkipNone SkipReason = iota
    SkipBlankRow
    SkipEmptyField    // counted as source row, then skipped
    SkipNonNumericSbd // counted as source row, then skipped
)

func ParseScores(diemThi string) map[string]float64
func ValidateRow(hoTen, soBaoDanh string, cfg *config.DatasetConfig,
                 stripBlankRows, allBlank bool) SkipReason
func TransformRow(row []reader.Cell, cfg *config.DatasetConfig) (*Student, error)
```

## Related Code Files

- Create: `go-parser/internal/transform/transform.go`, `transform_test.go`
- Reference: `parser/src/transform.rs` — `:52-64` ToAscii, `:89-97` SkipReason,
  `:101-107` validate_row signature, `:130-143` parse_scores, `:162` the `.expect()`,
  **`:201-409` the test module (29 tests)**

## Implementation Steps

**Tests first** — port all 29, not a subset.

1. Port **every** `#[test]` in `transform.rs`'s module (`:201-409`) into `transform_test.go`,
   including every Vietnamese fixture string. Stating it as "every test in the module" rather
   than a line range is deliberate: the range `:213-315` contains only the 20 `to_ascii` cases,
   and the 9 outside it (`:323, 332, 342, 359, 365, 374, 383, 393, 400`) are exactly the
   `parse_scores` and `validate_row` tests this phase calls its highest-value traps —
   including `validate_non_numeric_sbd_rejected` (`:383`) and `validate_blank_row_skipped`
   (`:400`).
2. Add the specific edge cases below as extra tests.
3. Implement `ToAscii`, `ParseScores`, `ValidateRow`, `TransformRow` until green.

## The three exactness traps

These are the highest-value details in the whole plan. Each is a silent corruption if missed.

1. **`ToAscii` filters a literal codepoint range, not a Unicode category.**
   `transform.rs:56` filters `'\u{0300}'..='\u{036f}'`. The *inline* comment at `:53` says
   "Unicode category M" — **that comment is wrong, the code is the spec** (the doc comment at
   `:49` correctly states the range). Go's `unicode.Is(unicode.Mn, r)` is strictly more
   permissive and would diverge on marks outside U+0300–U+036F. Implement the literal range
   check:
   ```go
   // NFD, then drop combining marks in U+0300..U+036F only — matches parser/src/transform.rs:56.
   // Deliberately NOT unicode.Mn, which is broader and would strip more than Rust does.
   ```
2. **`đ`/`Đ` are not decomposed by NFD** — they are precomposed Latin letters, so NFD leaves
   them intact. An explicit replacement to `d` is required (`transform.rs:60`), and it happens
   **before** lowercasing (`:63`). Preserve that order.
3. **There are TWO blank-row paths with opposite outcomes, and the caller owns the split.**
   The "not counted as a source row" behavior lives in `main.rs:135-137`, which returns
   *before* `total_source_rows += 1` at `:140`. Separately, when `validate_row` itself returns
   `Err(SkipReason::BlankRow)`, `main.rs:151` matches it as `=> {}` — which **falls through to
   transform and insert**. Same enum variant, opposite outcome, decided by which call site you
   are in. Do not collapse these; reproduce both call sites in Phase 4's build loop and keep
   `ValidateRow`'s 5-parameter shape so both remain expressible.

## Other details

- Order: NFD → filter range → replace `đ`/`Đ` → lowercase.
- **`diem_thi` is read WITHOUT `.trim()`** (`transform.rs:172-175`), while `ho_ten`,
  `ngay_sinh`, and `so_bao_danh` all trim via the closure at `:164-168`. Leading whitespace in
  the score cell reaches the regexes intact. Replicate the asymmetry.
- `ParseScores` uses first-match-anywhere (Rust `captures`, Go `FindStringSubmatch` — same
  default, unanchored). No change needed.
- Rust checks `is_finite()` on parsed scores (`transform.rs:136`). Unreachable given the
  pattern, but keep it for defensive parity.
- `require_numeric_sbd` is a digits-only check, not `strconv.Atoi` — a leading `+`, a `_`, or
  whitespace must fail. `Atoi` accepts a leading sign; use an explicit digit scan.
- `TransformRow` is only called on the non-2016 path, where `Columns` is non-nil. Rust relies
  on `.expect()` (`transform.rs:162`); in Go return an error rather than panicking.

## Success Criteria

- [x] **All 29** tests from `transform.rs:201-409` ported and passing
- [x] `ToAscii("Nguyễn Văn Đức") == "nguyen van duc"`
- [x] `ToAscii` uses the literal U+0300–U+036F range, with a comment saying why not `unicode.Mn`
- [x] `đ`/`Đ` → `d` verified independently of the NFD path
- [x] `ValidateRow` keeps the 5-parameter Rust shape
- [x] Both blank-row paths covered by tests (skip-before-count vs fall-through-to-insert)
- [x] `diem_thi` untrimmed while the other three fields are trimmed
- [x] Numeric-SBD check rejects `+123`, `12 3`, `1.0`, `ABC123`
- [x] Score parsing matches on the multi-subject fixture
      (`"Toán: 8.5  Ngữ văn: 7.0  Tiếng Anh: 9.25"`)

## Risk Assessment

| Risk | Mitigation |
|---|---|
| `unicode.Mn` used instead of the literal range | Called out explicitly; comment required in code |
| `đ` silently dropped instead of → `d` | Dedicated test |
| Tri-state collapsed to bool | Counter-distinguishing test; caught again in Phase 6 |
| `strconv.Atoi` accepts signs the Rust check rejects | Explicit rejection cases |

## RESULT — 2026-08-13: **PASS**

All 29 tests from `transform.rs:201-409` ported and green, plus guards for each trap.

**Cross-checked against Rust on real data, not just the unit cases.** A Rust-built database is
its own oracle: every row carries `ho_ten` next to the `ho_ten_ascii` Rust derived from it, so
the table is a name→slug corpus orders of magnitude larger than 20 hand-picked names.

| dataset | names compared | mismatches |
|---|---|---|
| `2016` | 877,461 | **0** |
| `2017-old2` | 679,764 | **0** |

Kept as `TestToAsciiAgainstRustOutput`, which skips unless `GO_PARSER_RUST_DB` points at a
Rust-built database — so the default suite stays hermetic while the check stays reusable for
Phase 6.

### Traps handled

- `ToAscii` filters the literal range U+0300–U+036F. `TestToAsciiUsesLiteralRangeNotUnicodeMn`
  asserts a mark *outside* that range (U+0654, which is in `Mn`) survives — so swapping in
  `unicode.Is(unicode.Mn, r)` fails the suite rather than silently changing `ho_ten_ascii`.
- `đ`/`Đ` → `d` before lowercasing, tested independently of the NFD path.
- `ValidateRow` keeps Rust's 5-parameter shape, so both blank-row paths stay expressible. The
  caller-side split is documented on `SkipReason` for Phase 4.
- Numeric-SBD is a digit scan, not `strconv.Atoi`; `TestValidateNumericSbdIsDigitScanNotAtoi`
  rejects `+123`, `-123`, `1.0`, and full-width digits.
- `diem_thi` is read untrimmed while the other three fields are trimmed — pinned by test so it
  cannot be "tidied away".
