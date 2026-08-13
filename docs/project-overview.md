# Project Overview

## Goal

A public lookup tool for Vietnam's National High School Graduation Exam scores,
running entirely in the browser and hosted for free on GitHub Pages. Covers the
2016 and 2017 exams — 1.7 million candidates across four datasets.

## Scope

- Lookup by exam ID or full name, with Vietnamese diacritics handled
- Read-only SQL queries against a single `student` table
- Admission-block (khối thi) totals computed per candidate
- Static datasets — both exams are long over and the data is frozen

## Target users

- Former candidates checking their scores
- Education researchers and data journalists running aggregate statistics
- Developers exploring SQL against a real-world dataset

## Constraints

- **Zero backend.** The full database (38–48 MB gzipped per dataset) is
  downloaded to the browser and queried in-process.
- **Read-only.** `INSERT`/`UPDATE`/`DELETE` are rejected, so nobody is misled
  into thinking edits persist. `sql.js` is in-memory anyway.
- **Row caps.** 100 rows for lookups, 1000 for custom SQL, to prevent browser
  hangs.
- **Vietnamese-first UI.** App labels and data are Vietnamese; documentation is
  English.

## Datasets

| id | Exam | Candidates | Notes |
| --- | --- | --- | --- |
| `2016` | 2016 | 877,461 | 119 files, three column layouts |
| `2017` | 2017 | 861,068 | current generation, reproducible from source |
| `2017-old` | 2017 | 847,348 | pre-refresh archive |
| `2017-old2` | 2017 | 679,764 | corrected re-export |

The three 2017 datasets are kept side by side because they disagree, and the
disagreement is itself informative. Only `2017` is re-fetchable; the rest exist
solely as the copies committed here.

Original 2016 aggregator link
(<https://dtntbacgiang.edu.vn/tin-tuc/tin-tuc-su-kien/cong-bo-diem-thi-thptqg-2016-toan-bo-120-cum-thi-da-co-diem.html>)
is no longer accessible, which is why the raw files are mirrored in `data/2016/`.

## History

Each year began as a standalone repository (`thptqg2016`, `thptqg2017`), merged
here with full history. They initially kept separate frontends and separate
copies of the same Rust parser, synchronised by hand. That duplication was
removed: there is now one frontend, one parser, and one canonical schema, with
per-dataset differences confined to four small config files and one registry
entry each.

The unification also fixed a latent data-loss bug — neither year's parser
config listed the complete set of subjects, so 1,691 candidates were missing
their foreign-language score. See `data-pipeline.md`.

## Status

Stable, data frozen. Work is limited to UX polish and keeping the pipeline
maintainable.
