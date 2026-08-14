# Project Overview

## Goal

A public lookup tool for Vietnam's National High School Graduation Exam scores,
running entirely in the browser and hosted for free on GitHub Pages. Covers the
2016 and 2017 exams — 1.7 million candidates across two datasets.

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

- **Zero backend.** The database (238–289 MB per dataset) stays on the server
  and the browser downloads it once and queries it in memory.
- **Read-only.** `INSERT`/`UPDATE`/`DELETE` are rejected, so nobody is misled
  into thinking edits persist. The file is fetched, never written.
- **Row caps.** 100 rows for lookups, 1000 for custom SQL, to prevent browser
  hangs.
- **Vietnamese-first UI.** App labels and data are Vietnamese; documentation is
  English.

## Datasets

| id | Exam | Candidates | Notes |
| --- | --- | --- | --- |
| `2016` | 2016 | 877,460 | 119 files, four column layouts |
| `2017` | 2017 | 861,068 | current generation of three publications |

Two further 2017 datasets (`2017-old`, `2017-old2`) were kept alongside these
because the three publications disagreed. They have been removed; git history
still has them.

Both datasets now have a crawler source. 2017 comes from the baotintuc.vn CDN,
which is still live. 2016 comes from an aggregator article on
`dtnt.bacninh.edu.vn`, also still online. Both datasets have been crawled
successfully, so either can be rebuilt from source — see
[data-pipeline](./data-pipeline.md#sources) for the full article URLs.

## History

Each year began as a standalone repository (`thptqg2016`, `thptqg2017`), merged
here with full history. They initially kept separate frontends and separate
copies of the same Rust parser, synchronised by hand. That duplication was
removed: there is now one frontend, one parser, and one canonical schema, with
per-dataset differences confined to one small config file and one registry
entry each.

The unification also fixed a latent data-loss bug — neither year's parser
config listed the complete set of subjects, so 1,691 candidates were missing
their foreign-language score. See `data-pipeline.md`.

## Status

Stable, data frozen. Work is limited to UX polish and keeping the pipeline
maintainable.
