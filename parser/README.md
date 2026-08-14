# parser

Reads the `.xls`/`.xlsx` source spreadsheets in `data/` and writes one SQLite
database per dataset.

```bash
go -C parser build -o bin/xlsxread ./cmd/xlsxread   # compile
go -C parser test ./...                             # unit tests + the reader-fidelity suite
```

```
xlsxread build --schema parser/configs/<id>.yml --input data/<id> --output <db>
xlsxread audit --schema parser/configs/<id>.yml --input data/<id> --db <db>
```

This stage only produces a database. Verifying it against the expected row
count, compressing it and publishing it belong to `assembler/`, which compiles
this binary and drives it per dataset:

```bash
go -C assembler run ./cmd/assemble db
```

## Layout

| path | role |
|---|---|
| `internal/reader` | spreadsheet reading; the only place that knows about file formats |
| `internal/ingest` | dataset policy — sheet selection, header skipping, blank rows, the build loop, and the 2016 per-sheet layout detection |
| `internal/transform` | `ToAscii`, score-regex parsing, row validation |
| `internal/schema` | the canonical 22-column table: DDL, INSERT, subject regexes |
| `internal/config` | per-dataset YAML parse rules |
| `internal/writer` | SQLite lifecycle and the stats block |
| `internal/audit` | source-vs-database SBD comparison |

The reader deliberately knows nothing about datasets: it reports every sheet and
every row verbatim. All policy lives in `ingest`. That split is what made the
reader independently verifiable against a hash oracle.

## Behaviour worth knowing

- `ToAscii` strips combining marks in the literal range U+0300–U+036F rather
  than by Unicode category. That covers every Vietnamese diacritic and must
  stay identical to `toAscii` in `web/src/lib/to-ascii.js`, or accent-insensitive
  search misses rows.
- Gender is normalised to `Nam`/`Nữ`; the Cần Thơ files write `0`/`1` instead
  and are translated. Anything else becomes NULL.
- A score of `0` is a real score — the candidate sat the paper and scored
  nothing — and is stored, not dropped.
- Birth dates are stored as `dd/mm/yyyy`. The Cần Thơ files' compact `ddmmyy`
  is expanded; the century is always 19xx, since a 2016 candidate born later
  would have sat the exam under age.

This code began as a port of a Rust crate that occupied the same path, and was
gated on a field-by-field comparison against it. That comparison is over:
correctness against the source spreadsheets decides behaviour now, not
agreement with the old implementation. `assemble verify` is the comparator
that gated it, and still compares any two sets of built databases.

## Verification

`testdata/reader-fidelity-hashes.tsv` holds a SHA-256 per input file over a
canonical dump of every cell of every sheet. It is **frozen**: the tool that
produced it no longer exists, so it cannot be regenerated. It still fails if
any single cell of any input file reads differently, which is what makes it
useful — cell rendering and sheet geometry are settled, and a change there is
a regression until proven otherwise. Dataset policy lives in `ingest`, so
fixing a layout never touches it.

A mismatch names the file but not the cell. `cmd/dumpcells` prints the stream
the hash is taken over, so two runs can be diffed:

```bash
go -C parser run ./cmd/dumpcells ../data/2017/an-giang.xls out.tsv
```

The assembler refuses to publish a database whose row count does not match
the known figure, or whose artifact is under 90% of its usual size.
