#!/usr/bin/env bash
# Regenerates the reader-fidelity oracle from the Rust/calamine ground truth.
#
# Emits SHA-256 per input file over the canonical cell dump. Only hashes are
# committed: the dumps are real student PII, and parser/tests/fixtures/README.md
# establishes that fixtures in this repo carry synthetic data only.
#
# Requires the Rust parser to still build. Run from the repo root.
set -euo pipefail

OUT=go-parser/testdata/reader-fidelity-hashes.tsv
CANON='BEGIN{OFS="\t"}
  $1=="FILE"{next}
  $1=="SHEETCOUNT"{print;next}
  $1=="SHEET"{print $1,$2,$3,$4,$5;next}
  $1=="ROW"{print;next}
  $1=="CELL"{print $1,$2,$3,$4,$7;next}'

{
  echo "# Canonical cell-dump SHA-256 per input file, produced by the Rust/calamine"
  echo "# ground truth (parser/examples/dump_cells.rs). The Go reader must reproduce"
  echo "# each hash exactly. Hashes only - the dumps themselves are real student PII"
  echo "# and are never committed, per parser/tests/fixtures/README.md."
  echo "# Regenerate: go-parser/scripts/regen-fidelity-hashes.sh"
} > "$OUT"

for f in data/2016/* data/2017/* data/2017-old/* data/2017-old2/*; do
  h=$(cargo run --release --quiet --manifest-path parser/Cargo.toml \
        --example dump_cells -- "$f" /dev/stdout \
      | awk -F'\t' "$CANON" | sha256sum | cut -d' ' -f1)
  printf '%s\t%s\n' "$f" "$h" >> "$OUT"
done

echo "wrote $(grep -cv '^#' "$OUT") hashes to $OUT"
