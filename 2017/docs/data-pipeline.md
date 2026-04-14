# Data Pipeline

From raw Excel files to a compressed SQLite file the browser can load.

## Canonical source (data/)

63 Excel files from baotintuc.vn CDN. Source article:
`https://baotintuc.vn/tuyen-sinh/tra-cuu-diem-thi-thpt-2017-cua-63-tinh-thanh-pho-tren-baotintucvn-20170706073512672.htm`

Re-download anytime:

```bash
node scripts/crawl-baotintuc.js
```

Idempotent — skips files already present. Saves to `data/<ascii-kebab-province>.xls`.

## Reference sources (data-old/, data-old2/)

Both were collected from Vietnamese news sites around the time of the 2017 exam.
Specific publisher URLs were not recorded and cannot be recovered, so these
datasets are **not reproducible** — the files committed in git are the only
copy. They are kept for historical comparison and to expose the deploy matrix;
the main deployment always uses the reproducible `data/` source.

## Source Excel shape

Every file is a single sheet (for `data-old/`) or up to 3 sheets (for `data/` .xls) with columns:

| Col | Name | Content |
|---|---|---|
| 0 | HO_TEN     | full name in Vietnamese |
| 1 | NGAY_SINH  | `dd/mm/yyyy` |
| 2 | SOBAODANH  | 8-digit string, first 2 digits = province |
| 3 | DIEM_THI   | concatenated per-subject score string in the source language, e.g. `"Toan: 6.80 Van: 5.25 ... English: 5.80"` |

Some files lack a header row. `isHeaderRow()` in `build-lib.js` detects and skips.

## Score text parsing

`SCORE_PATTERNS` in `build-lib.js` defines one regex per subject. The patterns must match the exact subject labels the source files use (Vietnamese).

Subjects supported (DB column / label in source text):

| Column | Source label | English |
|---|---|---|
| toan        | Toán        | Math |
| ngu_van     | Ngữ văn     | Literature |
| vat_ly      | Vật lí      | Physics |
| hoa_hoc     | Hóa học     | Chemistry |
| sinh_hoc    | Sinh học    | Biology |
| khtn        | KHTN        | Natural Sciences combined |
| lich_su     | Lịch sử     | History |
| dia_ly      | Địa lí      | Geography |
| gdcd        | GDCD        | Civic Education |
| khxh        | KHXH        | Social Sciences combined |
| tieng_anh   | Tiếng Anh   | English |
| tieng_phap  | Tiếng Pháp  | French |
| tieng_nga   | Tiếng Nga   | Russian |
| tieng_trung | Tiếng Trung | Chinese |

Missing-in-string → `NULL` in DB (student didn't take that subject).

## Why three parsers

One shared lib + three thin parsers, not one parser with flags. Each dataset's quirks stay visible in its own file:

| File | Source dir | Sheet strategy | Extra guard |
|---|---|---|---|
| `build-database.js`      | `data/`      | iterate ALL sheets (.xls row overflow) | — |
| `build-database-old.js`  | `data-old/`  | sheet 0 only | reject rows where `so_bao_danh` isn't digits-only (rejects an `HO_TEN` → `SOBAODANH` header leak present in this export) |
| `build-database-old2.js` | `data-old2/` | iterate ALL sheets (HCM overflow) | skip fully blank rows before counting |

## Overflow-sheet gotcha

The old `.xls` binary format caps at 65,536 rows per sheet. Hanoi and Ho Chi Minh City exceed that in the baotintuc export, so Excel auto-splits the overflow into `Sheet2`. **Only reading `Sheet1` silently drops 13,720 students** (Hanoi +7,275, HCM +6,445).

The audit `node scripts/audit-row-counts.js` catches this by comparing source-row-count (across all sheets) to DB-row-count.

## Audits

Run any time after a rebuild:

```bash
node scripts/audit-row-counts.js    # source rows vs DB rows (zero-loss check)
node scripts/check-duplicates.js    # md5 + row-content duplicates in data/
node scripts/diff-datasets.js       # compare two DBs (needs backup/ dir)
```

Expected baseline:

| DB | Source rows | Skipped | DB rows |
|---|---|---|---|
| main  | 861,131 | 63 empty rows        | **861,068** |
| old   | 847,349 | 1 header-leak         | **847,348** |
| old2  | 679,764 | 0                    | **679,764** |

## Refreshing the data

```bash
rm data/*.xls                     # clear current snapshot
node scripts/crawl-baotintuc.js   # re-download from CDN
npm run build:db                  # rebuild
node scripts/audit-row-counts.js  # verify
gzip -kf -9 public/thptqg2017.db  # compress for shipping
```

## Adding a new source dataset

See `docs/deployment-guide.md` → "Adding a new variant".
