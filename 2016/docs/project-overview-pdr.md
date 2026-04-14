# Project Overview — thptqg2016

## Goal

Provide a public lookup tool for Vietnam's 2016 National High School Graduation Exam scores (877,461 candidates), running entirely on the client, hosted for free on GitHub Pages.

## Scope

- Lookup by exam ID or full name (with Vietnamese diacritics handling)
- Read-only SQL queries against a single `student` table
- Static dataset — no updates (the 2016 exam is long over)

## Target users

- Former 2016 candidates checking their scores
- Education researchers / data journalists running aggregate stats
- Developers exploring SQL on a real-world dataset

## Constraints

- **Zero backend**: the full DB (tens of MB gzipped) is downloaded to the browser
- **Read-only**: INSERT/UPDATE/DELETE rejected to avoid the illusion that user edits persist
- **Row caps**: 100 rows (lookup), 1000 rows (SQL) to prevent browser hangs
- **Vietnamese-first UI**: app labels and data are Vietnamese; documentation is English

## Data sources

Excel files (`.xlsx`/`.xls`) collected from newspapers and exam clusters in 2016, stored in `data/`. One file per cluster, with several different column layouts (see `scripts/build-database.js`).

## Status

Stable. Data is frozen. Recent work focuses on UX polish (dark mode, accessibility, diacritics-insensitive search).
