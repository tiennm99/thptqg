# Data Pipeline

From raw Excel files to a compressed SQLite file the browser can load.

One Go binary (`go-parser/`) builds every dataset. What differs per dataset is
parse rules only — sheet strategy, column layout, validation guards — declared
in `go-parser/configs/<id>.yml`. The table shape, the INSERT and the subject
regexes are canonical and live in `go-parser/internal/schema/schema.go`.

## Sources

| id | Files | Origin | Host live? |
| --- | --- | --- | --- |
| `2016` | 4 `.xls` + 115 `.xlsx` | aggregator article, 119 exam clusters | unconfirmed |
| `2017` | 63 `.xls` | baotintuc.vn CDN | **yes** |

Crawling lives in `crawler/`, a separate Go module. It is never part of the
build — the source files are committed, so a crawl only refreshes them. Both
runs are idempotent: files already present are skipped.

```bash
npm run crawl:2016              # crawler/internal/sources/source_2016.go
npm run crawl:2017              # crawler/internal/sources/source_2017.go
npm run crawl:2017 -- --list    # show what would be downloaded, fetch nothing
```

**2017** comes from the article
`https://baotintuc.vn/tuyen-sinh/tra-cuu-diem-thi-thpt-2017-cua-63-tinh-thanh-pho-tren-baotintucvn-20170706073512672.htm`
and its CDN is still serving the files.

**2016** comes from the aggregator article
`cong-bo-diem-thi-thptqg-2016-toan-bo-120-cum-thi-da-co-diem.html`. The site
that originally published it (`dtntbacgiang.edu.vn`) no longer resolves, so the
link list was read from the Internet Archive's copy of the page. Two caveats
that matter:

- **The list is verified; the host is not.** All 119 filenames match `data/2016/`
  exactly in both directions, which is asserted by a test. But the Archive
  captured only the article, not the spreadsheets, so the URLs themselves could
  not be confirmed by fetching. `baseURL` in `source_2016.go` points at a mirror
  of the same article that is still online; if it serves the files from a
  different directory, `uploadPath` is the one constant to change.
- **`data/2016/` is still the only confirmed copy.** Do not delete it on the
  assumption that a crawl can restore it.

### Local filenames are load-bearing

`go-parser` sorts its input files and inserts with `INSERT OR REPLACE`, which is
last-wins, so **filenames decide which row survives a duplicate exam number**. A
re-crawl that names files differently can produce a database with the same row
count and different content, which the row-count guard in `build-db.js` would
not catch.

Each source therefore pins its local names, and
`crawler/internal/sources/sources_test.go` checks every source against its
committed `data/<id>/` in both directions — every name it would write exists,
and every file present is accounted for.

2017 derives its names from the province (the CDN's own names are inconsistent:
`Angiang.xls`, `1BaRiaVungTau.xls`, `23HaiPhong.xls`). 2016 keeps the
server-assigned names verbatim, since it did not choose them.

## Source Excel shapes

The 2017 datasets share one layout:

| Col | Name | Content |
| --- | --- | --- |
| 0 | HO_TEN | full name in Vietnamese |
| 1 | NGAY_SINH | `dd/mm/yyyy` |
| 2 | SOBAODANH | 8-digit string, first 2 digits = province |
| 3 | DIEM_THI | concatenated per-subject scores, e.g. `"Toán: 6.80 Ngữ văn: 5.25 …"` |

2016 has **three** layouts across its 119 files, chosen per file at runtime via
`format_detection: thptqg2016` in its config:

| Format | Detected by | Notes |
| --- | --- | --- |
| `separate-scores` | `row[0] == "SBD"` and `row[2] == "TOAN"` | one column per subject; scores read directly, no regex |
| `mapped` | header contains `SOBAODANH`/`SBD` plus `DIEM_THI` | column indices derived from the header |
| `default` | no recognised header | positional 6-column layout |

The `mapped` and `default` layouts also carry `TEN_CUMTHI` and `GIOI_TINH`,
which is why only 2016 populates those columns.

## Score text parsing

`SCORE_PATTERNS` in `go-parser/internal/schema/schema.go` defines one regex per subject, and
**all 16 run against every dataset**. A subject a given exam year did not offer
simply never matches and stays NULL.

| Column | Source label | English | 2016 | 2017 |
| --- | --- | --- | --- | --- |
| toan | Toán | Math | ✓ | ✓ |
| ngu_van | Ngữ văn | Literature | ✓ | ✓ |
| vat_ly | Vật lí | Physics | ✓ | ✓ |
| hoa_hoc | Hóa học | Chemistry | ✓ | ✓ |
| sinh_hoc | Sinh học | Biology | ✓ | ✓ |
| khtn | KHTN | Natural Sciences combined | — | ✓ |
| lich_su | Lịch sử | History | ✓ | ✓ |
| dia_ly | Địa lí | Geography | ✓ | ✓ |
| gdcd | GDCD | Civic Education | — | ✓ |
| khxh | KHXH | Social Sciences combined | — | ✓ |
| tieng_anh | Tiếng Anh | English | ✓ | ✓ |
| tieng_phap | Tiếng Pháp | French | ✓ | ✓ |
| tieng_nga | Tiếng Nga | Russian | ✓ | ✓ |
| tieng_duc | Tiếng Đức | German | ✓ | ✓ |
| tieng_nhat | Tiếng Nhật | Japanese | ✓ | ✓ |
| tieng_trung | Tiếng Trung | Chinese | ✓ | ✓ |

### The foreign-language recovery

Before the parser was unified, the 2016 config listed 12 subject patterns and
the 2017 configs listed 14. Neither list was complete: candidates could sit
German, Japanese and Russian in both years, so 1,691 students ended up with **no
foreign-language score at all**.

Running all 16 patterns everywhere recovered them — 182 Russian in 2016, and
German and Japanese across all three 2017 generations. See
`plans/reports/parser-parity-result.md` for the evidence that these are real
scores rather than false matches.

## Per-dataset quirks

| id | Sheet strategy | Guards |
| --- | --- | --- |
| `2016` | all sheets | per-file format detection; header-token rows rejected |
| `2017` | all sheets — Hà Nội and HCM overflow | none |

### Overflow-sheet gotcha

The old `.xls` format caps at 65,536 rows per sheet. Hanoi and Ho Chi Minh City
exceed that, so Excel splits the overflow into `Sheet2`. **Reading only `Sheet1`
silently drops 13,720 students** (Hanoi +7,275, HCM +6,445). That is what
`sheet_mode = "all"` exists for.

## Expected row counts

| id | Source rows | Skipped | DB rows |
| --- | --- | --- | --- |
| `2016` | 877,464 | 3 duplicate SBDs collapsed | **877,461** |
| `2017` | 861,068 | 0 | **861,068** |

## Verifying a rebuild

`npm run build:db` verifies itself: each database's row count must match the
figure in the table above, and each `.db.gz` must be at least 90% of its usual
size, or the build fails rather than publishing. That guard is the reason a
truncated dataset cannot reach the site with a green pipeline.

For a deeper check, `go-parser/scripts/differential-parity.mjs` compares two sets
of databases field-by-field — row counts, per-column non-NULL counts, a
full-table SHA-256 over every row ordered by `so_bao_danh`, schema metadata, and
build stdout:

```bash
node go-parser/scripts/differential-parity.mjs \
  --rust /path/to/a-{id}.db --go /path/to/b-{id}.db
```

It exits non-zero on any mismatch and fails loudly if a dataset is missing rather
than skipping it. Written for the Rust-to-Go migration, it works for any two
builds. Uses the built-in `node:sqlite`, so it needs no dependencies.

`go-parser/internal/reader` additionally carries a frozen oracle of per-file
cell-dump hashes covering all 299 inputs; `npm run test:go` fails if any single
cell of any input file reads differently.

## Refreshing the 2017 data

```bash
rm data/2017/*.xls
npm run crawl:2017
npm run build:db 2017
```

The row-count guard in `build:db` confirms the rebuild matches the expected
total. That guard checks the count only, so if the crawl was expected to change
the data, compare content with `differential-parity.mjs` against a copy of the
previous database rather than trusting the count.

## Removed scripts

`check-duplicates.js`, `diff-datasets.js`, `db-stats.js` and `verify-parity.js`
were dropped with the Rust parser. The first two had been broken since before the
repo was unified (a hardcoded Windows path in one, an undeclared
`better-sqlite3` dependency in the other) and neither had any automated caller.
The latter two are superseded by `differential-parity.mjs`, which compares more
and cannot silently skip a dataset.

`crawl-baotintuc.js` was not dropped but rewritten in Go as
`crawler/internal/sources/source_2017.go`, keeping the same link list and the same
local filenames. The rewrite added a per-file timeout and made downloads land on
a `.part` file first — writing straight to the destination meant an interrupted
run left a truncated file that the skip check would then skip forever.
