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
| `internal/ingest` | dataset policy — sheet selection, header skipping, blank rows, the build loop, and the 2016 per-sheet format detection |
| `internal/transform` | `ToAscii`, score-regex parsing, row validation |
| `internal/schema` | the canonical 22-column table: DDL, INSERT, subject regexes |
| `internal/config` | per-dataset YAML parse rules |
| `internal/writer` | SQLite lifecycle and the stats block |
| `internal/audit` | source-vs-database SBD comparison |

The reader deliberately knows nothing about datasets: it reports every sheet and
every row verbatim. All policy lives in `ingest`. That split is what made the
reader independently verifiable against a hash oracle.

## Provenance

This is a port of a Rust crate that occupied this same path until the Go
implementation reached full parity, when it was built alongside as `go-parser/`
and moved back here once the Rust was removed. Source comments cite the original
by file and line (`parser/src/transform.rs:56` and similar) — those refer to the
Rust tree and resolve at the tag **`pre-go-parser-removal`**, the last commit
containing it.

The port was gated on a field-by-field comparison of both implementations across
the four datasets that existed then — 3,265,641 rows with identical full-table
SHA-256, identical per-column non-NULL counts, identical schema metadata and
identical build stdout. `assemble verify` is that comparator today and
still runs against any two sets of databases.

Behaviour was matched bug-for-bug, deliberately. Several quirks look like
defects and are load-bearing for the published data:

- a parsed score of `0` becomes NULL in the 2016 separate-scores layout,
  replicating a JavaScript falsy check;
- `ToAscii` strips combining marks in the literal range U+0300–U+036F rather
  than by Unicode category, which is narrower;
- gender is a two-value allowlist, and anything else becomes NULL;
- `diem_thi` is read untrimmed while the other three fields are trimmed;
- the `"SINH "` header token carries a trailing space.

Each has a test naming it, so none can be tidied away by accident.

## Verification

`testdata/reader-fidelity-hashes.tsv` holds a SHA-256 per input file over a
canonical dump of every cell of every sheet. It is **frozen**: it was produced
by the Rust reader, which no longer exists, so it cannot be regenerated. It
still fails if any single cell of any input file reads differently.

A mismatch names the file but not the cell. `cmd/dumpcells` prints the stream
the hash is taken over, so two runs can be diffed:

```bash
go -C parser run ./cmd/dumpcells ../data/2017/an-giang.xls out.tsv
```

The assembler refuses to publish a database whose row count does not match
the known figure, or whose artifact is under 90% of its usual size.
