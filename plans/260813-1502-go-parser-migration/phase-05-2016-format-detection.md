---
phase: 5
title: 2016 format detection
status: completed
priority: P1
dependencies:
  - 4
effort: ''
---

# Phase 5: 2016 format detection

## Overview

Port `format_detect_2016.rs` (548 lines, the largest file in the crate) — per-file, per-sheet
runtime detection across the three inconsistent 2016 layouts. This is institutional knowledge
encoded as literals; there is no abstraction to derive it from.

## Requirements

- Functional: all three 2016 layouts detected and parsed identically to Rust.
- Non-functional: every hardcoded literal (header token list, column positions, gender
  allowlist) copied verbatim.

## Architecture

```go
type Format int
const (
    FormatSeparateScores Format = iota  // SBD(0) HOTEN(1) TOAN(2)...NGOAINGU-total(11)
    FormatMapped                        // dynamic column lookup by header name
    FormatDefault                       // headerless: fixed (0,1,2,3,4,5)
)

// 17 tokens, verbatim from format_detect_2016.rs:37-53.
// NOTE: the token "SINH " has a TRAILING SPACE. Copy it exactly; trimming it changes detection.
var KnownHeaders = []string{...}

func IsHeaderRow2016(row []string) bool
func DetectFormat(headerRow []string) (Format, *ColumnIdx)
func ProcessRow2016(row []string, f Format, idx *ColumnIdx) (*transform.Student, error)
```

Note `FormatDefault` is not a separate code path — it is `FormatMapped` with the fixed index
tuple `(0,1,Some(2),Some(3),Some(4),5)` (`format_detect_2016.rs:295-306`). Keep that structure
rather than duplicating logic.

## Related Code Files

- Create: `go-parser/internal/format2016/format2016.go`, `format2016_test.go`
- Modify: `go-parser/cmd/xlsxread/main.go` (dispatch when `format_detection == "thptqg2016"`)
- Reference: `parser/src/format_detect_2016.rs`, `parser/src/main.rs:211-377`

## Implementation Steps

**Tests first**, one per format plus the quirks below.

1. Port the **11 tests** in `format_detect_2016.rs`. They build fixtures from `calamine::Data`
   values (9 uses of `Data::Float`, e.g. `:440-449`; `Data::String` via the `s()` helper at
   `:348`). **Phase 1 settled the translation**, so no guessing is needed:

   | Rust fixture | Go fixture (`reader.Cell.Str`) |
   |---|---|
   | `Data::String(s)` | `s` verbatim |
   | `Data::Float(8.0)` | `"8"` — Rust `f64` Display drops `.0`; matches Go `FormatFloat(v,'f',-1,64)` |
   | `Data::Float(8.5)` / `Data::Float(0.25)` | `"8.5"` / `"0.25"` |
   | `Data::Empty` | `""` with `IsEmpty: true` |

   Corpus-verified: float renderings are plain decimals only — no exponents, at most 2 decimal
   places, across all 133,129 float cells. `Data::DateTime`, `Int`, `Bool`, and `Error` never
   occur, so no fixture needs them.
2. Build `excelize`-generated fixtures for each of the three layouts.
3. Write detection tests: `SeparateScores` header → correct format; `Mapped` header →
   correct dynamic indices; no recognized header → `Default`.
4. Write tests for each quirk in the section below.
5. Implement and wire the dispatch.

## Quirks that are not bugs — replicate verbatim

- **A parsed score of `0.0` becomes NULL** in the separate-scores format
  (`format_detect_2016.rs:165`). This replicates a JS `parseFloat(x) || null` falsy quirk.
  A literal zero score is indistinguishable from "no score". Do not fix.
- **Gender allowlist is exactly `"Nam"` / `"Nữ"`** — anything else becomes NULL
  (`:263-271`). Not a general enum; a two-value literal check.
- **`SeparateScores` maps column 11 (foreign-language total) to `tieng_anh`** (`:201-202`).
  `tieng_phap`/`tieng_duc`/`tieng_nhat`/`tieng_trung` are structurally unreachable in this
  format, and `ngay_sinh`/`ten_cum_thi`/`gioi_tinh` are always NULL (`:174-175, 211-213`).
- **Leaked-header guard**: if the SBD or HO_TEN cell value is itself a known header token, skip
  the row (`:244-250`). Defends against repeated headers on later sheets.
- **`Mapped` falls back to column 1 for `ho_ten`** when not found by name (`:132`).
- **Detection is per-sheet, not per-file** (`main.rs:344-349`) — sheets within one file may
  legitimately detect as different formats. Do not cache detection at file level.
- **Rows shorter than 2 cells are skipped** regardless of validation config (`main.rs:351-353`).
- `SeparateScores` has **no free-text score cell** — scores are parsed by direct float
  conversion, never by the subject regexes.

## Success Criteria

- [x] All 11 tests from `format_detect_2016.rs` ported, using Phase 1's recorded stringification
- [x] All three formats detected correctly from their header rows
- [x] `KnownHeaders` is all 17 tokens, verbatim — including `"SINH "` with its trailing space
- [x] `0.0` → NULL test passes for the separate-scores path
- [x] Gender allowlist test: `"Nam"`/`"Nữ"` pass, `"Unknown"`/`""`/`"M"` → NULL
- [x] Leaked-header row skipped
- [x] Per-sheet detection verified with a fixture whose two sheets differ in format
- [x] Short-row guard covered
- [x] Real 2016 build succeeds; row count equals the Rust-built 2016 DB
- [x] All 4 datasets now build with the Go binary

## Risk Assessment

| Risk | Mitigation |
|---|---|
| A quirk "cleaned up" during porting | Each listed explicitly with a required test |
| Detection cached per file instead of per sheet | Two-sheet mixed-format fixture |
| 2016 has 4 `.xls` + 115 `.xlsx` — mixed formats in one dataset | Phase 1 already proved both readers |
| Column-position literals transcribed wrong | Row-count parity against Rust catches gross errors; field-level diff in Phase 6 catches subtle ones |

## RESULT — 2026-08-13: **PASS**

2016 builds, and **stdout is byte-identical to Rust** — 128 lines covering all 119 per-file
counts plus the stats block. Only the `--output` path differs.

| | value |
|---|---|
| Source rows (post-header) | 877,464 |
| DB rows | **877,461** (documented: 877,461) |
| Audit line | `3 row(s) collapsed (duplicate SBDs overwriting).` |
| Size | 223.2 MB, same as Rust |
| stderr | empty on both sides |

All 11 `format_detect_2016.rs` tests ported, plus a guard per quirk: the `"SINH "` trailing
space, the `0` → NULL falsy rule, the exactly-`Nam`/`Nữ` gender allowlist, the leaked-header
guard, the col-1 `ho_ten` fallback, order-independent index resolution, and a test asserting
`FormatDefault` behaves identically to the equivalent `FormatMapped` so it cannot drift into a
separate code path.

Phase 1's recorded `Data` → string translation made the fixtures exact rather than guessed.

### Counter subtlety preserved

The 2016 path never increments `skipped` (main.rs:251 declares it immutable), so rows rejected
by `ProcessRow2016` are counted as source rows but not as skipped. The stats block therefore
reports `insertable == source rows`, and the Audit line absorbs the gap — which is why 2016
prints `3 row(s) collapsed` rather than a skip count. Reproducing this exactly is what makes
the stdout match.
