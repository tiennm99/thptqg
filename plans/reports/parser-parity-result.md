# Parser parity result

**Archived record.** This documents a verification run made on 2026-08-13,
against a tree that had a Rust parser, npm-script entry points and four
datasets — none of which exist now. It is kept because `docs/data-pipeline.md`
cites it as the evidence that the recovered foreign-language scores are real
rather than false regex matches. Do not expect the commands below to run.

Comparison of the databases the unified pipeline ships against a baseline built
from the pre-refactor code. Run on 2026-08-13 against the tree as it then stood
(`npm run build:rust && npm run build:db`), gate exit code 0.

Inputs:

- `plans/reports/parser-parity-baseline.json` — built with the two separate
  crates, before any change
- `plans/reports/parser-parity-current.json` — decompressed from
  `.build/public/db/*.db.gz`, i.e. the exact bytes published

## Result

| Dataset | Rows | Columns | Size |
| --- | --- | --- | --- |
| 2016 | 877,461 | 18 → 22 | +1.4% |
| 2017 | 861,068 | 18 → 22 | +2.2% |
| 2017-old | 847,348 | 18 → 22 | +2.2% |
| 2017-old2 | 679,764 | 18 → 22 | +2.2% |

Unchanged, for every dataset:

- row count
- non-NULL count for all 18 pre-existing columns
- every field of a deterministic student sample (SBDs ending `0000`)

## Approved recoveries

Unifying the subject regexes recovered 1,691 foreign-language scores that the
old per-year configs discarded. The 2016 config listed 12 subject patterns and
the 2017 configs 14; neither list was complete, so candidates who sat German,
Japanese or Russian ended up with no foreign-language score at all.

| Dataset | Recovered |
| --- | --- |
| 2016 | 182 × `tieng_nga` |
| 2017 | 93 × `tieng_duc`, 512 × `tieng_nhat` |
| 2017-old | 85 × `tieng_duc`, 484 × `tieng_nhat` |
| 2017-old2 | 22 × `tieng_duc`, 313 × `tieng_nhat` |

Confirmed real rather than spurious matches:

- Every student in all four datasets holds zero or exactly one foreign
  language — never two — so nothing is duplicated.
- Each affected student had all language columns NULL beforehand. SBD
  `01003198` went from all-NULL to `tieng_duc = 8`.
- Counts track dataset size across the three 2017 generations (93/85/22
  German, 512/484/313 Japanese).
- Values are ordinary exam scores in the 0–10 range.

These exact counts are pinned as `APPROVED_RECOVERY` in
`parser/scripts/verify-parity.js`. Any other newly-populated column, or drift
in these numbers, still fails the gate.

## Columns that must stay empty

Verified 0 non-NULL:

- 2016: `khtn`, `khxh`, `gdcd`
- 2017, 2017-old, 2017-old2: `ten_cum_thi`, `gioi_tinh`

## Reproducing

```sh
npm run build:rust && npm run build:db
node parser/scripts/db-stats.js \
  2016=<db> 2017=<db> 2017-old=<db> 2017-old2=<db> > current.json
node parser/scripts/verify-parity.js \
  plans/reports/parser-parity-baseline.json current.json
```

The baseline cannot be regenerated — it required the two pre-refactor crates,
which no longer exist. It is committed for that reason.

## Unresolved questions

None.
