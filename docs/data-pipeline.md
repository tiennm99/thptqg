# Data Pipeline

From raw Excel files to the SQLite file the browser downloads and queries.

One Go binary (`parser/`) builds every dataset. What differs per dataset is
parse rules only — sheet strategy, column layout, validation guards — declared
in `parser/configs/<id>.yml`. The table shape, the INSERT and the subject
regexes are canonical and live in `parser/internal/schema/schema.go`.

## Sources

| id | Files | Origin | Host live? |
| --- | --- | --- | --- |
| `2016` | 4 `.xls` + 115 `.xlsx` | `dtnt.bacninh.edu.vn` aggregator article, 119 exam clusters | **yes** |
| `2017` | 63 `.xls` | `baotintuc.vn` article, files on its CDN | **yes** |

Full article URLs are below.

Crawling lives in `crawler/`, a separate Go module. It is never part of the
build — the source files are committed, so a crawl only refreshes them. Both
runs are idempotent: files already present are skipped.

```bash
go -C crawler run ./cmd/crawl 2016     # sources/source_2016.go
go -C crawler run ./cmd/crawl 2017     # sources/source_2017.go
go -C crawler run ./cmd/crawl 2017 --list   # list only, download nothing
```

**2017** comes from the article
`https://baotintuc.vn/tuyen-sinh/tra-cuu-diem-thi-thpt-2017-cua-63-tinh-thanh-pho-tren-baotintucvn-20170706073512672.htm`
and its CDN is still serving the files.

**2016** comes from the aggregator article
`https://dtnt.bacninh.edu.vn/tin-tuc/tin-tuc-su-kien/cong-bo-diem-thi-thptqg-2016-toan-bo-120-cum-thi-da-co-diem.html`,
which lists one spreadsheet per exam cluster. A full crawl against it has been
run successfully and reproduces `data/2016/`, so the dataset is recoverable from
source like 2017 is.

That host is **not reachable from every network**, though: it resolves to a
Vietnamese address that times out from at least some hosts abroad, in which case
the crawl stops with a connection error before downloading anything. That is a
connectivity problem, not a missing dataset — retry from a network that can
reach the host.

## How a source is defined

A source carries no link list. It names the article that published the files and
says how to name what it finds there:

| field | meaning |
| --- | --- |
| `Article` | the page to read links from |
| `Exts` | which file extensions to pick out of it |
| `WantFiles` | how many links to expect — fewer aborts the crawl |
| `Dest` | the local filename for one link |

`internal/article` does the fetching and HTML parsing; `internal/fetch` does the
downloading. Because `Article` is read at run time, `--list` needs network access
too.

`WantFiles` exists because a partial crawl is otherwise silent: parser will
build a short database from whatever files are present, and only the row-count
guard would notice, after the fact. A page that changes shape stops the crawl
instead.

### Local filenames are load-bearing

`parser` sorts its input files and inserts with `INSERT OR REPLACE`, which is
last-wins, so **filenames decide which row survives a duplicate exam number**. A
re-crawl that names files differently can produce a database with the same row
count and different content, which the assembler's row-count guard would
not catch.

Each source therefore pins its local names, and
`crawler/internal/sources/sources_test.go` checks every source against its
committed `data/<id>/` in both directions — every name it would write exists,
and every file present is accounted for.

2017 derives its names from the province name in the link text, transliterated
to ASCII (the CDN's own names are inconsistent:
`Angiang.xls`, `1BaRiaVungTau.xls`, `23HaiPhong.xls`). 2016 keeps the
server-assigned names verbatim, since it did not choose them.

## Source Excel shapes

2017 has one layout:

| Col | Name | Content |
| --- | --- | --- |
| 0 | HO_TEN | full name in Vietnamese |
| 1 | NGAY_SINH | `dd/mm/yyyy` |
| 2 | SOBAODANH | 8-digit string, first 2 digits = province |
| 3 | DIEM_THI | concatenated per-subject scores, e.g. `"Toán: 6.80 Ngữ văn: 5.25 …"` |

2016 has **four** layouts across its 119 files, chosen per sheet at runtime via
`format_detection: thptqg2016` in its config:

| Format | Detected by | Notes |
| --- | --- | --- |
| `separate-scores` | `row[0] == "SBD"` and `row[2] == "TOAN"` | one column per subject; scores read directly, no regex |
| `mapped` | header contains `SOBAODANH`/`SBD` plus `DIEM_THI` | column indices derived from the header |
| `subject-columns` | header names three or more subject columns | university-cluster files; identity and scores both resolved by header name |
| `default` | no recognised header | positional 6-column layout |

The header may sit below a title block, so the first five rows are searched for
it; rows above it are not data.

`subject-columns` covers two spellings. The ĐH Công nghiệp Thực phẩm file names
its columns `TO VA LI HO SI SU DI NN` with the language code in `Môn NN`; the
three ĐH Cần Thơ files publish `cdiem1..cdiem8`, numbered in the order the 2016
exam was sat — Toán, Ngoại ngữ, Ngữ văn, Vật lí, Địa lí, Hóa học, Lịch sử, Sinh
học — with the language code in `ngoaingu`. The language score is filed under
the subject its `N1`..`N6` code names.

The `mapped`, `default` and `subject-columns` layouts also carry `GIOI_TINH`
(and `mapped`/`default` `TEN_CUMTHI`), which is why only 2016 populates those
columns.

## Score text parsing

`SCORE_PATTERNS` in `parser/internal/schema/schema.go` defines one regex per subject, and
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

Running all 16 patterns everywhere recovered them: 182 × `tieng_nga` in 2016,
and 93 × `tieng_duc` plus 512 × `tieng_nhat` in 2017.

Four things established that these are real scores and not false regex matches,
checked when the change was made:

- Every student holds zero or exactly one foreign language, never two, so
  nothing was double-counted.
- Each affected student had **all** language columns NULL beforehand — e.g. SBD
  `01003198` went from all-NULL to `tieng_duc = 8`.
- The counts tracked dataset size across the three 2017 publications that then
  existed (93/85/22 German, 512/484/313 Japanese).
- Every recovered value is an ordinary exam score in the 0–10 range.

Row counts and the non-NULL counts of all 18 pre-existing columns were unchanged
across every dataset; only the four language columns gained values.

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

The databases are written with 4 KiB pages (`PRAGMA page_size` in
`parser/internal/writer/writer.go`) — SQLite's own default, set explicitly so
the published file does not change shape if that default ever moves. Nothing on
the client depends on the figure any more. It did under the previous design,
which read the file over HTTP a page at a time and had to be told the page
size; the browser now downloads the file whole.

The pragma runs before the DDL, because a page size cannot change once a table
exists.

| id | Source rows | Skipped | DB rows |
| --- | --- | --- | --- |
| `2016` | 877,460 | 0 | **877,460** |
| `2017` | 861,068 | 0 | **861,068** |

## Verifying a rebuild

The assembler verifies itself: each database's row count must match the
figure in the table above, and each `.sqlite3` must be at least 90% of its usual
size, or the build fails rather than publishing. That guard is the reason a
truncated dataset cannot reach the site with a green pipeline.

For a deeper check, `assemble verify` compares two sets of built databases
field by field — row counts, per-column non-NULL counts, schema metadata, and a
full-table SHA-256 over every row ordered by `so_bao_danh`:

```bash
cp -r .build/public/db /tmp/before      # keep the databases you have
go -C assembler run ./cmd/assemble db   # rebuild
go -C assembler run ./cmd/assemble verify /tmp/before .build/public/db
```

Each side is a directory of `<id>.sqlite3`; a database under an older name, or
a gzipped one, is still opened automatically. It exits non-zero on any mismatch,
names the first differing rows and columns, and fails rather than skipping when a
dataset is absent from either side — silently comparing one of two datasets is
how a gate passes without proving anything.

This is the only check on database *content*. The reader oracle below covers
reading the spreadsheets and the row-count guard covers how many rows came out,
but a change in transform or writer logic can alter what is in those rows while
both of those still pass.

`parser/internal/reader` additionally carries a frozen oracle of per-file
cell-dump hashes covering all 182 inputs; `go -C parser test ./...` fails if any single
cell of any input file reads differently.

## Refreshing the 2017 data

```bash
rm data/2017/*.xls
go -C crawler run ./cmd/crawl 2017
go -C assembler run ./cmd/assemble db 2017
```

The row-count guard in `build:db` confirms the rebuild matches the expected
total. That guard checks the count only, so if the crawl was expected to change
the data, compare content with `assemble verify` against a copy of the previous
database rather than trusting the count.

## Removed scripts

`check-duplicates.js`, `diff-datasets.js`, `db-stats.js` and `verify-parity.js`
were dropped with the Rust parser. The first two had been broken since before the
repo was unified (a hardcoded Windows path in one, an undeclared
`better-sqlite3` dependency in the other) and neither had any automated caller.
The latter two are superseded by `assemble verify`, which compares more and
cannot silently skip a dataset. `differential-parity.mjs`, which was that
comparator, has itself been folded into the assembler as
`assembler/internal/verify` — the pipeline is Go outside `web/`, and the port
also fixed a weakness: the JavaScript version hashed each row's fields joined
bare, so a value shifted across a column boundary produced the same digest.

`crawl-baotintuc.js` was not dropped but rewritten as the Go `crawler/` module,
producing the same local filenames. Two changes beyond the port:

- It carried its 63 links as a hardcoded array. The Go version reads them from
  the article instead, so the list cannot drift from what was published.
- Downloads land on a `.part` file and are renamed on completion. Writing
  straight to the destination left a truncated file after an interrupted run,
  and since the skip check only tests for a non-empty file, every later run
  would skip it — the corruption was permanent and silent.
