---
title: "Static Score Lookup - Excel to SQLite + Vite React Site"
description: "Replace Java/Hibernate pipeline with Node.js parser + client-side SQLite lookup site"
status: pending
priority: P1
effort: 6h
branch: main
tags: [node, vite, react, sql.js, sqlite, migration]
created: 2026-04-12
---

# Static Score Lookup

## Goal

Replace the Java-based Excel-to-SQLite pipeline with a Node.js script, and build a static Vite+React site that loads the .db file client-side via sql.js for readonly score queries.

## Architecture Overview

```
[~119 .xlsx files] --> [Node.js parser script] --> [thptqg2017.db]
                                                        |
                                          [Vite build copies to public/]
                                                        |
                                          [React app loads via sql.js]
                                                        |
                                          [User searches by name/ID]
```

### Data Flow

1. **Parse**: Node script reads all .xlsx from `src/main/resources/raw/` and `raw/(update)/`
2. **Transform**: Extract 4 columns per row; regex-parse scores from column 4 text
3. **Load**: Insert rows into SQLite via better-sqlite3 (Node-native, fast)
4. **Serve**: Vite copies .db to `public/`; React app fetches + initializes sql.js
5. **Query**: User input -> SQL WHERE -> render results table

### Database Schema

```sql
CREATE TABLE student (
  so_bao_danh TEXT PRIMARY KEY,
  ho_ten      TEXT NOT NULL,
  ngay_sinh   TEXT,          -- stored as dd/MM/yyyy string
  toan        REAL,
  ngu_van     REAL,
  vat_ly      REAL,
  hoa_hoc     REAL,
  sinh_hoc    REAL,
  khtn        REAL,
  lich_su     REAL,
  dia_ly      REAL,
  gdcd        REAL,
  khxh        REAL,
  tieng_anh   REAL
);

CREATE INDEX idx_ho_ten ON student(ho_ten);
```

**Change from Java version**: `ngay_sinh` stored as TEXT (not DATE) — simpler, no timezone issues, display-only field.

## Phases

| # | Phase | Status | Effort | Details |
|---|-------|--------|--------|---------|
| 1 | Project scaffolding | Pending | 30m | [phase-01](./phase-01-project-scaffolding.md) |
| 2 | Excel parser + DB builder | Pending | 2h | [phase-02](./phase-02-excel-parser-db-builder.md) |
| 3 | React static site | Pending | 2.5h | [phase-03](./phase-03-react-static-site.md) |
| 4 | Integration + deploy config | Pending | 1h | [phase-04](./phase-04-integration-deploy.md) |

## Dependencies

```
Phase 1 --> Phase 2 --> Phase 3 --> Phase 4
                  |                    ^
                  +--(produces .db)----+
```

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| .db file too large for browser fetch (~97MB) | High | High | Compress with gzip; sql.js supports ArrayBuffer; lazy-load on first query |
| Excel format variations (missing headers, bad cells) | Medium | Medium | try/catch per row (same as Java); log skipped rows; validate after build |
| sql.js WASM loading fails on some browsers | Low | Medium | Fallback error message; test Chrome/Firefox/Safari |
| Duplicate soBaoDanh across files (raw + update) | Medium | Low | Use INSERT OR REPLACE (update folder overwrites raw) |

## Backwards Compatibility

- Existing `database.sqlite` (97MB) preserved until new pipeline verified
- Java source code left intact — can be removed in future cleanup
- New .db file placed at `public/thptqg2017.db` (different name/location)

## Rollback Plan

- Phase 1-2: Delete `scripts/` and `package.json`; Java pipeline still works
- Phase 3-4: Delete `web/` folder; .db file from phase 2 still standalone-usable

## Success Criteria

1. `node scripts/build-database.js` produces valid .db with ~800K+ rows (matching Java output count)
2. `npm run dev` serves site; search by soBaoDanh returns correct student
3. `npm run build` produces static dist/ deployable to any static host
4. .db file size < 60MB (better-sqlite3 is more compact than Hibernate output)
