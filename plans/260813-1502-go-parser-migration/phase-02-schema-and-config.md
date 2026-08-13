---
phase: 2
title: Schema and config
status: completed
priority: P1
dependencies:
  - 1
effort: ''
---

# Phase 2: Schema and config

## Overview

Port the two pure, I/O-free modules: `schema.rs` (DDL, INSERT SQL, column order, 16 subject
regexes) and `config.rs` (strict YAML loading). No file or database access — fully unit-testable.

Also **decides the SQLite driver** (see below) — it governs the CI shape and the integrity
story, so it is settled here rather than deferred to Phase 4.

## Requirements

- Functional: identical DDL text, identical column ordering, identical regex patterns; YAML
  loading that **rejects unknown fields**.
- Non-functional: `schema` stays the single source of truth, as in Rust. No duplicated column
  lists anywhere else in the port.

## Carried forward from Phase 1

- Reader is done and exact; it exposes `reader.Cell{Str, IsEmpty}`. Config work is independent
  of it.
- The corpus contains **only** `String`, `Empty`, and `Float` cell kinds — no dates, ints,
  bools, or errors. Nothing in `config.go` needs date or type-coercion handling.
- Go 1.26.5 / linux-arm64 confirmed working; `go.mod` currently pulls `pbnjay/grate` and
  `excelize/v2 v2.11.0`, both pure Go. **The SQLite driver choice below is what decides whether
  this module stays cgo-free.**

## Decision: SQLite driver

Open question 2 in `plan.md`. Record the choice and rationale in this file.

| Option | For | Against |
|---|---|---|
| `modernc.org/sqlite` | Pure Go, no cgo, trivial ARM64 CI and cross-compilation. Widely used (3,500+ importers) | **Machine-transpiled** SQLite, not the upstream C amalgamation. Its correctness argument is "the transpiler is correct", not "this is the code the SQLite authors tested". Requires exact `modernc.org/libc` version matching |
| `mattn/go-sqlite3` | Real upstream SQLite C, matching what `rusqlite --bundled` vendors (`Cargo.lock:520` `libsqlite3-sys 0.30.1`) | cgo: slower CI, cross-compilation friction, needs a C toolchain in the workflow |

This writes a published 1.5M-row dataset, so the integrity story is a real consideration, not a
formality. Phase 6's full-table checksum is the compensating control either way. Pin the exact
version and commit it to `go.sum`.

### DECIDED — `modernc.org/sqlite` (user, 2026-08-13)

Empirically verified on this linux/arm64 box before deciding: `modernc.org/sqlite v1.56.0`
embeds **SQLite 3.53.3** and handles the exact SQL this parser uses — the full DDL including
the partial `idx_ten_cum_thi` index, `INSERT OR REPLACE`, and `VACUUM`, producing 3 indexes.

Rationale:
- Keeps the module **entirely cgo-free** — `grate`, `excelize` and `yaml.v3` are all pure Go, so
  Phase 7 sets `CGO_ENABLED=0`, needs no C toolchain in CI, and cross-compiles trivially.
- The SQL surface is deliberately plain: no CTEs, window functions, triggers or extensions.
  That is the part of SQLite a transpiled port is least likely to get wrong.
- Phase 6's full-table SHA-256 plus `PRAGMA table_info`/`index_list` comparison against live
  Rust output is a real compensating control for the transpilation risk.

Accepted trade-off: it is a machine-transpiled SQLite, not the upstream C amalgamation that
`rusqlite --bundled` vendors (`libsqlite3-sys 0.30.1`, ~3.46). Version parity was never a goal —
`plan.md` explicitly rules byte-identical databases out as a criterion, since SQLite stamps its
own version into header bytes 96-99.

If Phase 6 ever shows a divergence traceable to the driver, `mattn/go-sqlite3` is the fallback:
`gcc` is present locally and on GitHub runners, so the switch costs only `CGO_ENABLED=1`.

## Architecture

```go
// internal/schema
const DDL = `...`           // verbatim from parser/src/schema.rs:27-54
const InsertSQL = `...`     // positional ?, order fixed by IdentityFields + ScoreFields
var IdentityFields = []string{...}   // 6
var ScoreFields = []string{...}      // 16
var ScorePatterns = map[string]*regexp.Regexp{...}  // compiled once at init

// internal/config
type DatasetConfig struct {
    FormatDetection *string
    Reader     ReaderCfg      // SheetMode "all"|"first", StripBlankRows bool
    Columns    *ColumnMap     // nil when FormatDetection is set
    Validation ValidationCfg  // 3 bools
    Header     HeaderCfg      // Tokens []string
}
func Load(path string) (*DatasetConfig, error)
```

## Related Code Files

- Create: `go-parser/internal/schema/schema.go`, `schema_test.go`,
  `go-parser/internal/config/config.go`, `config_test.go`
- Reference: `parser/src/schema.rs`, `parser/src/config.rs`, `parser/src/error.rs`
- Consumed unchanged: `parser/configs/{2016,2017,2017-old,2017-old2}.yml` — the Go binary
  reads the **same** config files; do not copy or fork them

## Implementation Steps

**Tests first**, ported from the Rust unit tests in `config.rs:131-137` and the DDL/schema
constants.

1. Write `schema_test.go`: assert DDL string equals the Rust DDL verbatim (paste it as the
   expected literal), assert `len(IdentityFields)+len(ScoreFields) == 22`, assert `InsertSQL`
   placeholder count matches, assert all 16 regexes compile.
2. Write `config_test.go`: port **all 4** tests from `config.rs:114-152` (not just the one at
   `:131-137`). Load all 4 real configs and assert the field values the scout recorded (2016 has
   no `columns:` mapping and `format_detection: thptqg2016`; 2017-old is `sheet_mode="first"`;
   2017-old2 has `strip_blank_rows: true` + `require_numeric_sbd: true`). Include the
   **unknown-field rejection** test — port of `config_rejects_leftover_sql_sections`.
3. Implement `schema.go`. Copy DDL, INSERT SQL, field lists, and the 16 patterns **verbatim**.
   Do not retype the Vietnamese pattern literals — copy them, byte-exactness matters.
4. Implement `config.go` with a YAML decoder configured for strict decoding.

## Specific things to get right

- **`deny_unknown_fields` is load-bearing** and has a test in Rust. Go YAML decoders ignore
  unknown keys by default; `gopkg.in/yaml.v3` enables the check with `KnownFields(true)`.
  Write the rejection test with a *valid* YAML key — a TOML-style `key = 1` line fails as a
  parse error instead, so the test would pass without proving anything.
- **`Columns` is nil for 2016** — represent as a pointer/optional, not a zero value. A zero
  `ColumnMap` would silently mean "all columns are index 0".
- **Regexes**: Rust `regex` and Go `regexp` are both RE2, and the scout confirmed no
  backreferences, lookaround, or `\p{}` in any pattern. This is the one zero-risk area — but
  the patterns contain literal Vietnamese (`Ngữ văn`, `Tiếng Đức`), so copy, never retype.
- **Partial index** in the DDL (`... WHERE ten_cum_thi IS NOT NULL`) is SQLite-specific and
  must survive verbatim.
- Column order in `InsertSQL` is positional — a reordering is a silent data corruption bug
  that no compiler will catch.

## Success Criteria

- [x] DDL string byte-identical to `parser/src/schema.rs`
- [x] 22 columns in the exact Rust order; INSERT placeholder count matches
- [x] All 16 subject regexes compile and match the Rust patterns byte-for-byte
- [x] All 4 real configs load with values matching the scout's recorded table
- [x] All 4 tests from `config.rs:114-152` ported
- [x] Unknown-field YAML is **rejected** (test passes, via a valid YAML key)
- [x] `Columns` is nil for 2016 and populated for the other three
- [x] No column list duplicated outside `internal/schema`
- [x] SQLite driver chosen, pinned, and the rationale written into this file

## RESULT — 2026-08-13: **PASS**

`internal/schema` and `internal/config` ported; 13 Go tests green, all 63 Rust tests still green.

### Config format changed to YAML (user decision, mid-phase)

The user prefers `.yml`. Converting only the Go side would have left two hand-synced copies of
four configs, and any drift would surface as a *database* mismatch that Phase 6 would blame on
the parser. The user chose to convert **both** parsers, so they keep reading the identical file
and the parity gate stays intact.

This amends the plan's "Rust `parser/` untouched" decision — deliberately, and recorded here.
Changes: `serde_yaml 0.9` replaces `toml 0.8` in `parser/Cargo.toml`; `config.rs` uses
`serde_yaml::from_str`; `error.rs` wraps `serde_yaml::Error`; the four `config.rs` tests and
`golden.rs`'s config paths move to YAML; `build-db.js:53` reads `.yml`. `deny_unknown_fields`
is a serde attribute, so strictness carried over for free.

`serde_yaml` is deprecated upstream but stable, and this crate is deleted at Phase 7 cutover —
noted inline in `Cargo.toml`.

**Verified semantically identical, end to end.** Rebuilt two real datasets with the Rust parser
reading the new YAML configs and compared against `docs/data-pipeline.md`:

| dataset | source rows | skipped | DB rows | documented | match |
|---|---|---|---|---|---|
| `2016` | 877,464 | 3 duplicate SBDs collapsed | 877,461 | 877,461 | yes |
| `2017-old2` | 679,764 | 0 | 679,764 | 679,764 | yes |

Those two were chosen because they are the structurally distinct configs: 2016 is the only one
with `format_detection:` and no `columns:` mapping, and 2017-old2 is the only one combining
`strip_blank_rows: true` with `require_numeric_sbd: true`.

### Go decoder notes

- `gopkg.in/yaml.v3` with `KnownFields(true)` for strictness.
- `SheetMode` is validated **after** decoding: the decoder assigns named string types directly
  and never calls a custom unmarshaler, so `sheet_mode: second` would otherwise decode silently
  and read as "not all" downstream.
- The unknown-key test had to use a valid YAML key (`unexpected_key: 1`). Written TOML-style
  (`unexpected_key = 1`) it passes on a YAML *parse* error and proves nothing about
  `KnownFields`.

## Risk Assessment

| Risk | Mitigation |
|---|---|
| Go YAML lib silently ignores unknown keys | Explicit rejection test; `KnownFields(true)` required |
| Vietnamese regex literals corrupted by retyping | Copy verbatim; test compares against Rust source |
| Column order drift | Test asserts full ordered list, not just count |
