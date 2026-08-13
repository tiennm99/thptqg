---
title: "Unify frontend, standardize SQL schema, restructure repo"
description: "One 2017-based frontend, one canonical 22-column student schema, one parser crate, four datasets under data/"
status: completed
priority: P2
branch: "main"
tags: [refactor, schema, frontend, parser]
blockedBy: []
blocks: []
created: "2026-08-13T03:01:04.073Z"
createdBy: "ck:plan"
source: skill
---

# Unify frontend, standardize SQL schema, restructure repo

## Overview

Today the repo holds two near-duplicate projects (`2016/`, `2017/`), each with its
own React frontend, its own copy of the same Rust parser crate, and its own SQL
schema. The 2017 frontend is a strict feature superset; the 2016 parser is a
strict code superset. Both duplications are drift hazards, not real divergence.

Collapse to one of each:

- **One canonical schema** — 22-column `student` table (6 identity + 16 subject),
  defined once in `parser/src/schema.rs`. Absent columns bind NULL.
- **One parser crate** — `parser/`, four config files carrying only per-dataset
  parse rules.
- **One frontend** — repo root, built from `2017/src/`, plus the two identity
  columns 2016 renders today. The app owns every page GitHub Pages serves,
  including the `/thptqg/` hub, from a **single Vite build**.
- **Four datasets** — `data/2016`, `data/2017`, `data/2017-old`, `data/2017-old2`.

Published URLs are **flat**, one segment per dataset:

```text
/thptqg/            hub
/thptqg/2016/
/thptqg/2017/
/thptqg/2017-old/   was /thptqg/2017/old/
/thptqg/2017-old2/  was /thptqg/2017/old2/
```

The two old-generation URLs change. That is deliberate: the flat form makes the
URL segment **identical to the dataset ID**, which is already the name of the
data directory, the config file, and the DB file. One identifier end to end:

```text
data/2017-old/  →  parser/configs/2017-old.toml  →  db/2017-old.db.gz  →  /thptqg/2017-old/
```

Phase 3 keeps the two legacy paths working via redirect so existing links do not
break (see that phase; droppable if you don't care).

Package manager: **npm**. pnpm is dropped (see Phase 2).

## Target Layout

```text
/
├── index.html            Vite entry — serves every route
├── vite.config.js        ONE build, base: /thptqg/
├── package.json          npm; package-lock.json committed
├── src/
│   ├── datasets.js       runtime registry: title, source, examples, SQL presets
│   ├── router.js         pathname → dataset (or hub)
│   ├── components/
│   │   └── hub.jsx       the /thptqg/ landing route
│   ├── hooks/
│   └── lib/
├── data/
│   ├── 2016/  2017/  2017-old/  2017-old2/
├── parser/               single Rust crate (from 2016/tools/xlsxread)
│   ├── src/schema.rs     canonical DDL + INSERT + 16 subject regexes
│   ├── configs/*.toml    per-dataset parse rules only
│   ├── scripts/          crawl-baotintuc.js, diff-datasets.js, check-duplicates.js,
│   │                     verify-parity.js
│   └── tests/golden.rs
└── docs/                 merged from 2016/docs + 2017/docs
```

## Published Artifact

One bundle, five entry points. Because `base` is absolute (`/thptqg/`), the
emitted `index.html` references `/thptqg/assets/index-HASH.js` regardless of the
directory it sits in — so copying it to each dataset path yields a real static
file at every URL. GitHub Pages serves them as directory indexes. **No SPA
404-fallback hack is required**, which matters because the existing `?q=`
deep-link behaviour would not survive one.

```text
_site/
├── index.html              hub route
├── 404.html                copy of index.html
├── assets/index-HASH.js    ONE bundle, cached across all five pages
├── db/2016.db.gz  2017.db.gz  2017-old.db.gz  2017-old2.db.gz
├── 2016/index.html         ┐
├── 2017/index.html         │ byte-identical copies of the root index.html
├── 2017-old/index.html     │
├── 2017-old2/index.html    ┘
└── 2017/old/index.html     ┐ legacy redirect stubs (same file again)
    2017/old2/index.html    ┘
```

## Canonical Schema

```sql
CREATE TABLE student (
  so_bao_danh   TEXT PRIMARY KEY,
  ho_ten        TEXT NOT NULL,
  ho_ten_ascii  TEXT NOT NULL,
  ngay_sinh     TEXT,
  ten_cum_thi   TEXT,          -- 2016 only; NULL elsewhere
  gioi_tinh     TEXT,          -- 2016 only; NULL elsewhere
  toan REAL, ngu_van REAL, vat_ly REAL, hoa_hoc REAL, sinh_hoc REAL,
  khtn REAL,                   -- 2017 only
  lich_su REAL, dia_ly REAL,
  gdcd REAL, khxh REAL,        -- 2017 only
  tieng_anh REAL, tieng_phap REAL,
  tieng_nga REAL,              -- 2017 only
  tieng_duc REAL, tieng_nhat REAL,  -- 2016 only
  tieng_trung REAL
);
CREATE INDEX idx_ho_ten       ON student(ho_ten);
CREATE INDEX idx_ho_ten_ascii ON student(ho_ten_ascii);
CREATE INDEX idx_ten_cum_thi  ON student(ten_cum_thi) WHERE ten_cum_thi IS NOT NULL;
```

`idx_ten_cum_thi` is partial so it costs nothing on the three 2017 datasets
(zero entries) while staying fully useful for 2016's cluster grouping queries.

## Phases

| Phase | Name | Status |
|-------|------|--------|
| 1 | [Standard schema and unified parser](./phase-01-standard-schema-and-unified-parser.md) | Completed |
| 2 | [Repo restructure](./phase-02-repo-restructure.md) | Completed |
| 3 | [Unified frontend and dataset registry](./phase-03-unified-frontend-and-dataset-registry.md) | Completed |
| 4 | [Build and deploy pipeline](./phase-04-build-and-deploy-pipeline.md) | Completed |
| 5 | [Parity verification and docs](./phase-05-parity-verification-and-docs.md) | Completed |

## Dependencies

Strictly sequential. Phase 1 must produce parity-verified DBs under the old
layout before Phase 2 moves 419 MB of tracked data. Phase 5's parity gate is
the release gate — nothing merges until all four DBs match their pre-refactor
row counts and per-column non-NULL counts.

No cross-plan dependencies (`plans/` was empty before this plan).

## Acceptance Criteria

- [x] One Rust crate; `2016/tools/` and `2017/tools/` gone
- [x] One `src/`; `2016/src/` and `2017/src/` gone
- [x] One Vite build producing all five pages
- [x] All four DBs built from the same DDL, same INSERT, same 16 regexes
- [x] Per-dataset row count and per-column non-NULL count identical to pre-refactor baseline
- [x] All five published URLs functional under the flat scheme, deep links included
- [x] Legacy `/2017/old/` and `/2017/old2/` redirect to their flat equivalents, preserving `?q=`
- [x] 2016 site still shows `ten_cum_thi` + `gioi_tinh`; 2017 sites do not
- [x] 2016 site gains 2017's features (deep links, student detail, tiers, live search)
- [x] npm only: `package-lock.json` committed, no pnpm files or CI steps remain
- [x] `cargo test` green; `npm run lint` green

## Rollback

Every phase is a separate commit on a feature branch. Phase 2's `git mv` is the
only hard-to-undo step; it is content-preserving, so `git revert` restores the
old layout exactly. Do not squash before the Phase 5 gate passes.

## Open Questions

None outstanding. Four decisions were taken before planning:

1. Frontend lives at repo root.
2. Schema is defined in parser code, not duplicated across the TOML configs.
3. The hub page is a route inside the app, not a separate static file — one
   Vite build covers every page on GitHub Pages. Trade-off accepted: the hub
   now needs the JS bundle (~150 KB gzipped) to paint, where today it is 25
   lines of static HTML.
4. npm replaces pnpm.
5. URLs are flat (`/thptqg/2017-old/`), not nested (`/thptqg/2017/old/`), so the
   URL segment equals the dataset ID everywhere. Legacy paths redirect.
