# go-parser

Reads the `.xls`/`.xlsx` source spreadsheets in `data/` and writes one SQLite
database per dataset.

```bash
npm run build:go    # compile go-parser/bin/xlsxread
npm run build:db    # build + verify + gzip all four datasets
npm run test:go     # unit tests + the 299-file reader-fidelity suite
```

```
xlsxread build --schema go-parser/configs/<id>.yml --input data/<id> --output <db>
xlsxread audit --schema go-parser/configs/<id>.yml --input data/<id> --db <db>
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

This is a port of a Rust crate that lived at `parser/` until the Go
implementation reached full parity. Source comments cite the original by file
and line (`parser/src/transform.rs:56` and similar); those paths resolve at the
tag **`pre-go-parser-removal`**, the last commit containing the Rust code.

The port was gated on a field-by-field comparison of both implementations across
all four datasets — 3,265,641 rows with identical full-table SHA-256, identical
per-column non-NULL counts, identical schema metadata and identical build
stdout. `scripts/differential-parity.mjs` is that comparator and still runs
against any two sets of databases.

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
still fails if any single cell of any of the 299 files reads differently.

`npm run build:db` refuses to publish a database whose row count does not match
the known figure, or whose artifact is under 90% of its usual size.
