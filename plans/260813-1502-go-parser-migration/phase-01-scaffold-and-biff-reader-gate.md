---
phase: 1
title: Scaffold and reader fidelity gate
status: completed
priority: P1
dependencies: []
effort: ''
---

# Phase 1: Scaffold and reader fidelity gate

## Overview

Scaffold `go-parser/` and settle the question that governs everything downstream: **does the Go
reader produce, for every cell of every sheet of all 299 files, the exact string calamine
produces?** Not "does it open" — the exact string, because that string is what gets stored and
regex-matched.

This phase also produces the answer that unblocks the `Data`-typed tests in Phases 4 and 5.

## Requirements

- Functional: byte-identical cell stringification vs calamine across all 299 input files, both
  formats, including dates, numerics, and empty cells.
- Non-functional: one reader contract, defined here, used unchanged by Phase 4.

## What was believed going in (and how it held up)

Red-team testing had swept all 67 BIFF files with `extrame/xls` and reported **0 failures,
0 panics**, correct Vietnamese at row 60,000. That was taken as evidence BIFF *readability*
was settled and only cell-value fidelity remained open.

**That evidence did not survive contact with a cell-level comparison** — see the RESULT below.
"Opens without panic" and a handful of spot-checks are not a fidelity test when 28% of cells
are correct. The lesson generalises: for every remaining phase, compare against ground truth
cell-by-cell, never by sampling.

Hazards that *did* hold up and were designed around:
- excelize applies number formats by default → **set `RawCellValue: true`** and verify.
- excelize `GetRows` trims trailing blank cells → rows are ragged; calamine's are rectangular.

Hazards that turned out not to exist in this corpus: date-serial rendering (zero `DateTime`
cells anywhere) and non-A1 used-range origins (all ranges start at (0,0)).

## Architecture — AS BUILT

The shipped API differs from the original sketch (which took a `DatasetConfig` and did header
skipping inline). The reader deliberately knows **nothing** about datasets: it reports every
sheet and every row exactly as calamine would, and all policy — sheet selection, header
skipping, blank-row handling — belongs to the Phase 4 build loop. That keeps the fidelity
contract testable in isolation, which is what made the 299/299 oracle possible.

```go
// go-parser/internal/reader
type Cell struct {
    Str     string // exactly what calamine's Data::to_string() yields
    IsEmpty bool   // calamine Data::Empty; diagnostic only — never compare on it
}

type Sheet struct{ Index int; Name string; Height, Width int } // used-range geometry

type RowFunc func(sheet Sheet, rowIdx int, row []Cell) error

type Workbook interface {
    Sheets() []Sheet                      // workbook order, all sheets
    EachRow(sheetIdx int, fn RowFunc) error
    Close() error
}

func Open(path string) (Workbook, error)  // dispatches on extension
```

Rows are padded to the sheet's used-range width. Width is load-bearing: every column read
downstream is positional, so a trimmed tail silently NULLs columns.

**Implementation note:** both backends materialise a workbook's rows rather than streaming
(`rows [][][]Cell`). Peak input is `data/2017/ha-noi.xls` at 72,276 rows × 4 columns; the full
299-file suite runs in 77s. Revisit only if a future dataset is far larger.

## Related Code Files

- Create: `go-parser/go.mod`, `go-parser/internal/reader/{reader.go,xls.go,xlsx.go}` + tests,
  `go-parser/internal/reader/fidelity_test.go`, `go-parser/testdata/`
- Reference (do not modify): `parser/src/reader.rs` (incl. its 7 tests at `:112-197`),
  `parser/Cargo.toml`
- Read-only inputs: all 299 files under `data/`

## Implementation Steps

**Tests first.** The oracle is Rust; extract it before writing Go.

1. **Generate ground truth.** Add a throwaway Rust bin (or `#[test]`) that, for a chosen file,
   dumps every sheet: sheet name, used-range dimensions, and every cell rendered exactly as
   `Data::to_string()` plus an `is_empty` flag.
2. **Commit hashes, not rows.** For each sampled file store sheet names, per-sheet row/column
   counts, and a SHA-256 over the canonical dump — **not** the dump itself. The raw dumps are
   real student names and birthdates; `parser/tests/fixtures/README.md` documents that all
   fixture PII is replaced with synthetic values, and committing real rows would reverse that
   convention and outlive the data they came from. Keep dumps under `.gitignore` and regenerate
   from Rust on demand (Rust still builds — that is this plan's whole advantage).
3. **Sample must cover both formats and the risky cell types.** At minimum: 3 `.xls`
   (2 from `data/2017`, 1 from `data/2016`) **and** 3 `.xlsx` (one each from `data/2016`,
   `data/2017-old`, `data/2017-old2`), each chosen to contain a date cell and a numeric SBD.
4. Write `fidelity_test.go` asserting the Go reader reproduces every committed hash. It fails —
   nothing is implemented.
5. Scaffold: `go mod init`, add deps **at pinned versions**, commit `go.sum`.
6. Implement both readers until the hashes match.
7. **Full sweep, all 299 files**: for each, assert sheet names in order, per-sheet row count,
   and per-sheet used-range width all match calamine. Record failures per file.
8. **Record the stringification answer** in this file — the literal rendering of a date cell, a
   float score, and a numeric SBD. Phases 4 and 5 depend on it to port `Data`-typed tests
   without guessing.

## Decision gate — *as written before execution; see RESULT below for the outcome*

- **PASS**: all 299 files match on sheet names, per-sheet row counts, widths, and sampled
  content hashes.
- **FAIL** → stop and escalate. Do not proceed with a partial pass. The pre-planned fallback
  (`.xls → .xlsx` conversion) is **deferred by user decision** and changes committed data, so
  re-opening it is the user's call, not the implementer's.

## RESULT — 2026-08-13: **PASS — 299 / 299 exact**

Every input file's canonical cell dump is byte-identical to calamine's, verified by SHA-256.
Locked in as `go test ./internal/reader/` (77s for the full corpus) against the committed
oracle `go-parser/testdata/reader-fidelity-hashes.tsv`.

Reached only after replacing the BIFF library. The first attempt failed hard; the record of
that is kept below because it is the reason the reader is built the way it is.

### Final reader stack

| Format | Library | Result |
|---|---|---|
| `.xls` (67 files) | **`github.com/pbnjay/grate`** | exact |
| `.xlsx` (232 files) | `github.com/xuri/excelize/v2 v2.11.0` | exact |

Five corrections were needed to match calamine, each verified against ground truth:

1. **`RawCellValue: true`** — otherwise excelize applies the cell number format.
2. **Rows padded to used-range width** — excelize trims trailing blank cells; calamine returns
   a rectangle. Width is load-bearing: `diem_thi` is the last column for 2017.
3. **grate merged-cell markers blanked** — grate fills merge-covered cells with `→`/`⇥`/`↓`/`⤓`
   (its exported constants); calamine reports them empty. 19 cells, in the merged title block
   of one 2016 file. Only an exact whole-value match is blanked.
4. **Numeric re-rendering, gated on cell type** — calamine parses numeric cells to f64 and
   renders with Rust's `Display`, so `6.0` becomes `6`. Applying that by value alone corrupts
   shared strings that merely look numeric: it turned `6.00`→`6`, `NAN`→`NaN`, and would have
   destroyed leading zeros in `so_bao_danh`. The type check is what makes it safe. Note
   excelize reports **`CellTypeUnset`, not `CellTypeNumber`**, for plain numeric cells, because
   OOXML omits the `t` attribute and excelize has no map entry for an empty one.
5. **CRLF restoration in shared strings** — Go's `encoding/xml` performs the line-ending
   normalisation XML 1.0 mandates (CRLF and lone CR → LF); calamine reads raw bytes and keeps
   CRLF. This reaches the database: 2,233 `TEN_CUMTHI` values in one 2016 file, populating
   `ten_cum_thi`. Fixed by rewriting literal CR to `&#13;` before decoding — character
   references are exempt from that normalisation — and mapping the normalised form back.
   A blanket `\n`→`\r\n` would have been wrong: 117 files carry a lone CR with no LF.

**Known divergence, behaviorally inert:** none remaining. The trailing 1×1 empty sheet in 63
`2017-old` and 53 `2017-old2` files is now reproduced exactly, using `GetCellType(A1)` to tell
an empty-shared-string cell (`CellTypeSharedString`) from a genuinely absent one
(`CellTypeUnset`, 230 such sheets in 2016).

### The rejected library: `extrame/xls` — HARD FAIL (67 files)

Full canonical diff of `data/2017/an-giang.xls` (56,244 cells) against calamine:

| Class | Cells | Share |
|---|---|---|
| Identical | 16,016 | 28% |
| **Different content** (corruption) | 38,664 | **69%** |
| **Rust has value, Go empty** (data loss) | 15,629 | 28% |
| Whitespace-only | 0 | — |

Only 28% of cells are read correctly. Three distinct defect classes, all confirmed against
ground truth:

1. **Undecoded BIFF bytes leak through.** Row 56 col 0: calamine `HỒ THỊ NHƯ Ý`; extrame
   `"\f\x00\x01H\x00Ò\x1e \x00T\x00H\x00Ê\x1e \x00N\x00H\x00¯\x01 \x00Ý\x00\b\x00\x0051009967t\x00…"`
   — raw UTF-16LE plus record framing, with the neighbouring SBD and score cells spliced in.
2. **Content teleports between cells.** extrame's (14055, 0) is calamine's **(6500, 3)**.
3. **Tail rows silently lost.** calamine rows 14056-14060 hold real students
   (`NGUYỄN HỮU ÁI` … `HUỲNH VĂN KIÊN`); extrame returns them blank.
4. Header cell (0,0) `HO_TEN` dropped — would defeat `is_header_row` and ingest the header
   as data.
5. `sh.Row(r)` panics (nil deref, `worksheet.go:30`) for `r > MaxRow`.

**Not a configuration problem.** Identical garbage under charsets `utf-8`, `utf-16`, `utf-16le`,
`windows-1258`, `cp1252`, and empty. Not an index-arithmetic problem on our side either —
geometry matches exactly (70,308 canonical lines both sides, `SHEET 0 Sheet1 14061 4` on both)
after correcting `LastCol()` exclusivity and the used-range height rule.

**This refutes the red-team finding that `extrame/xls` reads the corpus correctly.** That sweep
tested "opens without panic" plus a few spot-checks; spot-checks pass because 28% of cells are
right and the early rows of a file are among them. Cell-level comparison against ground truth
is what exposed it.

Library survey (2026-08-13): `youkuang/xls` and `f2xb/xls` are forks of `extrame/xls` and carry
the same defect. `qax-os/excelize` **is** excelize and does not read BIFF at all. The
independent implementations are `pbnjay/grate` (chosen — reproduced calamine exactly on all 67
files first try, needing only the merged-marker correction) and `shakinm/xlsReader` (not
evaluated; grate passed).

### Corpus facts established (worth keeping regardless of the decision)

Scanned all 299 files, 15.98M cells:
- **Zero `DateTime` cells.** Also zero `Int`, `Bool`, `Error`, `DateTimeIso`, `DurationIso`.
  Only `String` (15.1M), `Empty` (722k), `Float` (133k) occur. **The date-serial divergence
  that this plan called its dominant risk does not exist in this corpus** — `ngay_sinh` is
  stored as text everywhere.
- **Every used range starts at (0,0)** — the used-range-origin concern is moot.
- Floats occur only in `2016` (53,008) and `2017-old2` (80,121); none in `2017` or `2017-old`.
  All render as plain decimals, no exponents, max 2 decimal places.
- Trailing empty sheets: 293 sheets at height 0, 116 at height 1.
- The "63 empty" rows in `docs/data-pipeline.md:114` for 2017 do **not** come from trailing
  sheets — calamine reports height 0 for all 63 of those and yields no rows from them. The
  red team's stated mechanism for that count is wrong; provenance is a Phase 4 question.

### Artefacts

- `parser/examples/dump_cells.rs`, `parser/examples/scan_kinds.rs` — throwaway Rust ground-truth
  tooling (delete after migration)
- `go-parser/` — module, `internal/reader` (both formats), `cmd/dumpcells`
- Dumps are regenerable and gitignored; no PII committed, per the convention in
  `parser/tests/fixtures/README.md`

## Specific things to verify, not assume

- **Per-sheet row counts, including empty sheets.** All 63 `data/2017` files carry a trailing
  sheet that calamine renders as one blank row. `docs/data-pipeline.md:114` records the
  consequence exactly: `2017 | 861,131 source rows | 63 empty | 861,068 DB rows`. A Go reader
  that skips zero-row sheets produces an identical database and silently different counters.
- **Date cells** → `ngay_sinh`, stored verbatim. calamine prints the raw serial
  (`datatype.rs:771-775`); excelize applies the number format unless `RawCellValue: true`.
- **Numeric cells** → `so_bao_danh`, a `TEXT PRIMARY KEY`. Trailing `.0`? Scientific notation?
  A difference here re-keys the table.
- **Empty vs blank**: `reader.rs:42` checks both `Data::Empty` and stringified-empty, implying
  calamine emits empty-but-not-`Empty` cells. The `Cell.IsEmpty` field exists for this.
- **Row width / used-range origin**: see "What is already known".
- **Sheet order**: `sheet_mode = "all"` for 2016 and 2017. calamine's `sheet_names()` and
  excelize's `GetSheetList` can disagree when `workbook.xml` order differs from `sheetId` order.
  Order determines which duplicate SBD survives `INSERT OR REPLACE`.

## Dependency trust

- Pin every dependency to an exact version/pseudo-version; commit `go.sum`.
- Record in this file that `extrame/xls` is effectively unmaintained (last push 2023-09-12,
  53 open issues, no valid `go.mod`) and that it transitively adds `tealeg/xlsx`.
- Note the open excelize advisory `GHSA-h69g-9hx6-f3v4` (unbounded row-index allocation). The
  2017 refresh runbook (`docs/data-pipeline.md:137`) feeds network-downloaded spreadsheets
  straight into the parser, so this is a live path.
- `govulncheck` is added to CI in Phase 7b.

## Success Criteria

- [ ] `go-parser/` builds; `go test ./...` runs; `go.sum` committed with pinned versions
- [ ] Ground-truth **hashes** (not raw rows) committed for 3 `.xls` + 3 `.xlsx` files
- [ ] Go reader reproduces every committed hash
- [ ] All 299 files: sheet names in order, per-sheet row counts, and widths match calamine —
      **including the 63 trailing empty sheets in `data/2017`**
- [x] `RawCellValue` settled and justified in writing
- [x] Date, float, and numeric-SBD renderings recorded literally in this file
- [x] One reader contract, with `Cell.IsEmpty` and padded row width
- [~] `parser/src/reader.rs`'s 7 tests (`:112-197`) — **moved to Phase 4.** They exercise
      `is_header_row` and `is_all_blank`, which are dataset policy and therefore live in the
      build loop, not the reader package. Recorded here rather than silently dropped.
- [x] Dependency trust notes recorded
- [x] Explicit PASS/FAIL recorded

## Risk Assessment

| Risk | Mitigation |
|---|---|
| `.xlsx` date/number formatting differs | The primary target of this gate; `RawCellValue` verified against hashes |
| Trailing-blank trimming NULLs tail columns | Row-width equality asserted for all 299 files |
| Empty-sheet skipping breaks counters | Per-sheet row counts asserted, including empty sheets |
| Real PII committed as fixtures | Hashes committed instead; dumps gitignored and regenerable |
| `extrame/xls` unmaintained / OOM issues | Pinned pseudo-version; full sweep measures memory |
| Partial pass rationalized into a PASS | Gate is binary; fallback is a user decision |
