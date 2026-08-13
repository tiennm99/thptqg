# Brainstorm — Migrate parser from Rust to Go

Date: 2026-08-13. Branch: main. Scout input: `from-scout-to-brainstorm-260813-1502-rust-to-go-parser-migration-report.md`.

## Decision

Build a **new Go parser under `go-parser/`, side by side with the existing Rust `parser/`**. Validate by **differential comparison against live Rust output** — not a frozen baseline. Rust stays untouched and working throughout.

## Problem-first

User brought a preselected solution ("migrate to Go"). Inversion applied.

- **Driver**: preference for working in Go + AI makes iteration cheap. Not a defect in the Rust parser — none exists.
- **Evidence status**: none for a technical problem. Legitimate as a preference-driven migration; recorded as such rather than retrofitted with technical justification.
- **Rejected framings**: "one language too many" (Go is still a second non-JS toolchain — doesn't collapse the stack); "Rust hard to modify" (a Go port reproduces the same 2016-format complexity in different syntax); "performance" (419 MB of Excel I/O dominates deploy time, unchanged by language).

## Approaches evaluated

| # | Approach | Verdict |
|---|---|---|
| 1 | Don't migrate; drop dead `glob` dep | Rejected — user prefers Go, cost is acceptable |
| 2 | In-place Rust→Go rewrite of `parser/` | Rejected — no reversibility, broken half-states |
| 3 | Convert `.xls`→`.xlsx` first, then port | Held as **fallback** if `extrame/xls` fails |
| 4 | **Side-by-side `go-parser/` + differential validation** | **CHOSEN** |

Approach 4 beats the author's original phased proposal: keeping Rust live means ground truth is regenerable on demand, so no frozen parity baseline is needed. Reversibility is structural, not procedural.

## Constraints carried from scout

**Hard blocker to hit early**: 67 genuine OLE2/BIFF `.xls` files (verified by magic bytes `d0cf11e0a1b11ae1`) — 63 of them in `data/2017`, the 286 MB largest dataset. `calamine` reads BIFF+OOXML through one API; Go has no equivalent. `excelize` is xlsx-only; `extrame/xls` is the only real BIFF option and is lightly maintained + weak on non-UTF8 encodings, which matters because every cell is Vietnamese.

→ **Build the Go reader module FIRST**, before transform/writer. Fail fast. Fallback = approach 3 (data is frozen, git-tracked, untouched since the unification rename — a one-time conversion is legitimate here and would likely shrink the repo, since OOXML is zip-compressed and BIFF is not).

**Exactness traps that must be replicated, not improved:**
- `to_ascii`: literal codepoint range U+0300–U+036F filter, **not** `unicode.Mn` (Go's category check is more permissive → divergence). Plus explicit `đ/Đ → d` (NFD does not decompose them). `transform.rs:52-64`.
- TOML `deny_unknown_fields` is load-bearing and has a test. Most Go TOML libs ignore unknown keys by default.
- Tri-state skip reason (`BlankRow` vs `EmptyField` vs `NonNumericSbd`) drives the printed counters — a bool diverges.
- `0.0` score parsed as `None` in the 2016 separate-scores format (replicates a JS `||` falsy quirk). `format_detect_2016.rs:165`.
- Audit path reads **sheet 0 only**, deliberately ignoring `sheet_mode`. Preserve, don't "fix".
- Header check is **per-sheet**, not per-file.
- Unknown: how calamine stringifies date cells into `ngay_sinh`. Never inspected. Probe empirically against real files.

**Contracts to preserve**: CLI `build --schema --input --output`; SQLite `student` table, 22 cols, 3 indexes incl. partial `idx_ten_cum_thi`; non-zero exit kills the npm pipeline.

**Free wins**: regexes are already RE2-safe (zero port risk); SQLite is plain SQL text with positional `?`, no named params, no pragmas; `glob` dep is dead (confirmed zero references).

## Scope

Everything under `parser/` → `go-parser/`: crate, tests, and the 6 JS helper scripts.

**Sequencing constraint**: port `db-stats.js` / `verify-parity.js` **last**. They are the tools that prove the port is correct — rewriting them during the port is circular (a bug in the ported checker hides a bug in the ported parser). Verify Go versions against JS output before trusting them.

## Validation criteria

Go parser is done when, for all 4 datasets: Rust-built DB and Go-built DB are equivalent — row counts, per-column non-NULL counts, and field-by-field equality on a deterministic SBD sample. Plus black-box golden tests spawning the binary and inspecting the `.db`, including **an `.xls` fixture** (current suite has none, so it cannot catch a BIFF regression).

## Risks

| Risk | Severity | Mitigation |
|---|---|---|
| `extrame/xls` can't read the 67 BIFF files / mangles Vietnamese | **High** | Build reader first; fallback to `.xls`→`.xlsx` conversion |
| Date-cell stringification differs from calamine | Medium | Differential diff catches it; probe early |
| Silent value corruption across 419 MB | Medium | Differential gate is the only real defense — non-negotiable |
| Repo carries two parsers during migration | Low | Intentional; delete `parser/` only after sign-off |

## Next steps

1. `go-parser/` scaffold + reader module against real `.xls` — decisive gate.
2. Port config/transform/writer/audit/schema.
3. Black-box golden tests + `.xls` fixture.
4. Differential parity vs Rust across all 4 datasets.
5. CI swap (drop `dtolnay/rust-toolchain` + `Swatinem/rust-cache`, add `actions/setup-go`), `.gitignore`, docs.
6. Port JS scripts last. Remove `parser/` after sign-off.

## Unresolved

1. Keep binary name/path `parser/target/release/xlsxread` so `build-db.js` is untouched, or emit to `go-parser/bin/` and update the one constant at `build-db.js:25`?
2. Delete `parser/` after parity, or keep it as a reference implementation for some period?
3. `.xls` fallback: if conversion is needed, keep originals committed alongside converted files, or replace them?
